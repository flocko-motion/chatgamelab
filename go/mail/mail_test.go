package mail

import (
	"strings"
	"testing"
	"time"
)

func TestBuildMessageEncodesUmlauts(t *testing.T) {
	raw, err := buildMessage("ChatGameLab <noreply@example.org>", Message{
		To: "a@example.org", Subject: "Bestätigung", Body: "Grüße\nZeile 2", SentAt: time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	if !strings.Contains(s, "Subject: =?utf-8?q?Best=C3=A4tigung?=") {
		t.Errorf("subject not Q-encoded:\n%s", s)
	}
	if !strings.Contains(s, "Gr=C3=BC=C3=9Fe\r\nZeile 2") {
		t.Errorf("body not quoted-printable:\n%s", s)
	}
}

func TestSendWithoutSMTPKeeps(t *testing.T) {
	KeepSent()
	if err := Send(Server{}, Message{To: "x@example.org", Subject: "s", Body: "b"}); err != nil {
		t.Fatal(err)
	}
	if got := Sent(); len(got) == 0 || got[0].To != "x@example.org" {
		t.Fatalf("mail not kept: %+v", got)
	}
}

func TestEnvelopeAddress(t *testing.T) {
	if got := envelopeAddress("Name <a@b.de>"); got != "a@b.de" {
		t.Fatal(got)
	}
	if got := envelopeAddress(" a@b.de "); got != "a@b.de" {
		t.Fatal(got)
	}
}

func TestServerConfigDefaults(t *testing.T) {
	cfg := Server{Host: " smtp.example.org ", User: "me@example.org"}.config()
	if cfg.host != "smtp.example.org" || cfg.port != "587" || cfg.from != "me@example.org" {
		t.Errorf("config = %+v; want the host trimmed, port 587, the user as sender", cfg)
	}
	cfg = Server{Host: "h", Port: 465, User: "u", From: "Site <noreply@example.org>"}.config()
	if cfg.port != "465" || cfg.from != "Site <noreply@example.org>" {
		t.Errorf("config = %+v; want port 465 and the sender set", cfg)
	}
}
