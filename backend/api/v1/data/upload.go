package data

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pxc1984/nnkl-backend/api"
	"gorm.io/gorm"
)

func (a *DataAPI) upload(c *gin.Context) {
	params, ok := parseUploadParams(c)
	if !ok {
		return
	}
	form, err := c.MultipartForm()
	if err != nil {
		slog.Error("parse multipart form failed", "error", err)
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
		slog.Info("upload request", "filename", fileHeader.Filename, "size_bytes", fileHeader.Size, "max_mb", a.maxMB)
		if a.maxMB > 0 && fileHeader.Size > a.maxMB*1024*1024 {
			api.RespondError(c, http.StatusRequestEntityTooLarge, fmt.Sprintf("uploaded file is too large (size: %.2f MB, limit: %d MB)", float64(fileHeader.Size)/(1024*1024), a.maxMB), "payload_too_large")
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

		if err := a.reprocessBlob(c, blob.ID, defaultString(params.OutputFormat, "markdown"), defaultString(params.Language, "auto")); err != nil {
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
