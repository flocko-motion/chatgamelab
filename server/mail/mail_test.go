package mail

import (
	"strings"
	"testing"
	"time"
)

func TestBuildMessageEncodesUmlauts(t *testing.T) {
	raw, err := buildMessage("ChatGameLab <noreply@chatgamelab.eu>", Message{
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

func TestSendWithoutSMTPRecords(t *testing.T) {
	t.Setenv("SMTP_HOST", "")
	EnableRecording()
	if err := Send(Message{To: "x@example.org", Subject: "s", Body: "b"}); err != nil {
		t.Fatal(err)
	}
	if got := Recent(); len(got) == 0 || got[0].To != "x@example.org" {
		t.Fatalf("mail not recorded: %+v", got)
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
