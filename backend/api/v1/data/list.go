package data

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pxc1984/nnkl-backend/api"
	shared "github.com/pxc1984/nnkl-backend/api/v1/shared"
	"github.com/pxc1984/nnkl-backend/store"
)

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
