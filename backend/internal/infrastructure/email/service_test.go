package email

import (
	"bufio"
	"context"
	"errors"
	"io"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"strings"
	"testing"
	"time"
)

// fakeSMTP accepts one connection; rcptCode decides the RCPT reply.
func fakeSMTP(t *testing.T, rcptCode string) (addr string, got chan string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	got = make(chan string, 1)
	go func() {
		conn, err := ln.Accept()
		ln.Close()
		if err != nil {
			return
		}
		defer conn.Close()
		r := bufio.NewReader(conn)
		write := func(s string) { _, _ = conn.Write([]byte(s + "\r\n")) }
		write("220 fake")
		var data strings.Builder
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				got <- data.String()
				return
			}
			cmd := strings.ToUpper(strings.TrimSpace(line))
			switch {
			case strings.HasPrefix(cmd, "EHLO"):
				write("250-fake\r\n250 OK")
			case strings.HasPrefix(cmd, "RCPT"):
				write(rcptCode)
			case cmd == "DATA":
				write("354 go")
				for {
					l, _ := r.ReadString('\n')
					if l == ".\r\n" {
						break
					}
					data.WriteString(l)
				}
				write("250 queued")
			case cmd == "QUIT":
				write("221 bye")
				got <- data.String()
				return
			default:
				write("250 OK")
			}
		}
	}()
	return ln.Addr().String(), got
}

func newTestService(addr string) *Service {
	host, port, _ := net.SplitHostPort(addr)
	return NewServiceWithConfig(Config{Host: host, Port: port, TLSMode: "none", From: "orders@inspire.test", FromName: "Inspire", Timeout: 5 * time.Second})
}

func TestSendDeliversMultipartWithEncodedSubject(t *testing.T) {
	addr, got := fakeSMTP(t, "250 OK")
	msg := Transactional("আপনার অর্ডার নিশ্চিত হয়েছে", "Hi Rafi,", []string{"Book <b>shipped</b> & on its way."}, &Action{Label: "View order", URL: "https://app.test/o/1"}, "footer")
	msg.To = "rafi@example.com"
	if err := newTestService(addr).Send(context.Background(), msg); err != nil {
		t.Fatal(err)
	}
	raw := <-got
	parsed, err := mail.ReadMessage(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("not a valid message: %v\n%s", err, raw)
	}
	subject, _ := new(mime.WordDecoder).DecodeHeader(parsed.Header.Get("Subject"))
	if subject != "আপনার অর্ডার নিশ্চিত হয়েছে" {
		t.Fatalf("subject round-trip failed: %q", subject)
	}
	for _, h := range []string{"Date", "Message-ID", "From"} {
		if parsed.Header.Get(h) == "" {
			t.Fatalf("missing %s header", h)
		}
	}
	if !strings.Contains(parsed.Header.Get("Content-Type"), "multipart/alternative") {
		t.Fatalf("expected multipart/alternative, got %q", parsed.Header.Get("Content-Type"))
	}
	decoded, _ := io.ReadAll(quotedprintable.NewReader(strings.NewReader(raw)))
	if !strings.Contains(string(decoded), "&lt;b&gt;shipped&lt;/b&gt;") {
		t.Fatal("HTML in paragraphs must be escaped in the HTML part")
	}
}

func TestRefusedRecipientIsPermanent(t *testing.T) {
	addr, _ := fakeSMTP(t, "550 no such user")
	err := newTestService(addr).SendEmail(context.Background(), "ghost@example.com", "s", "b")
	var re RecipientError
	if !errors.As(err, &re) {
		t.Fatalf("expected RecipientError, got %v", err)
	}
}

func TestNotConfiguredIsAnError(t *testing.T) {
	err := NewService("", "", "", "", "no-reply@example.com").SendEmail(context.Background(), "a@example.com", "s", "b")
	if !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured (it used to pretend success), got %v", err)
	}
}

func TestStartTLSRequiredWhenExpected(t *testing.T) {
	addr, _ := fakeSMTP(t, "250 OK")
	host, port, _ := net.SplitHostPort(addr)
	s := NewServiceWithConfig(Config{Host: host, Port: port, From: "a@inspire.test", Timeout: 3 * time.Second})
	if err := s.SendEmail(context.Background(), "b@example.com", "s", "b"); err == nil || !strings.Contains(err.Error(), "STARTTLS") {
		t.Fatalf("must refuse to send in clear text when STARTTLS was expected, got %v", err)
	}
}

func TestHeaderInjectionStripped(t *testing.T) {
	addr, got := fakeSMTP(t, "250 OK")
	if err := newTestService(addr).SendEmail(context.Background(), "a@example.com", "Hi\r\nBcc: victim@example.com", "b"); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(<-got, "\r\nBcc:") {
		t.Fatal("subject newline allowed header injection")
	}
}

func TestVerifyConnectsWithoutSending(t *testing.T) {
	addr, got := fakeSMTP(t, "250 OK")
	if err := newTestService(addr).Verify(context.Background()); err != nil {
		t.Fatal(err)
	}
	if data := <-got; data != "" {
		t.Fatalf("Verify must not send a message, got %q", data)
	}
	if err := NewService("", "", "", "", "a@b.test").Verify(context.Background()); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured, got %v", err)
	}
}
