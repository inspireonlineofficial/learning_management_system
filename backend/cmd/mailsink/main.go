// Command mailsink is a development mail server: it accepts every message
// sent to it over SMTP and shows them in a web inbox, so the LMS can send
// real emails locally without any reaching real people.
//
//	go run ./cmd/mailsink                 # SMTP 127.0.0.1:1025, web http://127.0.0.1:8025
//	SMTP_HOST=127.0.0.1 SMTP_PORT=1025 SMTP_TLS=none  (backend settings)
//
// It is NOT a production mail server: it never relays mail onward. In
// production point SMTP_* at a real relay (see docs/email.md).
//
// Web API (used by the test scripts):
//
//	GET    /api/messages          newest first: id, from, to, subject, received_at
//	GET    /api/messages/{id}     the message with text and html bodies
//	DELETE /api/messages          clear the inbox
package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/http"
	"net/mail"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const maxMessageBytes = 10 << 20

type message struct {
	ID         int       `json:"id"`
	From       string    `json:"from"`
	To         []string  `json:"to"`
	Subject    string    `json:"subject"`
	ReceivedAt time.Time `json:"received_at"`
	Text       string    `json:"text,omitempty"`
	HTML       string    `json:"html,omitempty"`
	Raw        string    `json:"-"`
}

type store struct {
	mu     sync.Mutex
	next   int
	keep   int
	msgs   []*message
	dir    string
	logger *log.Logger
}

func (s *store) add(m *message) {
	s.mu.Lock()
	s.next++
	m.ID = s.next
	s.msgs = append(s.msgs, m)
	if len(s.msgs) > s.keep {
		s.msgs = s.msgs[len(s.msgs)-s.keep:]
	}
	s.mu.Unlock()
	if s.dir != "" {
		name := filepath.Join(s.dir, fmt.Sprintf("%s-%06d.eml", m.ReceivedAt.Format("20060102-150405"), m.ID))
		if err := os.WriteFile(name, []byte(m.Raw), 0o600); err != nil {
			s.logger.Printf("save %s: %v", name, err)
		}
	}
	s.logger.Printf("received #%d %q for %s", m.ID, m.Subject, strings.Join(m.To, ", "))
}

func (s *store) list() []*message {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*message, len(s.msgs))
	for i := range s.msgs {
		out[len(s.msgs)-1-i] = s.msgs[i]
	}
	return out
}

func (s *store) get(id int) *message {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, m := range s.msgs {
		if m.ID == id {
			return m
		}
	}
	return nil
}

func (s *store) clear() {
	s.mu.Lock()
	s.msgs = nil
	s.mu.Unlock()
}

func main() {
	smtpAddr := flag.String("smtp", envOr("MAILSINK_SMTP", "127.0.0.1:1025"), "SMTP listen address")
	httpAddr := flag.String("http", envOr("MAILSINK_HTTP", "127.0.0.1:8025"), "web inbox listen address")
	keep := flag.Int("keep", 1000, "messages kept in memory")
	dir := flag.String("dir", os.Getenv("MAILSINK_DIR"), "optional folder to also save .eml files")
	flag.Parse()

	logger := log.New(os.Stdout, "mailsink ", log.LstdFlags)
	st := &store{keep: *keep, dir: *dir, logger: logger}
	if st.dir != "" {
		if err := os.MkdirAll(st.dir, 0o700); err != nil {
			logger.Fatalf("create %s: %v", st.dir, err)
		}
	}

	ln, err := net.Listen("tcp", *smtpAddr)
	if err != nil {
		logger.Fatalf("listen SMTP: %v", err)
	}
	logger.Printf("SMTP on %s, inbox on http://%s", *smtpAddr, *httpAddr)
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				logger.Printf("accept: %v", err)
				continue
			}
			go serveSMTP(conn, st, logger)
		}
	}()

	srv := &http.Server{Addr: *httpAddr, Handler: routes(st), ReadHeaderTimeout: 10 * time.Second}
	logger.Fatal(srv.ListenAndServe())
}

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

// ---- SMTP ----------------------------------------------------------------

func serveSMTP(conn net.Conn, st *store, logger *log.Logger) {
	defer conn.Close()
	r := bufio.NewReader(io.LimitReader(conn, maxMessageBytes*2))
	w := bufio.NewWriter(conn)
	reply := func(line string) {
		_, _ = w.WriteString(line + "\r\n")
		_ = w.Flush()
	}
	_ = conn.SetDeadline(time.Now().Add(5 * time.Minute))
	reply("220 mailsink ESMTP (development only, nothing is relayed)")

	var from string
	var to []string
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")
		verb := strings.ToUpper(line)
		switch {
		case strings.HasPrefix(verb, "EHLO"):
			_, _ = w.WriteString("250-mailsink\r\n250-SIZE " + strconv.Itoa(maxMessageBytes) + "\r\n250-8BITMIME\r\n250-AUTH PLAIN LOGIN\r\n250 OK\r\n")
			_ = w.Flush()
		case strings.HasPrefix(verb, "HELO"):
			reply("250 mailsink")
		case strings.HasPrefix(verb, "AUTH"):
			// Any credentials are accepted: this server only captures mail.
			if strings.HasPrefix(verb, "AUTH LOGIN") {
				reply("334 VXNlcm5hbWU6")
				_, _ = r.ReadString('\n')
				reply("334 UGFzc3dvcmQ6")
				_, _ = r.ReadString('\n')
			} else if strings.TrimSpace(verb) == "AUTH PLAIN" {
				reply("334 ")
				_, _ = r.ReadString('\n')
			}
			reply("235 Authentication accepted")
		case strings.HasPrefix(verb, "MAIL FROM:"):
			from = strings.Trim(strings.TrimSpace(line[len("MAIL FROM:"):]), "<>")
			if i := strings.Index(from, "> "); i >= 0 {
				from = from[:i]
			}
			to = nil
			reply("250 OK")
		case strings.HasPrefix(verb, "RCPT TO:"):
			rcpt := strings.Trim(strings.TrimSpace(line[len("RCPT TO:"):]), "<>")
			to = append(to, rcpt)
			reply("250 OK")
		case verb == "DATA":
			if len(to) == 0 {
				reply("503 need RCPT first")
				continue
			}
			reply("354 End data with <CR><LF>.<CR><LF>")
			raw, err := readData(r)
			if err != nil {
				reply("552 " + err.Error())
				return
			}
			st.add(parse(from, to, raw))
			reply("250 OK: queued")
			from, to = "", nil
		case verb == "RSET":
			from, to = "", nil
			reply("250 OK")
		case verb == "NOOP":
			reply("250 OK")
		case verb == "QUIT":
			reply("221 Bye")
			return
		case strings.HasPrefix(verb, "STARTTLS"):
			reply("454 TLS not available (use SMTP_TLS=none with mailsink)")
		default:
			reply("502 command not implemented")
		}
	}
}

func readData(r *bufio.Reader) ([]byte, error) {
	var buf bytes.Buffer
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		if line == ".\r\n" || line == ".\n" {
			return buf.Bytes(), nil
		}
		if strings.HasPrefix(line, "..") {
			line = line[1:]
		}
		if buf.Len()+len(line) > maxMessageBytes {
			return nil, errors.New("message too large")
		}
		buf.WriteString(line)
	}
}

func parse(from string, to []string, raw []byte) *message {
	m := &message{From: from, To: to, ReceivedAt: time.Now(), Raw: string(raw)}
	msg, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		m.Text = string(raw)
		return m
	}
	dec := new(mime.WordDecoder)
	if subject, err := dec.DecodeHeader(msg.Header.Get("Subject")); err == nil {
		m.Subject = subject
	} else {
		m.Subject = msg.Header.Get("Subject")
	}
	if f := msg.Header.Get("From"); f != "" {
		m.From = f
	}
	collectParts(msg.Header.Get("Content-Type"), msg.Header.Get("Content-Transfer-Encoding"), msg.Body, m)
	return m
}

func collectParts(ctype, encoding string, body io.Reader, m *message) {
	mediaType, params, err := mime.ParseMediaType(ctype)
	if err != nil {
		mediaType = "text/plain"
	}
	if strings.HasPrefix(mediaType, "multipart/") {
		mr := multipart.NewReader(body, params["boundary"])
		for {
			p, err := mr.NextRawPart()
			if err != nil {
				return
			}
			collectParts(p.Header.Get("Content-Type"), p.Header.Get("Content-Transfer-Encoding"), p, m)
		}
	}
	var reader io.Reader = body
	switch strings.ToLower(encoding) {
	case "quoted-printable":
		reader = quotedprintable.NewReader(body)
	case "base64":
		reader = base64.NewDecoder(base64.StdEncoding, body)
	}
	data, _ := io.ReadAll(io.LimitReader(reader, maxMessageBytes))
	switch mediaType {
	case "text/html":
		m.HTML = string(data)
	case "text/plain":
		if m.Text == "" {
			m.Text = string(data)
		}
	}
}

// ---- Web inbox -------------------------------------------------------------

func routes(st *store) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/messages", func(w http.ResponseWriter, r *http.Request) {
		type summary struct {
			ID         int       `json:"id"`
			From       string    `json:"from"`
			To         []string  `json:"to"`
			Subject    string    `json:"subject"`
			ReceivedAt time.Time `json:"received_at"`
		}
		out := []summary{}
		for _, m := range st.list() {
			out = append(out, summary{m.ID, m.From, m.To, m.Subject, m.ReceivedAt})
		}
		writeJSON(w, out)
	})
	mux.HandleFunc("GET /api/messages/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.Atoi(r.PathValue("id"))
		m := st.get(id)
		if m == nil {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, m)
	})
	mux.HandleFunc("DELETE /api/messages", func(w http.ResponseWriter, r *http.Request) {
		st.clear()
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("GET /messages/{id}/html", func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.Atoi(r.PathValue("id"))
		m := st.get(id)
		if m == nil || m.HTML == "" {
			http.NotFound(w, r)
			return
		}
		// Shown inside a sandboxed iframe; scripts and remote loads blocked.
		w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; img-src data:; sandbox")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, m.HTML)
	})
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		var selected *message
		if id, err := strconv.Atoi(r.URL.Query().Get("id")); err == nil {
			selected = st.get(id)
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = inboxPage.Execute(w, struct {
			Messages []*message
			Selected *message
		}{st.list(), selected})
	})
	return mux
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

var inboxPage = template.Must(template.New("inbox").Parse(`<!DOCTYPE html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>mailsink — {{len .Messages}} messages</title>
<style>
body{margin:0;font:14px/1.4 system-ui,sans-serif;color:#1f2a44;background:#f5f3ee}
header{padding:12px 20px;background:#1f2a44;color:#fff;display:flex;justify-content:space-between;align-items:center}
main{display:grid;grid-template-columns:minmax(260px,380px) 1fr;height:calc(100vh - 48px)}
ul{list-style:none;margin:0;padding:0;overflow:auto;border-right:1px solid #ddd;background:#fff}
li a{display:block;padding:10px 16px;border-bottom:1px solid #eee;color:inherit;text-decoration:none}
li a:hover,li a[aria-current]{background:#eef0f5}
small{color:#555}
section{overflow:auto;padding:16px 20px}
iframe{width:100%;height:70vh;border:1px solid #ddd;background:#fff}
pre{white-space:pre-wrap;background:#fff;border:1px solid #ddd;padding:12px}
button{background:#fff;border:0;padding:6px 10px;cursor:pointer}
@media (max-width:700px){main{grid-template-columns:1fr;height:auto}}
</style></head><body>
<header><strong>mailsink</strong><span>{{len .Messages}} messages · development only, nothing is delivered
<button onclick="fetch('/api/messages',{method:'DELETE'}).then(()=>location='/')">Clear</button></span></header>
<main>
<ul>{{range .Messages}}<li><a href="/?id={{.ID}}"{{if $.Selected}}{{if eq $.Selected.ID .ID}} aria-current="true"{{end}}{{end}}>
<strong>{{.Subject}}</strong><br><small>To {{range .To}}{{.}} {{end}}· {{.ReceivedAt.Format "15:04:05"}}</small></a></li>
{{else}}<li style="padding:16px">No messages yet.</li>{{end}}</ul>
<section>{{with .Selected}}
<h2>{{.Subject}}</h2><p><small>From {{.From}} · To {{range .To}}{{.}} {{end}} · {{.ReceivedAt.Format "2006-01-02 15:04:05"}}</small></p>
{{if .HTML}}<iframe sandbox src="/messages/{{.ID}}/html" title="HTML version"></iframe>{{end}}
<h3>Text version</h3><pre>{{.Text}}</pre>
{{else}}<p>Select a message.</p>{{end}}</section>
</main></body></html>`))
