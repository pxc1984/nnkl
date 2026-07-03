package store

import (
	"time"

	"github.com/lib/pq"
)

type InputBlob struct {
	ID          string         `gorm:"type:uuid;primaryKey" json:"id"`
	Filename    string         `gorm:"not null;index" json:"filename"`
	FileType    string         `gorm:"not null;index" json:"type"`
	ContentType string         `gorm:"not null" json:"contentType"`
	Tags        pq.StringArray `gorm:"type:text[]" json:"tags"`
	SizeBytes   int64          `gorm:"not null" json:"sizeBytes"`
	SHA256      *string        `gorm:"size:64" json:"sha256,omitempty"`
	Content     []byte         `gorm:"type:bytea;not null" json:"-"`
	CreatedAt   time.Time      `json:"createdAt"`
}

type CreateInputBlobParams struct {
	Filename    string
	FileType    string
	ContentType string
	Tags        []string
	SizeBytes   int64
	SHA256      *string
	Content     []byte
}

type ListInputBlobsParams struct {
	Page     int
	PageSize int
	Query    string
	FileType string
	Tags     []string
}
