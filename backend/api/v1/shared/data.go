package shared

import (
	"time"

	"github.com/pxc1984/nnkl-backend/store"
)

type KnowledgeObjectResponse struct {
	ID          string    `json:"id"`
	Filename    string    `json:"filename"`
	Type        string    `json:"type"`
	ContentType string    `json:"contentType"`
	Tags        []string  `json:"tags"`
	SizeBytes   int64     `json:"sizeBytes"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type KnowledgeObject struct {
	KnowledgeObjectResponse
	Status string `json:"status,omitempty"`
}

type KnowledgeObjectDetails struct {
	KnowledgeObject
	SHA256       *string `json:"sha256,omitempty"`
	HasContent   bool    `json:"hasContent"`
	HasResult    bool    `json:"hasResult"`
	OutputFormat string  `json:"outputFormat,omitempty"`
	Language     string  `json:"language,omitempty"`
	Error        *string `json:"error,omitempty"`
}

type PaginationMeta struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"pageSize"`
	Total      int64 `json:"total"`
	TotalPages int64 `json:"totalPages"`
}

type PaginatedKnowledgeObjectList struct {
	Items []KnowledgeObjectResponse `json:"items"`
	Meta  PaginationMeta            `json:"meta"`
}

func ToKnowledgeObjectResponse(blob *store.Blob) KnowledgeObjectResponse {
	return KnowledgeObjectResponse{
		ID:          blob.ID,
		Filename:    blob.Filename,
		Type:        blob.FileType,
		ContentType: blob.ContentType,
		Tags:        append([]string(nil), blob.Tags...),
		SizeBytes:   blob.SizeBytes,
		CreatedAt:   blob.CreatedAt,
		UpdatedAt:   blob.UpdatedAt,
	}
}

func ToKnowledgeObjectResponses(blobs []store.Blob) []KnowledgeObjectResponse {
	response := make([]KnowledgeObjectResponse, 0, len(blobs))
	for i := range blobs {
		response = append(response, ToKnowledgeObjectResponse(&blobs[i]))
	}
	return response
}
