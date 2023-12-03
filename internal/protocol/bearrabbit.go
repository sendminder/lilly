package protocol

type RegisterRole struct {
	UserID   int64  `json:"user_id"`
	RoleType string `json:"role_type"`
}
