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
	UpdatedAt   time.Time      `json:"updatedAt"`
}

type ParseJob struct {
	ID           string      `gorm:"type:uuid;primaryKey" json:"id"`
	DocumentID   string      `gorm:"index" json:"documentId"`
	InputBlobID  string      `gorm:"type:uuid;index" json:"inputBlobId"`
	Status       string      `json:"status"`
	OutputFormat string      `json:"outputFormat"`
	Language     string      `json:"language"`
	Error        *string     `json:"error,omitempty"`
	CreatedAt    time.Time   `json:"createdAt"`
	UpdatedAt    time.Time   `json:"updatedAt"`
	Result       ParseResult `gorm:"foreignKey:JobID;references:ID" json:"-"`
}

func (ParseJob) TableName() string {
	return "parse_jobs"
}

type ParseResult struct {
	ID          string    `gorm:"type:uuid;primaryKey" json:"id"`
	JobID       string    `gorm:"type:uuid;uniqueIndex" json:"jobId"`
	ContentType string    `json:"contentType"`
	ContentText string    `json:"contentText"`
	AssetsZip   []byte    `gorm:"type:bytea" json:"-"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (ParseResult) TableName() string {
	return "parse_results"
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

type UpdateInputBlobParams struct {
	Filename    *string
	FileType    *string
	ContentType *string
	Tags        []string
	SizeBytes   *int64
	SHA256      *string
	Content     []byte
	ReplaceFile bool
}
