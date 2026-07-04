package models

import "time"

type User struct {
	ID            string     `gorm:"type:uuid;primaryKey" json:"id"`
	Email         string     `gorm:"uniqueIndex;not null" json:"email"`
	Name          string     `json:"name,omitempty"`
	Role          string     `gorm:"not null" json:"role"`
	PasswordHash  string     `gorm:"not null" json:"-"`
	EmailVerified bool       `gorm:"not null;default:true" json:"emailVerified"`
	AvatarURL     *string    `json:"avatarUrl"`
	AvatarData    []byte     `gorm:"type:bytea" json:"-"`
	LastLoginAt   *time.Time `json:"lastLoginAt"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

type Session struct {
	ID               string     `gorm:"type:uuid;primaryKey" json:"id"`
	UserID           string     `gorm:"type:uuid;index;not null" json:"userId"`
	RefreshTokenHash string     `gorm:"uniqueIndex;not null" json:"-"`
	IP               string     `gorm:"not null" json:"ip"`
	UserAgent        string     `gorm:"not null" json:"userAgent"`
	CreatedAt        time.Time  `json:"createdAt"`
	LastUsedAt       time.Time  `json:"lastUsedAt"`
	ExpiresAt        time.Time  `gorm:"index;not null" json:"expiresAt"`
	RevokedAt        *time.Time `json:"-"`
	User             User       `gorm:"foreignKey:UserID" json:"-"`
}

type UpdateUserParams struct {
	Name       *string
	AvatarData []byte
	AvatarURL  *string
}

type CreateUserParams struct {
	Email        string
	Name         string
	Role         string
	PasswordHash string
}

type CreateSessionParams struct {
	UserID           string
	RefreshTokenHash string
	IP               string
	UserAgent        string
	ExpiresAt        time.Time
	LastUsedAt       time.Time
}

type UpdateSessionTokenParams struct {
	SessionID        string
	RefreshTokenHash string
	ExpiresAt        time.Time
	LastUsedAt       time.Time
}
