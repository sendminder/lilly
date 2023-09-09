package protocol

type RegisterRole struct {
	UserId   int64  `json:"user_id"`
	RoleType string `json:"role_type"`
}
