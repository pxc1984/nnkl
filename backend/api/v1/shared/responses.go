package shared

import (
	"time"

	"github.com/pxc1984/nnkl-backend/store"
)

type UserResponse struct {
	ID            string     `json:"id"`
	Email         string     `json:"email"`
	Name          string     `json:"name,omitempty"`
	Role          string     `json:"role"`
	EmailVerified bool       `json:"emailVerified"`
	AvatarURL     *string    `json:"avatarUrl"`
	LastLoginAt   *time.Time `json:"lastLoginAt"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

type SessionResponse struct {
	ID         string    `json:"id"`
	IP         string    `json:"ip"`
	UserAgent  string    `json:"userAgent"`
	CreatedAt  time.Time `json:"createdAt"`
	LastUsedAt time.Time `json:"lastUsedAt"`
}

func ToUserResponse(user *store.User) UserResponse {
	return UserResponse{
		ID:            user.ID,
		Email:         user.Email,
		Name:          user.Name,
		Role:          user.Role,
		EmailVerified: user.EmailVerified,
		AvatarURL:     user.AvatarURL,
		LastLoginAt:   user.LastLoginAt,
		CreatedAt:     user.CreatedAt,
		UpdatedAt:     user.UpdatedAt,
	}
}

func ToSessionResponses(sessions []store.Session) []SessionResponse {
	response := make([]SessionResponse, 0, len(sessions))
	for _, session := range sessions {
		response = append(response, SessionResponse{
			ID:         session.ID,
			IP:         session.IP,
			UserAgent:  session.UserAgent,
			CreatedAt:  session.CreatedAt,
			LastUsedAt: session.LastUsedAt,
		})
	}
	return response
}
