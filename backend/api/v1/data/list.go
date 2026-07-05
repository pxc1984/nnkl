package data

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pxc1984/nnkl-backend/api"
	shared "github.com/pxc1984/nnkl-backend/api/v1/shared"
	"github.com/pxc1984/nnkl-backend/store/models"
)

func (a *DataAPI) list(c *gin.Context) {
	page := parsePositiveInt(c.DefaultQuery("page", "1"), 1)
	pageSize := parsePositiveInt(c.DefaultQuery("pageSize", "20"), 20)

	uploads, total, err := a.store.ListUploads(c.Request.Context(), models.ListUploadsParams{
		Page:     page,
		PageSize: pageSize,
		Query:    strings.TrimSpace(c.Query("query")),
		FileType: strings.ToLower(strings.TrimSpace(c.Query("type"))),
		Status:   normalizeUploadListStatus(strings.ToLower(strings.TrimSpace(c.Query("status")))),
		Language: normalizeLanguageFilter(c.Query("language")),
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

	c.JSON(http.StatusOK, shared.PaginatedKnowledgeObjectList{
		Items: shared.ToKnowledgeObjects(uploads),
		Meta: shared.PaginationMeta{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}

func normalizeLanguageFilter(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "ru", "en", "auto":
		return value
	default:
		return ""
	}
}

func normalizeUploadListStatus(value string) string {
	switch value {
	case "ready":
		return "completed"
	case "pending", "processing", "completed", "failed":
		return value
	default:
		return ""
	}
}
