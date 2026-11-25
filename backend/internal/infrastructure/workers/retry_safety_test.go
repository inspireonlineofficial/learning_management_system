package workers

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"lms-backend/internal/domain/notifications"

	"github.com/google/uuid"
)

type recordingNotifications struct{ ids map[uuid.UUID]int }

func (r *recordingNotifications) Create(_ context.Context, n *notifications.Notification) error {
	r.ids[n.ID]++
	return nil
}

type recordingJobs struct{ jobs []notifications.Job }

func (q *recordingJobs) Enqueue(_ context.Context, j notifications.Job) error {
	q.jobs = append(q.jobs, j)
	return nil
}

type fixedContact struct{}

func (fixedContact) Contact(context.Context, uuid.UUID) (string, string, bool, error) {
	return "Rafi Ahmed", "rafi@example.com", true, nil
}

// A job that runs twice (retry after a late failure, or queued twice) must
// produce the same notification and email IDs, so nothing is duplicated.
func TestRetriedJobReusesNotificationAndEmailIDs(t *testing.T) {
	store := &recordingNotifications{ids: map[uuid.UUID]int{}}
	queue := &recordingJobs{}
	h := NewBookOrderJobHandler(StudentMessenger{Notifications: store, Contacts: fixedContact{}, Queue: queue, FrontendURL: "https://app.test"})
	payload, _ := json.Marshal(map[string]string{"order_id": uuid.NewString(), "student_id": uuid.NewString(), "book_title": "Physics", "format": "physical"})
	job := notifications.Job{ID: uuid.NewString(), Type: "order_shipped", Payload: payload}
	ctx := notifications.WithJobKey(context.Background(), job.ID)
	for i := 0; i < 2; i++ {
		if err := h.Handle(ctx, job); err != nil {
			t.Fatal(err)
		}
	}
	if len(store.ids) != 1 {
		t.Fatalf("expected one notification ID across retries, got %d", len(store.ids))
	}
	if len(queue.jobs) != 2 || queue.jobs[0].ID == "" || queue.jobs[0].ID != queue.jobs[1].ID {
		t.Fatalf("email job must keep the same ID across retries: %+v", queue.jobs)
	}

	// A different job for the same event gets different IDs.
	other := notifications.WithJobKey(context.Background(), uuid.NewString())
	if err := h.Handle(other, job); err != nil {
		t.Fatal(err)
	}
	if len(store.ids) != 2 {
		t.Fatalf("a separate job must create its own notification, got %d IDs", len(store.ids))
	}
}

func TestDedupeIDOutsideJobIsRandom(t *testing.T) {
	a := notifications.DedupeID(context.Background(), "x")
	b := notifications.DedupeID(context.Background(), "x")
	if a == b {
		t.Fatal("outside a job, IDs must be random")
	}
	ctx := notifications.WithJobKey(context.Background(), "job-1")
	if notifications.DedupeID(ctx, "x") != notifications.DedupeID(ctx, "x") || notifications.DedupeID(ctx, "x") == notifications.DedupeID(ctx, "y") {
		t.Fatal("inside a job, IDs must be stable per part and distinct across parts")
	}
}

func TestRetryPolicyPerJobType(t *testing.T) {
	w := NewWorker(nil, "test")
	w.SetMaxAttempts("transcode_video", 2)
	if w.attemptsFor("transcode_video") != 2 || w.attemptsFor("send_email") != MaxJobAttempts {
		t.Fatal("per-type attempt limits not applied")
	}
	if retryBackoff(1) >= retryBackoff(2) || retryBackoff(3) >= retryBackoff(4) {
		t.Fatal("backoff must grow")
	}
}

type fixedAdmins []uuid.UUID

func (a fixedAdmins) ActiveAdminIDs(context.Context) ([]uuid.UUID, error) { return a, nil }

func TestRefundRequestAlertsEveryAdminOnce(t *testing.T) {
	store := &recordingNotifications{ids: map[uuid.UUID]int{}}
	queue := &recordingJobs{}
	admins := fixedAdmins{uuid.New(), uuid.New(), uuid.New()}
	h := NewRefundRequestAdminAlertHandler(StudentMessenger{Notifications: store, Contacts: fixedContact{}, Queue: queue, FrontendURL: "https://app.test"}, admins)
	payload, _ := json.Marshal(map[string]interface{}{"book_title": "Physics", "student_name": "Rafi", "amount": 250, "currency": "BDT", "reason": "Wrong class"})
	job := notifications.Job{ID: uuid.NewString(), Type: "refund_request_admin_alert", Payload: payload}
	ctx := notifications.WithJobKey(context.Background(), job.ID)
	_ = h.Handle(ctx, job)
	_ = h.Handle(ctx, job) // retry
	if len(store.ids) != 3 {
		t.Fatalf("each of 3 admins should have exactly one notification, got %d", len(store.ids))
	}
	if len(queue.jobs) != 6 {
		t.Fatalf("expected 3 emails queued per run, got %d", len(queue.jobs))
	}
	var email struct{ HTML, Body string }
	_ = json.Unmarshal(queue.jobs[0].Payload, &email)
	if !strings.Contains(email.Body, "Wrong class") || !strings.Contains(email.Body, "/admin/bookshop/refunds") {
		t.Fatalf("admin email should include the reason and a link: %s", email.Body)
	}
}
