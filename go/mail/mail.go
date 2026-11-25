// package: mail / outgoing e-mail
// type:    infrastructure
// job:     sends plain-text mails through the SMTP server its caller names (the admin's mail settings), or logs them without one; in development mode it keeps the mails sent, so tests can read them
// limits:  no texts of particular mails (-> their callers), no queue, no retry
package mail

import (
	"sync"
	"time"

	"cgl/log"
)

// Server is the mail server a mail goes through.
type Server struct {
	Host     string // empty: mails are only logged
	Port     int    // 465: implicit TLS; any other: STARTTLS when offered
	User     string // login, PLAIN or LOGIN, whichever the server offers; empty: none
	Password string
	From     string // the sender, e.g. "Name <noreply@example.org>"; empty: the user
}

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

// Send delivers msg through server; without a host it only logs it. A
// failure is logged and returned.
func Send(server Server, msg Message) error {
	msg.SentAt = time.Now().UTC()
	keep(msg)

	cfg := server.config()
	if cfg.host == "" {
		log.Info("mail not sent (no mail server set), logging instead",
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
