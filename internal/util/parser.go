package util

import (
	"log/slog"
	"net/http"
	"strconv"
)

// 클라이언트의 식별자(예: 유저 이름 또는 ID)를 추출하는 함수
func GetClientUserId(r *http.Request) int64 {
	// 클라이언트로부터 전달된 식별자 값을 추출하는 로직을 구현해야 합니다.
	// 예를 들어, URL 쿼리 매개변수로 식별자를 전달받는다고 가정하면 다음과 같이 구현할 수 있습니다.
	userId := r.URL.Query().Get("user_id")
	num, err := strconv.ParseInt(userId, 10, 64)
	if err != nil {
		slog.Error("getClientUserId ParseInt error", "error", err)
		return -1
	}
	return num
}
