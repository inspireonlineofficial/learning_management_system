package main

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"lms-backend/internal/infrastructure/email"
	"lms-backend/internal/infrastructure/redis"
	"lms-backend/internal/infrastructure/workers"
	"lms-backend/internal/interfaces/http/handlers"

	"github.com/google/uuid"
)

// corsStatus records the outcome of the startup storage CORS check.
var corsStatus atomic.Value // string

func setCORSStatus(s string) { corsStatus.Store(s) }

// opsProvider reports the real platform state for the admin System page.
type opsProvider struct {
	started  time.Time
	db       *sql.DB
	redis    *redis.Client
	workers  map[string]*workers.Worker
	nWorkers int
	mailer   *email.Service
}

func (p *opsProvider) Health(ctx context.Context) handlers.OpsHealth {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	h := handlers.OpsHealth{
		Status:        "ok",
		UptimeSeconds: int64(time.Since(p.started).Seconds()),
		DBStatus:      "ok",
		RedisStatus:   "ok",
		Queues:        map[string]handlers.QueueDepth{},
		WorkerCount:   p.nWorkers,
		Email:         handlers.EmailStatus{Configured: p.mailer.Configured(), Server: p.mailer.Describe()},
		StorageCORS:   "checking",
		Problems:      []string{},
	}
	if v, ok := corsStatus.Load().(string); ok {
		h.StorageCORS = v
	}
	if err := p.db.PingContext(ctx); err != nil {
		h.DBStatus = "down"
		h.Problems = append(h.Problems, "Database unreachable")
	}
	if err := p.redis.Ping(ctx); err != nil {
		h.RedisStatus = "down"
		h.Problems = append(h.Problems, "Redis unreachable (queues and sessions affected)")
	} else {
		for name, w := range p.workers {
			st, err := w.Stats(ctx)
			if err != nil {
				continue
			}
			h.Queues[name] = handlers.QueueDepth{Pending: st.Pending, Retry: st.Retry, Dead: st.Dead}
			h.QueueDepth += st.Pending
			h.DeadJobs += st.Dead
		}
	}
	if !h.Email.Configured {
		h.Problems = append(h.Problems, "Email is not configured: OTP, password reset and order emails fail (set SMTP_*)")
	}
	if strings.HasPrefix(h.StorageCORS, "not set") {
		h.Problems = append(h.Problems, "Storage CORS is not set: the book reader falls back to the browser PDF viewer ("+h.StorageCORS+")")
	}
	if h.DeadJobs > 0 {
		h.Problems = append(h.Problems, fmt.Sprintf("%d background job(s) failed permanently (see docs/email.md, Monitoring)", h.DeadJobs))
	}
	if len(h.Problems) > 0 {
		h.Status = "degraded"
	}
	return h
}

func (p *opsProvider) SendTestEmail(ctx context.Context, adminID uuid.UUID) (string, error) {
	var name, address string
	if err := p.db.QueryRowContext(ctx, `SELECT full_name, email FROM users WHERE id = $1`, adminID).Scan(&name, &address); err != nil {
		return "", fmt.Errorf("could not find your account email")
	}
	msg := email.Transactional("Inspire email test", "Hi "+name+",",
		[]string{"This test was sent from System → Health. If you can read it, the LMS can send email.",
			"If it landed in spam, add the SPF, DKIM and DMARC records from your email provider (docs/email.md)."},
		nil, "Sent by an administrator from the Inspire admin panel.")
	msg.To = address
	return address, p.mailer.Send(ctx, msg)
}
