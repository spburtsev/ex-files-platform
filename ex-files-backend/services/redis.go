package services

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisClient wraps a go-redis client and implements CacheService and ResetTokenStore.
type RedisClient struct {
	rdb *redis.Client
}

func NewRedisClient(addr string) (*RedisClient, error) {
	rdb := redis.NewClient(&redis.Options{Addr: addr})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	slog.Info("connected to Redis", "addr", addr)
	return &RedisClient{rdb: rdb}, nil
}

// --- CacheService ---

func (r *RedisClient) Get(key string) ([]byte, bool) {
	ctx := context.Background()
	val, err := r.rdb.Get(ctx, key).Bytes()
	if err != nil {
		return nil, false
	}
	return val, true
}

func (r *RedisClient) Set(key string, value []byte, ttl time.Duration) {
	ctx := context.Background()
	if err := r.rdb.Set(ctx, key, value, ttl).Err(); err != nil {
		slog.Error("redis SET failed", "key", key, "error", err)
	}
}

func (r *RedisClient) Delete(key string) {
	ctx := context.Background()
	r.rdb.Del(ctx, key)
}

// --- ResetTokenStore ---

const resetTokenPrefix = "reset_token:"

func (r *RedisClient) StoreResetToken(token string, userID uint, ttl time.Duration) error {
	ctx := context.Background()
	return r.rdb.Set(ctx, resetTokenPrefix+token, strconv.FormatUint(uint64(userID), 10), ttl).Err()
}

func (r *RedisClient) GetResetTokenUserID(token string) (uint, error) {
	ctx := context.Background()
	val, err := r.rdb.Get(ctx, resetTokenPrefix+token).Result()
	if err != nil {
		return 0, fmt.Errorf("token not found or expired")
	}
	id, err := strconv.ParseUint(val, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid token data")
	}
	return uint(id), nil
}

func (r *RedisClient) DeleteResetToken(token string) error {
	ctx := context.Background()
	return r.rdb.Del(ctx, resetTokenPrefix+token).Err()
}
