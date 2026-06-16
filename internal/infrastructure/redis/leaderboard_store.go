package redis

import (
	"context"
	"fmt"

	apppoints "lms-backend/internal/application/points"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

const (
	leaderboardWeeklyKey  = "leaderboard:weekly"
	leaderboardAlltimeKey = "leaderboard:alltime"
	optOutKeyPrefix       = "leaderboard:optout:"
)

// LeaderboardStore implements application/points.LeaderboardStore using Redis sorted sets.
type LeaderboardStore struct {
	client *Client
}

// NewLeaderboardStore creates a new LeaderboardStore.
func NewLeaderboardStore(client *Client) *LeaderboardStore {
	return &LeaderboardStore{client: client}
}

// AddScore increments the score for a member in the given sorted-set key.
func (s *LeaderboardStore) AddScore(ctx context.Context, key string, memberID uuid.UUID, score float64) error {
	return s.client.rdb.ZIncrBy(ctx, key, score, memberID.String()).Err()
}

// GetTopN returns the top N members with their scores in descending order.
func (s *LeaderboardStore) GetTopN(ctx context.Context, key string, n int) ([]apppoints.LeaderboardRawEntry, error) {
	results, err := s.client.rdb.ZRevRangeWithScores(ctx, key, 0, int64(n-1)).Result()
	if err != nil {
		return nil, fmt.Errorf("leaderboard GetTopN: %w", err)
	}

	entries := make([]apppoints.LeaderboardRawEntry, 0, len(results))
	for _, z := range results {
		memberStr, ok := z.Member.(string)
		if !ok {
			continue
		}
		id, err := uuid.Parse(memberStr)
		if err != nil {
			continue
		}
		entries = append(entries, apppoints.LeaderboardRawEntry{
			MemberID: id,
			Score:    z.Score,
		})
	}
	return entries, nil
}

// ResetWeekly deletes the weekly leaderboard sorted set.
func (s *LeaderboardStore) ResetWeekly(ctx context.Context) error {
	return s.client.rdb.Del(ctx, leaderboardWeeklyKey).Err()
}

// GetOptOutStatus returns true if the student has opted out of the leaderboard.
func (s *LeaderboardStore) GetOptOutStatus(ctx context.Context, studentID uuid.UUID) (bool, error) {
	key := optOutKeyPrefix + studentID.String()
	val, err := s.client.rdb.Get(ctx, key).Result()
	if err == goredis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return val == "1", nil
}

// SetOptOutStatus persists the student's opt-out preference.
func (s *LeaderboardStore) SetOptOutStatus(ctx context.Context, studentID uuid.UUID, optOut bool) error {
	key := optOutKeyPrefix + studentID.String()
	val := "0"
	if optOut {
		val = "1"
	}
	return s.client.rdb.Set(ctx, key, val, 0).Err()
}
