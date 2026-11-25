package handlers

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

// OpsHealth is the real state of the platform shown on the admin System page.
type OpsHealth struct {
	Status        string                `json:"status"` // ok | degraded
	UptimeSeconds int64                 `json:"uptime_seconds"`
	DBStatus      string                `json:"db_status"`
	RedisStatus   string                `json:"redis_status"`
	Queues        map[string]QueueDepth `json:"queues"`
	QueueDepth    int64                 `json:"queue_depth"`
	DeadJobs      int64                 `json:"dead_jobs"`
	WorkerCount   int                   `json:"worker_count"`
	Email         EmailStatus           `json:"email"`
	StorageCORS   string                `json:"storage_cors"`
	Problems      []string              `json:"problems"`
}

// QueueDepth counts jobs waiting, scheduled for retry and dead-lettered.
type QueueDepth struct {
	Pending int64 `json:"pending"`
	Retry   int64 `json:"retry"`
	Dead    int64 `json:"dead"`
}

// EmailStatus says whether email is configured (no credentials).
type EmailStatus struct {
	Configured bool   `json:"configured"`
	Server     string `json:"server"`
}

// OpsProvider supplies live health data and the test-email action.
type OpsProvider interface {
	Health(ctx context.Context) OpsHealth
	SendTestEmail(ctx context.Context, adminID uuid.UUID) (to string, err error)
}

// OpsHandler serves the admin system health and test email endpoints.
// The health endpoint used to return a hard-coded "ok" with a queue depth of
// 0 whatever the real state was.
type OpsHandler struct{ ops OpsProvider }

// NewOpsHandler creates an OpsHandler; Set wires the provider later.
func NewOpsHandler() *OpsHandler { return &OpsHandler{} }

// Set attaches the provider.
func (h *OpsHandler) Set(p OpsProvider) { h.ops = p }

// Health handles GET /v1/admin/system/health.
func (h *OpsHandler) Health(w http.ResponseWriter, r *http.Request) {
	if h.ops == nil {
		writeJSONResponse(w, http.StatusServiceUnavailable, map[string]string{"status": "unknown"})
		return
	}
	writeJSONResponse(w, http.StatusOK, h.ops.Health(r.Context()))
}

// SendTestEmail handles POST /v1/admin/system/test-email: sends a test email
// to the signed-in admin and reports the SMTP result.
func (h *OpsHandler) SendTestEmail(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	if h.ops == nil {
		writeJSONResponse(w, http.StatusServiceUnavailable, map[string]string{"error": "not available"})
		return
	}
	to, err := h.ops.SendTestEmail(r.Context(), userID)
	if err != nil {
		writeJSONResponse(w, http.StatusBadGateway, map[string]interface{}{"sent": false, "to": to, "error": err.Error()})
		return
	}
	writeJSONResponse(w, http.StatusOK, map[string]interface{}{"sent": true, "to": to})
}
