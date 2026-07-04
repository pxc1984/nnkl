package data

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pxc1984/nnkl-backend/api"
	"github.com/pxc1984/nnkl-backend/store"
	"gorm.io/gorm"
)

func (a *DataAPI) reprocessBlob(c *gin.Context, blobID, outputFormat, language string) error {
	if err := a.ocr.Parse(c.Request.Context(), OCRParseRequest{
		DocumentID:   blobID,
		InputBlobID:  blobID,
		OutputFormat: outputFormat,
		Language:     language,
	}); err != nil {
		api.RespondError(c, http.StatusBadGateway, fmt.Sprintf("ocr parse failed: %v", err), "ocr_error")
		return err
	}

	if outputFormat == "markdown" && a.lightrag != nil && a.lightrag.IsConfigured() {
		a.sendToLightRAG(c, blobID, outputFormat)
	}
	return nil
}

func (a *DataAPI) sendToLightRAG(c *gin.Context, blobID, outputFormat string) {
	job, err := a.store.GetParseJobByDocumentID(c.Request.Context(), blobID)
	if err != nil {
		slog.Warn("lightrag: failed to fetch parse job", "blob_id", blobID, "error", err)
		return
	}
	if job.Result.ContentText == "" {
		slog.Warn("lightrag: parse result has no content text", "blob_id", blobID)
		return
	}

	source := fmt.Sprintf("%s.md", blobID)
	if err := a.lightrag.SendText(c.Request.Context(), job.Result.ContentText, source); err != nil {
		slog.Warn("lightrag: failed to send text", "blob_id", blobID, "error", err)
		return
	}
	slog.Info("lightrag: text indexed", "blob_id", blobID, "source", source)
}

func (a *DataAPI) persistFile(c *gin.Context, fileHeader *multipart.FileHeader, fileType string, tags []string) (*store.InputBlob, error) {
	content, contentType, sha, err := readMultipartFile(fileHeader)
	if err != nil {
		return nil, err
	}

	return a.store.CreateInputBlob(c.Request.Context(), store.CreateInputBlobParams{
		Filename:    fileHeader.Filename,
		FileType:    fileType,
		ContentType: contentType,
		Tags:        tags,
		SizeBytes:   int64(len(content)),
		SHA256:      &sha,
		Content:     content,
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

func parseUpdateParams(c *gin.Context) (DataUpdateParams, bool) {
	raw := strings.TrimSpace(c.PostForm("params"))
	if raw == "" {
		return DataUpdateParams{}, true
	}
	var params DataUpdateParams
	if err := json.Unmarshal([]byte(raw), &params); err != nil {
		api.RespondError(c, http.StatusBadRequest, "invalid params json", "bad_request")
		return DataUpdateParams{}, false
	}
	params.Tags = trimNonEmpty(params.Tags)
	params.OutputFormat = strings.ToLower(strings.TrimSpace(params.OutputFormat))
	params.Language = strings.ToLower(strings.TrimSpace(params.Language))
	if params.OutputFormat != "" && params.OutputFormat != "latex" && params.OutputFormat != "markdown" {
		api.RespondError(c, http.StatusBadRequest, "unsupported output format", "bad_request")
		return DataUpdateParams{}, false
	}
	return params, true
}

func readMultipartFile(fileHeader *multipart.FileHeader) ([]byte, string, string, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return nil, "", "", err
	}
	defer func() { _ = file.Close() }()
	content, err := io.ReadAll(file)
	if err != nil {
		return nil, "", "", err
	}
	digest := sha256.Sum256(content)
	sha := hex.EncodeToString(digest[:])
	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(content)
	}
	return bytes.Clone(content), contentType, sha, nil
}

func chooseTags(updated []string, existing []string) []string {
	if updated == nil {
		return append([]string(nil), existing...)
	}
	return updated
}

func respondStoreNotFound(c *gin.Context, err error, message string) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		api.RespondError(c, http.StatusNotFound, message, "not_found")
		return
	}
	api.RespondError(c, http.StatusInternalServerError, "storage error", "internal_error")
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
