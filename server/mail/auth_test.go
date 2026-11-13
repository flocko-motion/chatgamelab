package mail

import (
	"bufio"
	"encoding/base64"
	"net"
	"net/smtp"
	"strings"
	"testing"
	"time"
)

func TestChooseAuthPrefersPlainAndFallsBackToLogin(t *testing.T) {
	cfg := smtpConfig{host: "smtp.example.org", user: "u", password: "p"}
	if a, err := chooseAuth("LOGIN PLAIN", cfg); err != nil {
		t.Fatal(err)
	} else if _, isLogin := a.(*loginAuth); isLogin {
		t.Error("PLAIN offered but LOGIN chosen")
	}
	// Microsoft 365 offers exactly this.
	if a, err := chooseAuth("LOGIN XOAUTH2", cfg); err != nil {
		t.Fatal(err)
	} else if _, isLogin := a.(*loginAuth); !isLogin {
		t.Errorf("expected LOGIN, got %T", a)
	}
	if _, err := chooseAuth("XOAUTH2", cfg); err == nil {
		t.Error("expected error when neither PLAIN nor LOGIN is offered")
	}
}

func TestLoginAuthRefusesUnencrypted(t *testing.T) {
	a := &loginAuth{user: "u", password: "p", host: "smtp.example.org"}
	if _, _, err := a.Start(&smtp.ServerInfo{Name: "smtp.example.org", TLS: false}); err == nil {
		t.Error("LOGIN without TLS to a remote host must be refused")
	}
	if mech, _, err := a.Start(&smtp.ServerInfo{Name: "smtp.example.org", TLS: true}); err != nil || mech != "LOGIN" {
		t.Errorf("got %q, %v", mech, err)
	}
}

// TestSendSMTPWithLoginAuth talks to a fake server that, like Microsoft 365, offers only AUTH LOGIN.
func TestSendSMTPWithLoginAuth(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	type result struct {
		user, password, mailFrom, rcptTo, data string
		err                                    string
	}
	done := make(chan result, 1)
	go func() {
		var r result
		defer func() { done <- r }()
		conn, err := ln.Accept()
		if err != nil {
			r.err = err.Error()
			return
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
		rd := bufio.NewReader(conn)
		write := func(s string) { _, _ = conn.Write([]byte(s + "\r\n")) }
		read := func() string { l, _ := rd.ReadString('\n'); return strings.TrimRight(l, "\r\n") }
		decode := func(s string) string { b, _ := base64.StdEncoding.DecodeString(s); return string(b) }

		write("220 fake ESMTP")
		for {
			line := read()
			cmd := strings.ToUpper(line)
			switch {
			case strings.HasPrefix(cmd, "EHLO"):
				write("250-fake")
				write("250 AUTH LOGIN XOAUTH2")
			case cmd == "AUTH LOGIN":
				write("334 " + base64.StdEncoding.EncodeToString([]byte("Username:")))
				r.user = decode(read())
				write("334 " + base64.StdEncoding.EncodeToString([]byte("Password:")))
				r.password = decode(read())
				write("235 2.7.0 Authentication successful")
			case strings.HasPrefix(cmd, "AUTH"):
				r.err = "unexpected " + line
				write("504 5.7.4 Unrecognized authentication type")
			case strings.HasPrefix(cmd, "MAIL FROM:"):
				r.mailFrom = line[len("MAIL FROM:"):]
				write("250 OK")
			case strings.HasPrefix(cmd, "RCPT TO:"):
				r.rcptTo = line[len("RCPT TO:"):]
				write("250 OK")
			case cmd == "DATA":
				write("354 go ahead")
				var b strings.Builder
				for l := read(); l != "."; l = read() {
					b.WriteString(l + "\n")
				}
				r.data = b.String()
				write("250 queued")
			case cmd == "QUIT":
				write("221 bye")
				return
			default:
				r.err = "unexpected " + line
				return
			}
		}
	}()

	_, port, _ := net.SplitHostPort(ln.Addr().String())
	cfg := smtpConfig{host: "127.0.0.1", port: port, user: "ulrich@example.org", password: "geheim",
		from: "Chatgamelab <chatgamelab@example.org>"}
	if err := sendSMTP(cfg, Message{To: "a@example.org", Subject: "Test", Body: "Hallo", SentAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	r := <-done
	if r.err != "" {
		t.Fatal(r.err)
	}
	if r.user != "ulrich@example.org" || r.password != "geheim" {
		t.Errorf("login sent %q / %q", r.user, r.password)
	}
	if r.mailFrom != "<chatgamelab@example.org>" || r.rcptTo != "<a@example.org>" {
		t.Errorf("envelope %q -> %q", r.mailFrom, r.rcptTo)
	}
	if !strings.Contains(r.data, "From: Chatgamelab <chatgamelab@example.org>") {
		t.Errorf("From header missing:\n%s", r.data)
	}
}
