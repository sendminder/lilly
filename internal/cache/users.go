package cache

import (
	"context"
	"fmt"
	"lilly/internal/config"
	"log"

	"github.com/go-redis/redis/v8"
)

var RedisClient *redis.Client

func CreateRedisConnection() {
	redisIP := config.GetString("redis.ip")
	redisPort := config.GetString("redis.port")
	redisPassword := config.GetString("redis.password")
	// Redis 클라이언트 초기화
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     redisIP + ":" + redisPort, // Redis 서버 주소
		Password: redisPassword,             // 비밀번호 (없는 경우 빈 문자열)
		DB:       0,                         // 데이터베이스 번호
	})

	// 연결 확인
	pong, err := RedisClient.Ping(context.Background()).Result()
	if err != nil {
		log.Fatal("Failed to connect to Redis:", err)
	}
	fmt.Println("Connected to Redis:", pong)
}

func SetUserLocation(userID int64, location string) error {
	// 키와 값을 Redis에 저장
	err := RedisClient.Set(context.Background(), fmt.Sprintf("user:%d:location", userID), location, 0).Err()
	if err != nil {
		return err
	}
	return nil
}

func DeleteUserLocation(userID int64) error {
	// 키에 해당하는 데이터 삭제
	err := RedisClient.Del(context.Background(), fmt.Sprintf("user:%d:location", userID)).Err()
	if err != nil {
		return err
	}
	return nil
}

func GetUserLocation(userID int64) (string, error) {
	// 키를 이용해 Redis에서 값을 가져옴
	location, err := RedisClient.Get(context.Background(), fmt.Sprintf("user:%d:location", userID)).Result()
	if err != nil {
		if err == redis.Nil {
			return "", nil
		}
		return "", err
	}
	return location, nil
}

func GetUserLocations(userIDs []int64) (map[int64]string, error) {
	// TODO: use pipeline
	// 결과를 map으로 변환
	locations := make(map[int64]string)
	for i, userId := range userIDs {
		location, err := GetUserLocation(userId)
		if err != nil {
			if err != redis.Nil {
				return nil, err
			}
			locations[userIDs[i]] = ""
		} else {
			locations[userIDs[i]] = location
		}
	}

	return locations, nil
}

func EnqueueReadyUser(roleType string, userId int64) bool {
	key := "queue" + roleType
	_, err := RedisClient.LPush(context.Background(), key, userId).Result()
	if err != nil {
		log.Println("Error enqueuing user:", err)
		return false
	}
	log.Printf("[EnqueueReadyUser] key=%s, userId=%d", key, userId)
	return true
}

func DequeueReadyUser(roleType string) (int64, error) {
	key := "queue" + roleType
	userId, err := RedisClient.RPop(context.Background(), key).Int64()
	if err != nil && err != redis.Nil {
		fmt.Println("Error dequeuing user:", err)
		return 0, err
	}
	log.Printf("[DequeueReadyUser] key=%s, userId=%d", key, userId)
	return userId, nil
}
