package store

import (
	"time"

	"github.com/lib/pq"
)

type Blob struct {
	ID          string         `gorm:"type:uuid;primaryKey" json:"id"`
	Filename    string         `gorm:"not null;index" json:"filename"`
	FileType    string         `gorm:"not null;index" json:"type"`
	ContentType string         `gorm:"not null;index" json:"contentType"`
	SizeBytes   int64          `gorm:"not null" json:"sizeBytes"`
	SHA256      *string        `gorm:"size:64;index" json:"sha256,omitempty"`
	Content     []byte         `gorm:"type:bytea;not null" json:"-"`
	Tags        pq.StringArray `gorm:"-" json:"tags"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
}

func (Blob) TableName() string {
	return "blobs"
}

type Upload struct {
	ID           string    `gorm:"type:uuid;primaryKey" json:"id"`
	InputBlobID  string    `gorm:"column:input_blob;type:uuid;index" json:"inputBlobId"`
	OutputBlobID *string   `gorm:"column:output_blob;type:uuid;index" json:"outputBlobId,omitempty"`
	Status       string    `json:"status"`
	Language     string    `json:"language"`
	Error        *string   `json:"error,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
	InputBlob    Blob      `gorm:"foreignKey:InputBlobID;references:ID" json:"-"`
	OutputBlob   *Blob     `gorm:"foreignKey:OutputBlobID;references:ID" json:"-"`
}

func (Upload) TableName() string {
	return "uploads"
}

type CreateBlobParams struct {
	Filename    string
	FileType    string
	ContentType string
	SizeBytes   int64
	SHA256      *string
	Content     []byte
}

type CreateUploadParams struct {
	ID          string
	InputBlobID string
	Status      string
	Language    string
	Error       *string
}

type ListUploadsParams struct {
	Page     int
	PageSize int
	Query    string
	FileType string
	Tags     []string
}

type UpdateUploadParams struct {
	InputBlobID  *string
	OutputBlobID *string
	Status       *string
	Language     *string
	Error        *string
}
