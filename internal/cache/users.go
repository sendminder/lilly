package cache

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/go-redis/redis/v8"
)

type RedisClient interface {
	SetUserLocation(userID int64, location string) error
	DeleteUserLocation(userID int64) error
	GetUserLocation(userID int64) (string, error)
	GetUserLocations(userIDs []int64) (map[int64]string, error)
	EnqueueReadyUser(roleType string, userID int64) bool
	DequeueReadyUser(roleType string) (int64, error)
}

var _ RedisClient = (*redisClient)(nil)

type redisClient struct {
	client *redis.Client
	ctx    context.Context
}

func NewRedisClient(ctx context.Context, host string) RedisClient {
	// Redis 클라이언트 초기화
	client := redis.NewClient(&redis.Options{
		Addr:     host, // Redis 서버 주소
		Password: "",   // 비밀번호 (없는 경우 빈 문자열)
		DB:       0,    // 데이터베이스 번호
	})

	pong, err := client.Ping(ctx).Result()
	if err != nil {
		slog.Error("Failed to connect to Redis", "error", err)
		return nil
	}
	slog.Info("Connected to Redis", "pong", pong)
	return &redisClient{
		client: client,
		ctx:    ctx,
	}
}

func (r *redisClient) SetUserLocation(userID int64, location string) error {
	// 키와 값을 Redis에 저장
	err := r.client.Set(r.ctx, fmt.Sprintf("user:%d:location", userID), location, 0).Err()
	if err != nil {
		return err
	}
	return nil
}

func (r *redisClient) DeleteUserLocation(userID int64) error {
	// 키에 해당하는 데이터 삭제
	err := r.client.Del(r.ctx, fmt.Sprintf("user:%d:location", userID)).Err()
	if err != nil {
		return err
	}
	return nil
}

func (r *redisClient) GetUserLocation(userID int64) (string, error) {
	// 키를 이용해 Redis에서 값을 가져옴
	location, err := r.client.Get(r.ctx, fmt.Sprintf("user:%d:location", userID)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", nil
		}
		return "", err
	}
	return location, nil
}

func (r *redisClient) GetUserLocations(userIDs []int64) (map[int64]string, error) {
	// TODO: use pipeline
	// 결과를 map으로 변환
	locations := make(map[int64]string)
	for i, userID := range userIDs {
		location, err := r.GetUserLocation(userID)
		if err != nil {
			if !errors.Is(err, redis.Nil) {
				return nil, err
			}
			locations[userIDs[i]] = ""
		} else {
			locations[userIDs[i]] = location
		}
	}

	return locations, nil
}

func (r *redisClient) EnqueueReadyUser(roleType string, userID int64) bool {
	key := "queue" + roleType
	_, err := r.client.LPush(r.ctx, key, userID).Result()
	if err != nil {
		slog.Error("Error enqueuing user", "error", err)
		return false
	}
	slog.Info("[EnqueueReadyUser]", "key", key, "userID", userID)
	return true
}

func (r *redisClient) DequeueReadyUser(roleType string) (int64, error) {
	key := "queue" + roleType
	userID, err := r.client.RPop(r.ctx, key).Int64()
	if err != nil && !errors.Is(err, redis.Nil) {
		slog.Error("Error dequeuing user", "error", err)
		return 0, err
	}
	slog.Info("[DequeueReadyUser]", "key", key, "userID", userID)
	return userID, nil
}
