// package: mail / outgoing e-mail
// type:    infrastructure
// job:     sends plain-text mails through the SMTP server the environment names, or logs them without one; in development mode it keeps the mails sent, so tests can read them
// limits:  no texts of particular mails (-> their callers), no queue, no retry
package mail

import (
	"strings"
	"sync"
	"time"

	"cgl/functional"
	"cgl/log"
)

// The environment configures sending:
//
//	SMTP_HOST      the mail server; empty: mails are only logged
//	SMTP_PORT      587 (STARTTLS when offered) by default; 465: implicit TLS
//	SMTP_USER      login, PLAIN or LOGIN, whichever the server offers
//	SMTP_PASSWORD
//	SMTP_FROM      "ChatGameLab <noreply@chatgamelab.eu>" by default
//	ORG_REGISTRATION_NOTIFY_EMAIL  receives a copy of every organisation's consent

// keptLimit is how many sent mails development mode keeps.
const keptLimit = 50

// Message is one plain-text mail.
type Message struct {
	To      string    `json:"to"`
	Subject string    `json:"subject"`
	Body    string    `json:"body"`
	SentAt  time.Time `json:"sentAt"`
}

var (
	mu      sync.Mutex
	keeping bool
	kept    []Message
)

// KeepSent keeps the mails sent from now on, for Sent (development mode only).
func KeepSent() {
	mu.Lock()
	defer mu.Unlock()
	keeping = true
}

// Sent are the mails kept since KeepSent, newest first.
func Sent() []Message {
	mu.Lock()
	defer mu.Unlock()
	out := make([]Message, len(kept))
	for i, m := range kept {
		out[len(kept)-1-i] = m
	}
	return out
}

// NotifyAddress is where copies of organisations' consents go:
// ORG_REGISTRATION_NOTIFY_EMAIL, or else fallback, the site's support address.
func NotifyAddress(fallback string) string {
	return strings.TrimSpace(functional.EnvOrDefault("ORG_REGISTRATION_NOTIFY_EMAIL", fallback))
}

// Send delivers msg; without SMTP_HOST it only logs it. A failure is logged
// and returned.
func Send(msg Message) error {
	msg.SentAt = time.Now().UTC()
	keep(msg)

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

// keep holds msg for Sent while keeping, dropping the oldest beyond keptLimit.
func keep(msg Message) {
	mu.Lock()
	defer mu.Unlock()
	if !keeping {
		return
	}
	kept = append(kept, msg)
	if len(kept) > keptLimit {
		kept = kept[len(kept)-keptLimit:]
	}
}
