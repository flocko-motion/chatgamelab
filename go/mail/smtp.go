package mail

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/smtp"
	"slices"
	"strconv"
	"strings"
	"time"
)

// dialTimeout bounds connecting to the mail server.
const dialTimeout = 15 * time.Second

// sessionTimeout bounds the whole SMTP conversation: Microsoft answers failed
// logins very slowly.
const sessionTimeout = 2 * time.Minute

// smtpConfig is the mail server and its login.
type smtpConfig struct {
	host, port, user, password, from string
}

// defaultPort is the submission port, used while a server names none.
const defaultPort = 587

func (s Server) config() smtpConfig {
	port := s.Port
	if port == 0 {
		port = defaultPort
	}
	from := strings.TrimSpace(s.From)
	if from == "" {
		from = strings.TrimSpace(s.User)
	}
	return smtpConfig{
		host:     strings.TrimSpace(s.Host),
		port:     strconv.Itoa(port),
		user:     s.User,
		password: s.Password,
		from:     from,
	}
}

// sendSMTP delivers msg through the server cfg names: implicit TLS on port
// 465, otherwise STARTTLS when the server offers it; logging in when cfg has a
// user. Errors name the step that failed.
func sendSMTP(cfg smtpConfig, msg Message) error {
	data, err := buildMessage(cfg.from, msg)
	if err != nil {
		return err
	}
	addr := net.JoinHostPort(cfg.host, cfg.port)
	tlsConfig := &tls.Config{ServerName: cfg.host}

	var conn net.Conn
	if cfg.port == "465" {
		conn, err = tls.DialWithDialer(&net.Dialer{Timeout: dialTimeout}, "tcp", addr, tlsConfig)
	} else {
		conn, err = net.DialTimeout("tcp", addr, dialTimeout)
	}
	if err != nil {
		return fmt.Errorf("connecting to %s: %w", addr, err)
	}
	_ = conn.SetDeadline(time.Now().Add(sessionTimeout))
	c, err := smtp.NewClient(conn, cfg.host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("greeting from %s: %w", addr, err)
	}
	defer c.Close()

	if cfg.port != "465" {
		if ok, _ := c.Extension("STARTTLS"); ok {
			if err := c.StartTLS(tlsConfig); err != nil {
				return fmt.Errorf("STARTTLS: %w", err)
			}
		}
	}
	if cfg.user != "" {
		_, mechanisms := c.Extension("AUTH")
		auth, err := chooseAuth(mechanisms, cfg)
		if err != nil {
			return err
		}
		if err := c.Auth(auth); err != nil {
			return fmt.Errorf("login as %s: %w", cfg.user, err)
		}
	}
	return deliver(c, envelopeAddress(cfg.from), envelopeAddress(msg.To), data)
}

// buildMessage renders msg as RFC 5322 headers and a quoted-printable UTF-8
// body.
func buildMessage(from string, msg Message) ([]byte, error) {
	var body bytes.Buffer
	qp := quotedprintable.NewWriter(&body)
	if _, err := qp.Write([]byte(strings.ReplaceAll(msg.Body, "\n", "\r\n"))); err != nil {
		return nil, err
	}
	if err := qp.Close(); err != nil {
		return nil, err
	}

	id := make([]byte, 12)
	_, _ = rand.Read(id)
	domain := "chatgamelab.eu"
	if at := strings.LastIndex(from, "@"); at >= 0 {
		domain = strings.Trim(from[at+1:], "> ")
	}

	var b bytes.Buffer
	fmt.Fprintf(&b, "From: %s\r\n", from)
	fmt.Fprintf(&b, "To: %s\r\n", msg.To)
	fmt.Fprintf(&b, "Subject: %s\r\n", mime.QEncoding.Encode("utf-8", msg.Subject))
	fmt.Fprintf(&b, "Date: %s\r\n", msg.SentAt.Format(time.RFC1123Z))
	fmt.Fprintf(&b, "Message-ID: <%s@%s>\r\n", hex.EncodeToString(id), domain)
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
	b.Write(body.Bytes())
	return b.Bytes(), nil
}

// envelopeAddress is the bare address of "Name <address>".
func envelopeAddress(s string) string {
	if i := strings.LastIndex(s, "<"); i >= 0 {
		return strings.Trim(s[i+1:], "> ")
	}
	return strings.TrimSpace(s)
}

// chooseAuth is PLAIN or LOGIN, whichever the server offers, PLAIN first.
// Microsoft 365 offers only LOGIN (and XOAUTH2).
func chooseAuth(mechanisms string, cfg smtpConfig) (smtp.Auth, error) {
	offered := strings.Fields(strings.ToUpper(mechanisms))
	switch {
	case slices.Contains(offered, "PLAIN"):
		return smtp.PlainAuth("", cfg.user, cfg.password, cfg.host), nil
	case slices.Contains(offered, "LOGIN"):
		return &loginAuth{user: cfg.user, password: cfg.password, host: cfg.host}, nil
	default:
		return nil, fmt.Errorf("mail server %s offers no supported login method (offered: %q, supported: PLAIN, LOGIN)", cfg.host, mechanisms)
	}
}

// loginAuth is SMTP's AUTH LOGIN, which net/smtp lacks.
type loginAuth struct {
	user, password, host string
}

// Start refuses, like smtp.PlainAuth, to send the password unencrypted to
// any host but localhost.
func (a *loginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	if !server.TLS && !isLocalhost(server.Name) {
		return "", nil, fmt.Errorf("refusing AUTH LOGIN without TLS to %s", server.Name)
	}
	if server.Name != a.host {
		return "", nil, fmt.Errorf("wrong host name %q, expected %q", server.Name, a.host)
	}
	return "LOGIN", nil, nil
}

// Next answers the server's prompts for user name and password.
func (a *loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if !more {
		return nil, nil
	}
	prompt := strings.ToLower(strings.TrimSpace(string(fromServer)))
	switch {
	case strings.HasPrefix(prompt, "username"):
		return []byte(a.user), nil
	case strings.HasPrefix(prompt, "password"):
		return []byte(a.password), nil
	default:
		return nil, fmt.Errorf("unexpected AUTH LOGIN prompt from server: %q", string(fromServer))
	}
}

func isLocalhost(name string) bool {
	return name == "localhost" || name == "127.0.0.1" || name == "::1"
}

// deliver sends data from one sender to one recipient and ends the session.
func deliver(c *smtp.Client, from, to string, data []byte) error {
	if err := c.Mail(from); err != nil {
		return fmt.Errorf("sender %s: %w", from, err)
	}
	if err := c.Rcpt(to); err != nil {
		return fmt.Errorf("recipient %s: %w", to, err)
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("DATA: %w", err)
	}
	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("writing message: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("message not accepted: %w", err)
	}
	return c.Quit()
}
