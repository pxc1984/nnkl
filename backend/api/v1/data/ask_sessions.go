package data

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pxc1984/nnkl-backend/api"
	"github.com/pxc1984/nnkl-backend/store/models"
)

type QuerySessionResponse struct {
	ID        string `json:"id"`
	Query     string `json:"query"`
	Answer    string `json:"answer"`
	Mode      string `json:"mode"`
	CreatedAt string `json:"createdAt"`
}

func (a *DataAPI) listAskSessions(c *gin.Context) {
	user, ok := api.CurrentUserFromContext(c)
	if !ok || user == nil {
		api.RespondError(c, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	page := parsePositiveInt(c.DefaultQuery("page", "1"), 1)
	pageSize := parsePositiveInt(c.DefaultQuery("pageSize", "8"), 8)

	sessions, _, err := a.store.ListQuerySessions(c.Request.Context(), user.ID, models.ListQuerySessionsParams{
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		api.RespondError(c, http.StatusInternalServerError, "failed to list ask sessions", "internal_error")
		return
	}

	response := make([]QuerySessionResponse, 0, len(sessions))
	for _, session := range sessions {
		response = append(response, QuerySessionResponse{
			ID:        session.ID,
			Query:     session.Query,
			Answer:    session.Response,
			Mode:      session.Mode,
			CreatedAt: session.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	c.JSON(http.StatusOK, response)
}
