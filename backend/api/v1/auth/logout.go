package auth

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pxc1984/nnkl-backend/api"
	auth2 "github.com/pxc1984/nnkl-backend/auth"
	"github.com/pxc1984/nnkl-backend/metrics"
)

func (a *AuthAPI) logout(c *gin.Context) {
	var req logoutRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		api.RespondError(c, http.StatusBadRequest, "invalid request", "bad_request")
		return
	}

	user, _ := api.CurrentUserFromContext(c)
	claims, _ := api.CurrentClaimsFromContext(c)

	var err error
	if strings.TrimSpace(req.RefreshToken) != "" {
		err = a.store.DeleteSessionByUserAndHash(c.Request.Context(), user.ID, auth2.HashToken(req.RefreshToken))
	} else {
		err = a.store.DeleteSessionByID(c.Request.Context(), claims.SessionID)
	}
	if err != nil {
		api.RespondError(c, http.StatusForbidden, "session does not belong to user", "forbidden")
		return
	}
	metrics.AuthEventsTotal.WithLabelValues("logout", "success").Inc()
	c.Status(http.StatusNoContent)
}

func (a *AuthAPI) logoutAll(c *gin.Context) {
	user, _ := api.CurrentUserFromContext(c)
	if err := a.store.DeleteUserSessions(c.Request.Context(), user.ID); err != nil {
		api.RespondError(c, http.StatusInternalServerError, "failed to revoke sessions", "internal_error")
		return
	}
	metrics.AuthEventsTotal.WithLabelValues("logout_all", "success").Inc()
	c.Status(http.StatusNoContent)
}
