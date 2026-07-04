package data

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pxc1984/nnkl-backend/api"
	shared "github.com/pxc1984/nnkl-backend/api/v1/shared"
	"github.com/pxc1984/nnkl-backend/store/models"
	"github.com/pxc1984/nnkl-backend/worker"
)

func (a *DataAPI) update(c *gin.Context) {
	current, err := a.store.GetUploadByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondStoreNotFound(c, err, "object not found")
		return
	}

	params, ok := parseUpdateParams(c)
	if !ok {
		return
	}

	form, err := c.MultipartForm()
	if err != nil {
		form = nil
	}

	newBlobType := current.InputBlob.FileType

	if form != nil {
		files := form.File["data"]
		if len(files) > 1 {
			api.RespondError(c, http.StatusBadRequest, "only one file can be uploaded for update", "bad_request")
			return
		}
		if len(files) == 1 {
			fileHeader := files[0]
			newBlobType = detectSupportedFileType(fileHeader.Filename)
			if newBlobType == "" {
				api.RespondError(c, http.StatusBadRequest, "unsupported file type", "bad_request")
				return
			}
			if a.maxMB > 0 && fileHeader.Size > a.maxMB*1024*1024 {
				api.RespondError(c, http.StatusRequestEntityTooLarge, "uploaded file is too large", "payload_too_large")
				return
			}
			content, contentType, sha, err := readMultipartFile(fileHeader)
			if err != nil {
				api.RespondError(c, http.StatusInternalServerError, "failed to read uploaded file", "internal_error")
				return
			}
			blob, err := a.store.GetBlobBySHA256(c.Request.Context(), sha)
			if err != nil {
				blob, err = a.store.CreateBlob(c.Request.Context(), models.CreateBlobParams{
					Filename:    fileHeader.Filename,
					FileType:    newBlobType,
					ContentType: contentType,
					SizeBytes:   int64(len(content)),
					SHA256:      &sha,
					Content:     content,
				})
				if err != nil {
					api.RespondError(c, http.StatusInternalServerError, "failed to store uploaded file", "internal_error")
					return
				}
			}
			inputBlobID := blob.ID
			language := defaultString(params.Language, current.Language)
			status := "pending"
			var outputBlobID *string
			if isMarkdownBlob(blob) {
				status = "completed"
				outputBlobID = &inputBlobID
			}
			upload, err := a.store.UpdateUpload(c.Request.Context(), current.ID, models.UpdateUploadParams{
				InputBlobID:  &inputBlobID,
				OutputBlobID: outputBlobID,
				Status:       &status,
				Language:     &language,
				Error:        stringPtr(""),
			})
			if err != nil {
				api.RespondError(c, http.StatusInternalServerError, "failed to update object", "internal_error")
				return
			}
			if !isMarkdownBlob(blob) {
				job := worker.Job{
					UploadID:     upload.ID,
					OutputFormat: defaultString(params.OutputFormat, "markdown"),
					Language:     language,
					FileType:     blob.FileType,
				}
				switch blob.FileType {
				case "docx", "pptx":
					a.queue.EnqueueSimple(job)
				case "pdf":
					a.queue.EnqueueOCR(job)
				}
			}
			c.JSON(http.StatusOK, shared.KnowledgeObject{
				KnowledgeObjectResponse: shared.ToKnowledgeObjectResponse(&upload.InputBlob),
				Status:                  upload.Status,
			})
			return
		}
	}

	c.JSON(http.StatusOK, shared.KnowledgeObject{
		KnowledgeObjectResponse: shared.ToKnowledgeObjectResponse(&current.InputBlob),
		Status:                  current.Status,
	})
}
