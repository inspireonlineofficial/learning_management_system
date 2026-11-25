// Command emailcheck verifies the email setup with the same SMTP_* settings
// the server uses, and checks the sender domain's DNS records that decide
// whether mail reaches the inbox or spam.
//
//	go run ./cmd/emailcheck                       # connect, TLS, login; DNS checks
//	go run ./cmd/emailcheck -to you@example.com   # also send a test email
//	go run ./cmd/emailcheck -dkim-selector s1     # check a specific DKIM selector
//
// It never prints the SMTP password.
package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/mail"
	"os"
	"strings"
	"time"

	"lms-backend/internal/infrastructure/email"
)

var problems, warnings int

func pass(f string, a ...interface{}) { fmt.Printf("  PASS  "+f+"\n", a...) }
func warn(f string, a ...interface{}) { warnings++; fmt.Printf("  WARN  "+f+"\n", a...) }
func fail(f string, a ...interface{}) { problems++; fmt.Printf("  FAIL  "+f+"\n", a...) }

func main() {
	to := flag.String("to", "", "send a test email to this address")
	selector := flag.String("dkim-selector", "", "DKIM selector to check (otherwise common selectors are tried)")
	flag.Parse()

	svc := email.NewServiceWithConfig(email.Config{
		Host: os.Getenv("SMTP_HOST"), Port: os.Getenv("SMTP_PORT"), TLSMode: os.Getenv("SMTP_TLS"),
		Username: os.Getenv("SMTP_USERNAME"), Password: os.Getenv("SMTP_PASSWORD"),
		From: os.Getenv("SMTP_FROM"), FromName: os.Getenv("SMTP_FROM_NAME"),
	})
	fmt.Println("Email setup check")
	fmt.Println("\nSMTP")
	if !svc.Configured() {
		fail("SMTP_HOST and SMTP_FROM must be set (see docs/email.md)")
		finish()
	}
	pass("configured: %s", svc.Describe())
	if strings.EqualFold(os.Getenv("SMTP_TLS"), "none") && !isLocal(os.Getenv("SMTP_HOST")) {
		fail("SMTP_TLS=none sends mail and credentials unencrypted; use it only with the local mailsink")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	if err := svc.Verify(ctx); err != nil {
		fail("connect / TLS / login: %v", err)
	} else {
		steps := []string{"connected"}
		if !strings.EqualFold(os.Getenv("SMTP_TLS"), "none") {
			steps = append(steps, "TLS negotiated")
		}
		if os.Getenv("SMTP_USERNAME") != "" {
			steps = append(steps, "logged in")
		}
		pass("%s", strings.Join(steps, ", "))
	}

	from, err := mail.ParseAddress(os.Getenv("SMTP_FROM"))
	if err != nil {
		fail("SMTP_FROM is not a valid address: %v", err)
		finish()
	}
	domain := from.Address[strings.LastIndex(from.Address, "@")+1:]
	fmt.Printf("\nDNS for the sender domain %s\n", domain)
	if isLocalDomain(domain) {
		warn("%s is not a public domain; DNS checks skipped (fine for local development only)", domain)
	} else {
		checkDNS(domain, *selector)
	}

	if *to != "" {
		fmt.Println("\nTest email")
		msg := email.Transactional("Inspire email test", "Hi,",
			[]string{"This is a test email from the Inspire LMS email check. If you can read this, delivery works.",
				"Check whether it landed in the inbox or in spam; spam usually means SPF/DKIM/DMARC need fixing."}, nil,
			"Sent by cmd/emailcheck.")
		msg.To = *to
		if err := svc.Send(ctx, msg); err != nil {
			fail("send to %s: %v", *to, err)
		} else {
			pass("sent to %s — check the inbox (and the spam folder)", *to)
		}
	}
	finish()
}

func checkDNS(domain, selector string) {
	txt, err := net.LookupTXT(domain)
	switch {
	case err != nil:
		fail("SPF: could not read TXT records: %v", err)
	default:
		spf := ""
		for _, t := range txt {
			if strings.HasPrefix(strings.ToLower(t), "v=spf1") {
				if spf != "" {
					fail("SPF: more than one SPF record (only one is allowed)")
				}
				spf = t
			}
		}
		switch {
		case spf == "":
			fail("SPF: no \"v=spf1\" TXT record; add the one your provider gives you")
		case strings.Contains(spf, "+all"):
			fail("SPF: \"+all\" lets anyone send as you: %s", spf)
		default:
			pass("SPF: %s", spf)
		}
	}

	if dm, err := net.LookupTXT("_dmarc." + domain); err != nil || len(dm) == 0 {
		warn("DMARC: no record at _dmarc.%s (add at least \"v=DMARC1; p=none; rua=mailto:...\")", domain)
	} else {
		found := false
		for _, t := range dm {
			if strings.HasPrefix(strings.ToUpper(t), "V=DMARC1") {
				found = true
				pass("DMARC: %s", t)
			}
		}
		if !found {
			warn("DMARC: _dmarc.%s exists but has no v=DMARC1 record", domain)
		}
	}

	selectors := []string{selector}
	if selector == "" {
		selectors = []string{"default", "s1", "s2", "k1", "k2", "selector1", "selector2", "google", "mail", "smtp", "mx", "dkim", "brevo", "mailgun", "pm", "resend"}
	}
	for _, s := range selectors {
		name := s + "._domainkey." + domain
		if recs, err := net.LookupTXT(name); err == nil {
			for _, r := range recs {
				if strings.Contains(r, "p=") {
					pass("DKIM: key found at %s", name)
					return
				}
			}
		}
		if cname, err := net.LookupCNAME(name); err == nil && strings.TrimSuffix(cname, ".") != name {
			pass("DKIM: %s points to %s", name, strings.TrimSuffix(cname, "."))
			return
		}
	}
	if selector != "" {
		fail("DKIM: no key at %s._domainkey.%s", selector, domain)
	} else {
		warn("DKIM: no key found at common selectors; rerun with -dkim-selector <selector from your provider>")
	}
}

func isLocal(host string) bool {
	return host == "localhost" || host == "127.0.0.1" || host == "::1" || host == "mailsink"
}

func isLocalDomain(d string) bool {
	return d == "localhost" || strings.HasSuffix(d, ".test") || strings.HasSuffix(d, ".local") ||
		strings.HasSuffix(d, ".example") || strings.HasSuffix(d, ".invalid") || !strings.Contains(d, ".")
}

func finish() {
	fmt.Println()
	switch {
	case problems > 0:
		fmt.Printf("%d problem(s), %d warning(s)\n", problems, warnings)
		os.Exit(1)
	case warnings > 0:
		fmt.Printf("works, with %d warning(s)\n", warnings)
	default:
		fmt.Println("all checks passed")
	}
	os.Exit(0)
}
