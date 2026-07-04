package data

import (
	"net/http"

	"github.com/gin-gonic/gin"
	shared "github.com/pxc1984/nnkl-backend/api/v1/shared"
)

func (a *DataAPI) get(c *gin.Context) {
	upload, err := a.store.GetUploadByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondStoreNotFound(c, err, "object not found")
		return
	}
	blob := &upload.InputBlob

	response := shared.KnowledgeObjectDetails{
		KnowledgeObject: shared.KnowledgeObject{
			KnowledgeObjectResponse: shared.ToKnowledgeObjectResponse(blob),
			Status:                  upload.Status,
		},
		SHA256:     blob.SHA256,
		HasContent: len(blob.Content) > 0,
		Language:   upload.Language,
		Error:      upload.Error,
		HasResult:  upload.OutputBlobID != nil,
	}

	c.JSON(http.StatusOK, response)
}
