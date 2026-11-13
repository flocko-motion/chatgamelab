// package: mail / outgoing e-mail
// type:    infrastructure
// job:     sends plain-text e-mails via SMTP (configured by environment; login via PLAIN or LOGIN, the latter for Microsoft 365) or, without SMTP_HOST, writes them to the log; in dev mode sent mails are also kept in memory so tests can read confirmation links.
// limits:  no templates for specific mails (-> callers), no queue or retry.
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
	"strings"
	"sync"
	"time"

	"cgl/functional"
	"cgl/log"
)

// DefaultNotifyAddress receives a copy of every organisation registration confirmation.
const DefaultNotifyAddress = "chatgamelab@jff.de"

// recentLimit is how many sent mails dev mode keeps in memory.
const recentLimit = 50

// Message is one plain-text e-mail.
type Message struct {
	To      string    `json:"to"`
	Subject string    `json:"subject"`
	Body    string    `json:"body"`
	SentAt  time.Time `json:"sentAt"`
}

var (
	mu        sync.Mutex
	recording bool
	recent    []Message
)

// EnableRecording keeps sent mails in memory (dev mode only).
func EnableRecording() {
	mu.Lock()
	defer mu.Unlock()
	recording = true
}

// Recent returns the mails sent since start, newest first (empty unless recording).
func Recent() []Message {
	mu.Lock()
	defer mu.Unlock()
	out := make([]Message, len(recent))
	for i, m := range recent {
		out[len(recent)-1-i] = m
	}
	return out
}

// NotifyAddress is where copies of registration confirmations go (ORG_REGISTRATION_NOTIFY_EMAIL).
func NotifyAddress() string {
	return strings.TrimSpace(functional.EnvOrDefault("ORG_REGISTRATION_NOTIFY_EMAIL", DefaultNotifyAddress))
}

// PublicURL is the app's public base URL without trailing slash (PUBLIC_URL), used for links in mails.
func PublicURL() string {
	return strings.TrimRight(functional.EnvOrDefault("PUBLIC_URL", "http://localhost"), "/")
}

// Configured reports whether a mail server is set (SMTP_HOST); without one, mails are only logged.
func Configured() bool {
	return smtpConfigFromEnv().host != ""
}

// Send delivers one message. Without SMTP_HOST the mail is only logged.
func Send(msg Message) error {
	msg.SentAt = time.Now().UTC()
	record(msg)

	cfg := smtpConfigFromEnv()
	if cfg.host == "" {
		log.Info("mail not sent (SMTP_HOST empty), logging instead",
			"to", msg.To, "subject", msg.Subject, "body", msg.Body)
		return nil
	}
	if err := sendSMTP(cfg, msg); err != nil {
		log.Error("sending mail failed", "to", msg.To, "subject", msg.Subject, "error", err)
		return err
	}
	log.Info("mail sent", "to", msg.To, "subject", msg.Subject)
	return nil
}

func record(msg Message) {
	mu.Lock()
	defer mu.Unlock()
	if !recording {
		return
	}
	recent = append(recent, msg)
	if len(recent) > recentLimit {
		recent = recent[len(recent)-recentLimit:]
	}
}

type smtpConfig struct {
	host, port, user, password, from string
}

func smtpConfigFromEnv() smtpConfig {
	return smtpConfig{
		host:     strings.TrimSpace(functional.EnvOrDefault("SMTP_HOST", "")),
		port:     strings.TrimSpace(functional.EnvOrDefault("SMTP_PORT", "587")),
		user:     functional.EnvOrDefault("SMTP_USER", ""),
		password: functional.EnvOrDefault("SMTP_PASSWORD", ""),
		from:     strings.TrimSpace(functional.EnvOrDefault("SMTP_FROM", "ChatGameLab <noreply@chatgamelab.eu>")),
	}
}

// buildMessage renders RFC 5322 headers plus a quoted-printable UTF-8 body.
func buildMessage(from string, msg Message) ([]byte, error) {
	var body bytes.Buffer
	qp := quotedprintable.NewWriter(&body)
	if _, err := qp.Write([]byte(strings.ReplaceAll(msg.Body, "\n", "\r\n"))); err != nil {
		return nil, err
	}
	if err := qp.Close(); err != nil {
		return nil, err
	}

	idBytes := make([]byte, 12)
	_, _ = rand.Read(idBytes)
	domain := "chatgamelab.eu"
	if at := strings.LastIndex(from, "@"); at >= 0 {
		domain = strings.Trim(from[at+1:], "> ")
	}

	var b bytes.Buffer
	fmt.Fprintf(&b, "From: %s\r\n", from)
	fmt.Fprintf(&b, "To: %s\r\n", msg.To)
	fmt.Fprintf(&b, "Subject: %s\r\n", mime.QEncoding.Encode("utf-8", msg.Subject))
	fmt.Fprintf(&b, "Date: %s\r\n", msg.SentAt.Format(time.RFC1123Z))
	fmt.Fprintf(&b, "Message-ID: <%s@%s>\r\n", hex.EncodeToString(idBytes), domain)
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
	b.Write(body.Bytes())
	return b.Bytes(), nil
}

// envelopeAddress extracts the bare address from "Name <addr>".
func envelopeAddress(s string) string {
	if i := strings.LastIndex(s, "<"); i >= 0 {
		return strings.Trim(s[i+1:], "> ")
	}
	return strings.TrimSpace(s)
}

func sendSMTP(cfg smtpConfig, msg Message) error {
	data, err := buildMessage(cfg.from, msg)
	if err != nil {
		return err
	}
	addr := net.JoinHostPort(cfg.host, cfg.port)
	tlsConfig := &tls.Config{ServerName: cfg.host}

	// Port 465 uses implicit TLS; everything else uses STARTTLS when the server offers it.
	var conn net.Conn
	if cfg.port == "465" {
		conn, err = tls.DialWithDialer(&net.Dialer{Timeout: dialTimeout}, "tcp", addr, tlsConfig)
	} else {
		conn, err = net.DialTimeout("tcp", addr, dialTimeout)
	}
	if err != nil {
		return fmt.Errorf("connecting to %s: %w", addr, err)
	}
	// Microsoft delays answers to failed logins considerably; never hang forever.
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

// dialTimeout bounds how long connecting to the mail server may take.
const dialTimeout = 15 * time.Second

// sessionTimeout bounds the whole SMTP conversation.
const sessionTimeout = 2 * time.Minute

// chooseAuth picks PLAIN or LOGIN, whichever the server offers (Microsoft 365 offers only LOGIN and XOAUTH2).
func chooseAuth(mechanisms string, cfg smtpConfig) (smtp.Auth, error) {
	offered := strings.Fields(strings.ToUpper(mechanisms))
	has := func(name string) bool {
		for _, m := range offered {
			if m == name {
				return true
			}
		}
		return false
	}
	switch {
	case has("PLAIN"):
		return smtp.PlainAuth("", cfg.user, cfg.password, cfg.host), nil
	case has("LOGIN"):
		return &loginAuth{user: cfg.user, password: cfg.password, host: cfg.host}, nil
	default:
		return nil, fmt.Errorf("mail server %s offers no supported login method (offered: %q, supported: PLAIN, LOGIN)", cfg.host, mechanisms)
	}
}

// loginAuth implements the SMTP "AUTH LOGIN" mechanism, which net/smtp lacks.
type loginAuth struct {
	user, password, host string
}

func (a *loginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	// Like smtp.PlainAuth: never send the password unencrypted, except to localhost.
	if !server.TLS && !isLocalhost(server.Name) {
		return "", nil, fmt.Errorf("refusing AUTH LOGIN without TLS to %s", server.Name)
	}
	if server.Name != a.host {
		return "", nil, fmt.Errorf("wrong host name %q, expected %q", server.Name, a.host)
	}
	return "LOGIN", nil, nil
}

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
