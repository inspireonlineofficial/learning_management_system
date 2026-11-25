package workers

import (
	"context"
	"encoding/json"
	"fmt"
	"lms-backend/internal/domain/notifications"
	"lms-backend/internal/infrastructure/redis"
	"time"

	"github.com/google/uuid"
)

// RedisQueue implements JobQueue using Redis lists
type RedisQueue struct {
	redis *redis.Client
	queue string
}

// NewRedisQueue creates a new Redis-backed job queue
func NewRedisQueue(redisClient *redis.Client, queue string) *RedisQueue {
	return &RedisQueue{
		redis: redisClient,
		queue: queue,
	}
}

// Enqueue adds a job to the queue
func (q *RedisQueue) Enqueue(ctx context.Context, job notifications.Job) error {
	if job.ID == "" {
		job.ID = uuid.New().String()
	}
	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	key := fmt.Sprintf("queue:%s", q.queue)
	if err := q.redis.RPush(ctx, key, string(data)); err != nil {
		return fmt.Errorf("failed to enqueue job: %w", err)
	}

	return nil
}

// Dequeue removes and returns a job from the queue
func (q *RedisQueue) Dequeue(ctx context.Context, timeout time.Duration) (*notifications.Job, error) {
	key := fmt.Sprintf("queue:%s", q.queue)
	result, err := q.redis.LPop(ctx, key)
	if err != nil {
		return nil, nil // No job available
	}

	var job notifications.Job
	if err := json.Unmarshal([]byte(result), &job); err != nil {
		return nil, fmt.Errorf("failed to unmarshal job: %w", err)
	}

	return &job, nil
}

// NotificationQueue implements the notification queue interface
type NotificationQueue struct {
	redis *redis.Client
}

// NewNotificationQueue creates a new notification queue
func NewNotificationQueue(redisClient *redis.Client) *NotificationQueue {
	return &NotificationQueue{
		redis: redisClient,
	}
}

// EnqueueNotification adds a notification job to the queue
func (q *NotificationQueue) EnqueueNotification(ctx context.Context, userID uuid.UUID, notificationType, title, body string) error {
	// Marshal properly: titles/bodies containing quotes or newlines (e.g. an
	// assignment named `Essay "Draft 2"`) produced invalid JSON before.
	payload, err := json.Marshal(map[string]string{
		"user_id": userID.String(),
		"type":    notificationType,
		"title":   title,
		"body":    body,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal notification payload: %w", err)
	}
	job := notifications.Job{ID: uuid.New().String(), Type: "send_notification", Payload: payload}

	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal notification job: %w", err)
	}

	key := "queue:notifications"
	if err := q.redis.RPush(ctx, key, string(data)); err != nil {
		return fmt.Errorf("failed to enqueue notification: %w", err)
	}

	return nil
}
