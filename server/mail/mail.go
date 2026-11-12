// package: mail / outgoing e-mail
// type:    infrastructure
// job:     sends plain-text e-mails via SMTP (configured by environment) or, without SMTP_HOST, writes them to the log; in dev mode sent mails are also kept in memory so tests can read confirmation links.
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

	var auth smtp.Auth
	if cfg.user != "" {
		auth = smtp.PlainAuth("", cfg.user, cfg.password, cfg.host)
	}

	// Port 465 uses implicit TLS; everything else uses STARTTLS when the server offers it.
	if cfg.port == "465" {
		conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: cfg.host})
		if err != nil {
			return err
		}
		c, err := smtp.NewClient(conn, cfg.host)
		if err != nil {
			return err
		}
		defer c.Close()
		if auth != nil {
			if err := c.Auth(auth); err != nil {
				return err
			}
		}
		return deliver(c, envelopeAddress(cfg.from), envelopeAddress(msg.To), data)
	}
	return smtp.SendMail(addr, auth, envelopeAddress(cfg.from), []string{envelopeAddress(msg.To)}, data)
}

func deliver(c *smtp.Client, from, to string, data []byte) error {
	if err := c.Mail(from); err != nil {
		return err
	}
	if err := c.Rcpt(to); err != nil {
		return err
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(data); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return c.Quit()
}
