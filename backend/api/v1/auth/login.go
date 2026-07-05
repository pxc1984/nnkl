package auth

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pxc1984/nnkl-backend/api"
	shared "github.com/pxc1984/nnkl-backend/api/v1/shared"
	auth2 "github.com/pxc1984/nnkl-backend/auth"
	"github.com/pxc1984/nnkl-backend/metrics"
	"github.com/pxc1984/nnkl-backend/store/models"
)

func (a *AuthAPI) login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.RespondError(c, http.StatusBadRequest, "invalid request", "bad_request")
		return
	}

	user, err := a.store.GetUserByEmail(c.Request.Context(), strings.ToLower(strings.TrimSpace(req.Email)))
	if err != nil {
		api.RespondInvalidCredentials(c)
		return
	}
	if err := auth2.CheckPassword(req.Password, user.PasswordHash); err != nil {
		metrics.AuthEventsTotal.WithLabelValues("login", "failure").Inc()
		api.RespondInvalidCredentials(c)
		return
	}

	now := time.Now().UTC()
	refreshToken, refreshHash, err := a.tokens.GenerateRefreshToken()
	if err != nil {
		api.RespondError(c, http.StatusInternalServerError, "failed to create session", "internal_error")
		return
	}
	session, err := a.store.CreateSession(c.Request.Context(), models.CreateSessionParams{
		UserID:           user.ID,
		RefreshTokenHash: refreshHash,
		IP:               c.ClientIP(),
		UserAgent:        c.Request.UserAgent(),
		ExpiresAt:        a.tokens.RefreshTokenExpiry(now),
		LastUsedAt:       now,
	})
	if err != nil {
		api.RespondError(c, http.StatusInternalServerError, "failed to create session", "internal_error")
		return
	}
	if err := a.store.UpdateUserLastLogin(c.Request.Context(), user.ID, now); err == nil {
		user.LastLoginAt = &now
		user.UpdatedAt = now
	}
	accessToken, expiresAt, err := a.tokens.GenerateAccessToken(user.ID, session.ID, user.Role, now)
	if err != nil {
		api.RespondError(c, http.StatusInternalServerError, "failed to issue tokens", "internal_error")
		return
	}

	metrics.AuthEventsTotal.WithLabelValues("login", "success").Inc()

	c.JSON(http.StatusOK, authSessionResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
		User:         shared.ToUserResponse(user),
	})
}
