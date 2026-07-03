package auth

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pxc1984/nnkl-backend/api"
	auth2 "github.com/pxc1984/nnkl-backend/auth"
	"github.com/pxc1984/nnkl-backend/store"
)

func (a *AuthAPI) refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.RespondError(c, http.StatusBadRequest, "invalid request", "bad_request")
		return
	}

	session, err := a.store.GetSessionByRefreshTokenHash(c.Request.Context(), auth2.HashToken(req.RefreshToken))
	if err != nil || session.ExpiresAt.Before(time.Now().UTC()) {
		api.RespondError(c, http.StatusUnauthorized, "invalid refresh token", "unauthorized")
		return
	}

	now := time.Now().UTC()
	refreshToken, refreshHash, err := a.tokens.GenerateRefreshToken()
	if err != nil {
		api.RespondError(c, http.StatusInternalServerError, "failed to issue tokens", "internal_error")
		return
	}
	session, err = a.store.UpdateSessionToken(c.Request.Context(), store.UpdateSessionTokenParams{
		SessionID:        session.ID,
		RefreshTokenHash: refreshHash,
		ExpiresAt:        a.tokens.RefreshTokenExpiry(now),
		LastUsedAt:       now,
	})
	if err != nil {
		api.RespondError(c, http.StatusInternalServerError, "failed to rotate session", "internal_error")
		return
	}
	accessToken, expiresAt, err := a.tokens.GenerateAccessToken(session.UserID, session.ID, session.User.Role, now)
	if err != nil {
		api.RespondError(c, http.StatusInternalServerError, "failed to issue tokens", "internal_error")
		return
	}

	c.JSON(http.StatusOK, tokenPairResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
	})
}
