package info

import (
	"context"
	"fmt"
	"log"

	"github.com/go-redis/redis/v8"
)

var RedisClient *redis.Client

func init() {
	// Redis 클라이언트 초기화
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     "localhost:26379", // Redis 서버 주소
		Password: "",                // 비밀번호 (없는 경우 빈 문자열)
		DB:       0,                 // 데이터베이스 번호
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

func GetLocationsWithPipeline(userIDs []int64) (map[int64]string, error) {
	pipe := RedisClient.Pipeline()

	// 각 user_id에 대한 키 생성
	keys := make([]string, len(userIDs))
	for _, userID := range userIDs {
		keys = append(keys, fmt.Sprintf("user:%d:location", userID))
	}

	getCmds := make([]*redis.StringCmd, len(keys))
	for i, key := range keys {
		getCmds[i] = pipe.Get(context.Background(), key)
	}

	// Pipeline 실행
	_, err := pipe.Exec(context.Background())
	if err != nil {
		return nil, err
	}

	// 결과를 map으로 변환
	locations := make(map[int64]string)
	for i, cmd := range getCmds {
		location, err := cmd.Result()
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

// func main() {
// 	// 여러 사용자의 위치 가져오기 (Pipeline 사용)
// 	userIDs := []int64{12345, 67890}
// 	locationMap, err := GetLocationsWithPipeline(userIDs)
// 	if err != nil {
// 		log.Fatal("Failed to get locations from Redis:", err)
// 	}

// 	for userID, location := range locationMap {
// 		fmt.Printf("User %d's location: %s\n", userID, location)
// 	}

// 	// 단일 사용자의 위치 설정 및 가져오기
// 	singleUserID := int64(54321)
// 	singleUserLocation := "Los Angeles"

// 	err = SetLocation(singleUserID, singleUserLocation)
// 	if err != nil {
// 		log.Fatal("Failed to set location in Redis:", err)
// 	}
// 	fmt.Printf("Location set for user %d\n", singleUserID)

// 	retrievedLocation, err := GetLocation(singleUserID)
// 	if err != nil {
// 		log.Fatal("Failed to get location from Redis:", err)
// 	}
// 	fmt.Printf("Retrieved location for user %d: %s\n", singleUserID, retrievedLocation)
// }
