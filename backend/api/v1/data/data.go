package data

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pxc1984/nnkl-backend/api"
	shared "github.com/pxc1984/nnkl-backend/api/v1/shared"
	"github.com/pxc1984/nnkl-backend/store"
	"gorm.io/gorm"
)

type DataAPI struct {
	store store.Store
	ocr   *OCRClient
	maxMB int64
}

type DataUploadParams struct {
	Tags         []string `json:"tags"`
	OutputFormat string   `json:"outputFormat"`
	Language     string   `json:"language"`
}

type DataUploadItem struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
	Type     string `json:"type"`
	Status   string `json:"status"`
}

type DataUploadResponse struct {
	Items []DataUploadItem `json:"items"`
}

func (a *DataAPI) list(c *gin.Context) {
	page := parsePositiveInt(c.DefaultQuery("page", "1"), 1)
	pageSize := parsePositiveInt(c.DefaultQuery("pageSize", "20"), 20)
	tags := trimNonEmpty(c.QueryArray("tags"))

	blobs, total, err := a.store.ListInputBlobs(c.Request.Context(), store.ListInputBlobsParams{
		Page:     page,
		PageSize: pageSize,
		Query:    strings.TrimSpace(c.Query("query")),
		FileType: strings.ToLower(strings.TrimSpace(c.Query("type"))),
		Tags:     tags,
	})
	if err != nil {
		api.RespondError(c, http.StatusInternalServerError, "failed to list objects", "internal_error")
		return
	}
	if len(blobs) == 0 {
		c.Status(http.StatusNoContent)
		return
	}

	c.JSON(http.StatusOK, shared.PaginatedKnowledgeObjectList{
		Items:    shared.ToKnowledgeObjectResponses(blobs),
		Page:     page,
		PageSize: pageSize,
		Total:    total,
	})
}

func (a *DataAPI) upload(c *gin.Context) {
	params, ok := parseUploadParams(c)
	if !ok {
		return
	}
	form, err := c.MultipartForm()
	if err != nil {
		api.RespondError(c, http.StatusBadRequest, "invalid multipart form", "bad_request")
		return
	}
	files := form.File["data"]
	if len(files) == 0 {
		api.RespondError(c, http.StatusBadRequest, "missing files in data field", "bad_request")
		return
	}

	response := DataUploadResponse{Items: make([]DataUploadItem, 0, len(files))}
	for _, fileHeader := range files {
		fileType := detectSupportedFileType(fileHeader.Filename)
		if fileType == "" {
			api.RespondError(c, http.StatusBadRequest, "unsupported file type", "bad_request")
			return
		}
		if a.maxMB > 0 && fileHeader.Size > a.maxMB*1024*1024 {
			api.RespondError(c, http.StatusRequestEntityTooLarge, "uploaded file is too large", "payload_too_large")
			return
		}

		blob, err := a.persistFile(c, fileHeader, fileType, params.Tags)
		if err != nil {
			if err == gorm.ErrDuplicatedKey {
				api.RespondError(c, http.StatusConflict, "blob already exists", "conflict")
				return
			}
			api.RespondError(c, http.StatusInternalServerError, "failed to store blob", "internal_error")
			return
		}

		if err := a.ocr.Parse(c.Request.Context(), OCRParseRequest{
			DocumentID:   blob.ID,
			InputBlobID:  blob.ID,
			OutputFormat: defaultString(params.OutputFormat, "markdown"),
			Language:     defaultString(params.Language, "auto"),
		}); err != nil {
			api.RespondError(c, http.StatusBadGateway, fmt.Sprintf("ocr parse failed: %v", err), "ocr_error")
			return
		}

		response.Items = append(response.Items, DataUploadItem{
			ID:       blob.ID,
			Filename: blob.Filename,
			Type:     blob.FileType,
			Status:   "created",
		})
	}

	c.JSON(http.StatusCreated, response)
}

func (a *DataAPI) persistFile(c *gin.Context, fileHeader *multipart.FileHeader, fileType string, tags []string) (*store.InputBlob, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	content, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256(content)
	sha := hex.EncodeToString(digest[:])
	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(content)
	}

	return a.store.CreateInputBlob(c.Request.Context(), store.CreateInputBlobParams{
		Filename:    fileHeader.Filename,
		FileType:    fileType,
		ContentType: contentType,
		Tags:        tags,
		SizeBytes:   int64(len(content)),
		SHA256:      &sha,
		Content:     bytes.Clone(content),
	})
}

func parseUploadParams(c *gin.Context) (DataUploadParams, bool) {
	raw := strings.TrimSpace(c.PostForm("params"))
	if raw == "" {
		return DataUploadParams{}, true
	}
	var params DataUploadParams
	if err := json.Unmarshal([]byte(raw), &params); err != nil {
		api.RespondError(c, http.StatusBadRequest, "invalid params json", "bad_request")
		return DataUploadParams{}, false
	}
	params.Tags = trimNonEmpty(params.Tags)
	params.OutputFormat = strings.ToLower(strings.TrimSpace(params.OutputFormat))
	params.Language = strings.ToLower(strings.TrimSpace(params.Language))
	if params.OutputFormat != "" && params.OutputFormat != "latex" && params.OutputFormat != "markdown" {
		api.RespondError(c, http.StatusBadRequest, "unsupported output format", "bad_request")
		return DataUploadParams{}, false
	}
	return params, true
}

func detectSupportedFileType(filename string) string {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".pdf":
		return "pdf"
	case ".docx":
		return "docx"
	case ".pptx":
		return "pptx"
	default:
		return ""
	}
}

func trimNonEmpty(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func parsePositiveInt(raw string, fallback int) int {
	var parsed int
	if _, err := fmt.Sscanf(strings.TrimSpace(raw), "%d", &parsed); err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
