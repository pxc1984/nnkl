package data

import (
	"net/http"

	"github.com/gin-gonic/gin"
	shared "github.com/pxc1984/nnkl-backend/api/v1/shared"
	"github.com/pxc1984/nnkl-backend/store/models"
)

func (a *DataAPI) get(c *gin.Context) {
	upload, err := a.store.GetUploadByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondStoreNotFound(c, err, "object not found")
		return
	}
	blob := &upload.InputBlob
	content, outputFormat := extractKnowledgeObjectContent(upload)

	response := shared.KnowledgeObjectDetails{
		KnowledgeObject: shared.KnowledgeObject{
			KnowledgeObjectResponse: shared.ToKnowledgeObjectResponse(blob),
			Status:                  upload.Status,
		},
		SHA256:       blob.SHA256,
		HasContent:   len(blob.Content) > 0,
		HasResult:    upload.OutputBlobID != nil,
		OutputFormat: outputFormat,
		Content:      content,
		Language:     upload.Language,
		Error:        upload.Error,
	}

	c.JSON(http.StatusOK, response)
}

func extractKnowledgeObjectContent(upload *models.Upload) (string, string) {
	if isMarkdownBlob(upload.OutputBlob) {
		return string(upload.OutputBlob.Content), "markdown"
	}

	if isMarkdownBlob(&upload.InputBlob) {
		return string(upload.InputBlob.Content), "markdown"
	}

	return "", ""
}
