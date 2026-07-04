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

	uploads, total, err := a.store.ListUploads(c.Request.Context(), store.ListUploadsParams{
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

	var totalPages int64
	if pageSize > 0 {
		totalPages = (total + int64(pageSize) - 1) / int64(pageSize)
	}
	if totalPages == 0 {
		totalPages = 1
	}

	blobs := make([]store.Blob, 0, len(uploads))
	for i := range uploads {
		blobs = append(blobs, uploads[i].InputBlob)
	}

	c.JSON(http.StatusOK, shared.PaginatedKnowledgeObjectList{
		Items: shared.ToKnowledgeObjectResponses(blobs),
		Meta: shared.PaginationMeta{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}
