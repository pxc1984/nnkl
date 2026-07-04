package models

import (
	"encoding/json"
	"time"
)

// QuerySession records a user's query to the knowledge base (LightRAG).
type QuerySession struct {
	ID         string          `gorm:"type:uuid;primaryKey" json:"id"`
	UserID     string          `gorm:"type:uuid;index;not null" json:"userId"`
	Query      string          `gorm:"type:text;not null" json:"query"`
	Mode       string          `gorm:"not null" json:"mode"`
	Response   string          `gorm:"type:text" json:"response"`
	References json.RawMessage `gorm:"type:jsonb" json:"references,omitempty"`
	CreatedAt  time.Time       `json:"createdAt"`
	User       User            `gorm:"foreignKey:UserID" json:"-"`
}

func (QuerySession) TableName() string {
	return "query_sessions"
}

type CreateQuerySessionParams struct {
	UserID     string
	Query      string
	Mode       string
	Response   string
	References json.RawMessage
}

type ListQuerySessionsParams struct {
	Page     int
	PageSize int
}
