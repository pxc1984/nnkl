package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pxc1984/nnkl-backend/auth"
	"github.com/pxc1984/nnkl-backend/store"
	"github.com/pxc1984/nnkl-backend/store/models"
)

func AuthenticateRequest(c *gin.Context, st store.Store, tokens *auth.Manager) (*models.User, bool) {
	claims, session, ok := authenticateSession(c, st, tokens)
	if !ok {
		return nil, false
	}
	SetAuthenticatedRequest(c, claims, &session.User)
	return &session.User, true
}

func RequireAuth(st store.Store, tokens *auth.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, session, ok := authenticateSession(c, st, tokens)
		if !ok {
			c.Abort()
			return
		}

		now := time.Now().UTC()
		if err := st.TouchSession(c.Request.Context(), session.ID, now); err == nil {
			session.LastUsedAt = now
		}

		SetAuthenticatedRequest(c, claims, &session.User)
		c.Next()
	}
}

func authenticateSession(c *gin.Context, st store.Store, tokens *auth.Manager) (*auth.AccessClaims, *models.Session, bool) {
	authorization := strings.TrimSpace(c.GetHeader("Authorization"))
	if !strings.HasPrefix(authorization, "Bearer ") {
		RespondError(c, http.StatusUnauthorized, "missing bearer token", "unauthorized")
		return nil, nil, false
	}

	token := strings.TrimSpace(strings.TrimPrefix(authorization, "Bearer "))
	claims, err := tokens.ParseAccessToken(token)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "invalid access token", "unauthorized")
		return nil, nil, false
	}

	session, err := st.GetSessionByID(c.Request.Context(), claims.SessionID)
	if err != nil || session.UserID != claims.Subject || session.ExpiresAt.Before(time.Now().UTC()) {
		RespondError(c, http.StatusUnauthorized, "session expired or revoked", "unauthorized")
		return nil, nil, false
	}

	return claims, session, true
}
