package models

import "time"

type AuditLog struct {
	ID           uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Timedate     time.Time `gorm:"not null;index" json:"timedate"`
	Method       string    `gorm:"not null" json:"method"`
	Path         string    `gorm:"not null;index" json:"path"`
	RemoteIP     string    `gorm:"not null;index" json:"remoteIp"`
	RemoteAgent  string    `json:"remoteAgent,omitempty"`
	ResponseTime int64     `gorm:"not null" json:"responseTime"`
	StatusCode   int       `gorm:"not null" json:"statusCode"`
	RequestJSON  *string   `gorm:"type:text" json:"requestJson,omitempty"`
	ResponseJSON *string   `gorm:"type:text" json:"responseJson,omitempty"`
	Headers      *string   `gorm:"type:text" json:"headers,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
}

func (AuditLog) TableName() string {
	return "audit_logs"
}
