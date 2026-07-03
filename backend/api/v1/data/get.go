package data

import (
	"net/http"

	"github.com/gin-gonic/gin"
	shared "github.com/pxc1984/nnkl-backend/api/v1/shared"
)

func (a *DataAPI) get(c *gin.Context) {
	blob, err := a.store.GetInputBlobByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondStoreNotFound(c, err, "object not found")
		return
	}

	response := shared.KnowledgeObjectDetails{
		KnowledgeObject: shared.KnowledgeObject{
			KnowledgeObjectResponse: shared.ToKnowledgeObjectResponse(blob),
		},
		SHA256:     blob.SHA256,
		HasContent: len(blob.Content) > 0,
	}

	job, err := a.store.GetParseJobByDocumentID(c.Request.Context(), blob.ID)
	if err == nil {
		response.Status = job.Status
		response.OutputFormat = job.OutputFormat
		response.Language = job.Language
		response.Error = job.Error
		response.HasResult = job.Result.ID != ""
	}

	c.JSON(http.StatusOK, response)
}
