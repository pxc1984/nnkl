package data

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pxc1984/nnkl-backend/api"
	shared "github.com/pxc1984/nnkl-backend/api/v1/shared"
	"github.com/pxc1984/nnkl-backend/store"
)

func (a *DataAPI) update(c *gin.Context) {
	current, err := a.store.GetInputBlobByID(c.Request.Context(), c.Param("id"))
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

	update := store.UpdateInputBlobParams{Tags: chooseTags(params.Tags, current.Tags)}
	newBlobType := current.FileType

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
			filename := fileHeader.Filename
			sizeBytes := int64(len(content))
			update.ReplaceFile = true
			update.Content = content
			update.ContentType = &contentType
			update.Filename = &filename
			update.FileType = &newBlobType
			update.SizeBytes = &sizeBytes
			update.SHA256 = &sha
		}
	}

	blob, err := a.store.UpdateInputBlob(c.Request.Context(), current.ID, update)
	if err != nil {
		api.RespondError(c, http.StatusInternalServerError, "failed to update object", "internal_error")
		return
	}

	if update.ReplaceFile {
		if err := a.reprocessBlob(c, blob.ID, defaultString(params.OutputFormat, "markdown"), defaultString(params.Language, "auto")); err != nil {
			return
		}
	}

	status := "updated"
	job, err := a.store.GetParseJobByDocumentID(c.Request.Context(), blob.ID)
	if err == nil {
		status = job.Status
	}

	c.JSON(http.StatusOK, shared.KnowledgeObject{
		KnowledgeObjectResponse: shared.ToKnowledgeObjectResponse(blob),
		Status:                  status,
	})
}
