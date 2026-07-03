package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pxc1984/nnkl-backend/api"
	shared "github.com/pxc1984/nnkl-backend/api/v1/shared"
)

func (a *AuthAPI) sessions(c *gin.Context) {
	user, _ := api.CurrentUserFromContext(c)
	sessions, err := a.store.ListUserSessions(c.Request.Context(), user.ID)
	if err != nil {
		api.RespondError(c, http.StatusInternalServerError, "failed to load sessions", "internal_error")
		return
	}
	c.JSON(http.StatusOK, shared.ToSessionResponses(sessions))
}
