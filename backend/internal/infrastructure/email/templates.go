package email

import (
	"bytes"
	"html/template"
	"strings"
)

// Action is an optional call-to-action link in a transactional email.
type Action struct {
	Label string
	URL   string
}

// Transactional renders a simple, mobile-friendly email with a plain-text
// alternative. All values are escaped by html/template.
func Transactional(subject, greeting string, paragraphs []string, action *Action, footer string) Message {
	var text strings.Builder
	if greeting != "" {
		text.WriteString(greeting + "\n\n")
	}
	for _, p := range paragraphs {
		text.WriteString(p + "\n\n")
	}
	if action != nil && action.URL != "" {
		text.WriteString(action.Label + ": " + action.URL + "\n\n")
	}
	if footer != "" {
		text.WriteString("-- \n" + footer + "\n")
	}

	var html bytes.Buffer
	_ = transactionalHTML.Execute(&html, struct {
		Subject, Greeting, Footer string
		Paragraphs                []string
		Action                    *Action
	}{subject, greeting, footer, paragraphs, action})

	return Message{Subject: subject, Text: strings.TrimSpace(text.String()) + "\n", HTML: html.String()}
}

var transactionalHTML = template.Must(template.New("t").Parse(`<!DOCTYPE html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>{{.Subject}}</title></head>
<body style="margin:0;padding:0;background:#f5f3ee;font-family:Arial,Helvetica,sans-serif;color:#1f2a44">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background:#f5f3ee;padding:24px 12px">
<tr><td align="center">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="max-width:560px;background:#ffffff;border:1px solid #e6e1d6">
<tr><td style="padding:20px 28px;border-bottom:1px solid #e6e1d6;font-family:Georgia,serif;font-size:20px">Inspire</td></tr>
<tr><td style="padding:28px">
<h1 style="margin:0 0 16px;font-family:Georgia,serif;font-size:22px;font-weight:normal">{{.Subject}}</h1>
{{if .Greeting}}<p style="margin:0 0 14px;font-size:15px;line-height:1.6">{{.Greeting}}</p>{{end}}
{{range .Paragraphs}}<p style="margin:0 0 14px;font-size:15px;line-height:1.6">{{.}}</p>{{end}}
{{if .Action}}{{if .Action.URL}}<p style="margin:24px 0 8px"><a href="{{.Action.URL}}" style="display:inline-block;background:#1f2a44;color:#ffffff;text-decoration:none;padding:12px 22px;font-size:14px">{{.Action.Label}}</a></p>{{end}}{{end}}
</td></tr>
{{if .Footer}}<tr><td style="padding:16px 28px;border-top:1px solid #e6e1d6;font-size:12px;line-height:1.5;color:#5b6378">{{.Footer}}</td></tr>{{end}}
</table>
</td></tr></table>
</body></html>`))
