package httpapi

import (
	"time"

	"pimed/internal/domain/user"
)

type userDTO struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"createdAt"`
}

func toUserDTO(u user.User) userDTO {
	return userDTO{ID: u.ID, Username: u.Username, Role: string(u.Role), CreatedAt: u.CreatedAt.UTC()}
}

type userResponse struct {
	User userDTO `json:"user"`
}

type usersResponse struct {
	Users []userDTO `json:"users"`
}

type setupStatusResponse struct {
	NeedsSetup bool `json:"needsSetup"`
}

type credentialsRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type createUserRequest struct {
	Username string  `json:"username"`
	Password string  `json:"password"`
	Role     *string `json:"role"` // nil when omitted (defaults to "user")
}
