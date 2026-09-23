package utils

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// CleanupStaleRunningTasks removes running-task keys that may have been left
// behind. It is a debugging and maintenance helper for clearing running keys
// stranded by abnormal shutdowns.
func CleanupStaleRunningTasks(ctx context.Context, redisClient *redis.Client, keyPrefix string, maxAge time.Duration) (int, error) {
	keys, err := redisClient.Keys(ctx, keyPrefix+"*").Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get keys: %w", err)
	}

	if len(keys) == 0 {
		return 0, nil
	}

	var staleTasks []string
	for _, key := range keys {
		ttl, err := redisClient.TTL(ctx, key).Result()
		if err != nil {
			continue
		}

		// Treat a key as stale when the TTL is negative (never expires) or the
		// remaining time is implausibly long.
		if ttl < 0 || ttl > maxAge {
			staleTasks = append(staleTasks, key)
		}
	}

	if len(staleTasks) == 0 {
		return 0, nil
	}

	deleted, err := redisClient.Del(ctx, staleTasks...).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to delete stale keys: %w", err)
	}

	return int(deleted), nil
}

// CheckRunningTaskStatus reports the status of a given running task.
func CheckRunningTaskStatus(ctx context.Context, redisClient *redis.Client, runningKey, progressKey string) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	runningTaskID, err := redisClient.Get(ctx, runningKey).Result()
	if err != nil {
		if err == redis.Nil {
			result["running_task_exists"] = false
		} else {
			return nil, fmt.Errorf("failed to get running task: %w", err)
		}
	} else {
		result["running_task_exists"] = true
		result["running_task_id"] = runningTaskID

		ttl, _ := redisClient.TTL(ctx, runningKey).Result()
		result["running_task_ttl"] = ttl.String()
	}

	progressData, err := redisClient.Get(ctx, progressKey).Result()
	if err != nil {
		if err == redis.Nil {
			result["progress_exists"] = false
		} else {
			return nil, fmt.Errorf("failed to get progress: %w", err)
		}
	} else {
		result["progress_exists"] = true
		result["progress_data"] = progressData

		ttl, _ := redisClient.TTL(ctx, progressKey).Result()
		result["progress_ttl"] = ttl.String()
	}

	return result, nil
}
