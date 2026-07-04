package auth

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pxc1984/nnkl-backend/api"
	shared "github.com/pxc1984/nnkl-backend/api/v1/shared"
	auth2 "github.com/pxc1984/nnkl-backend/auth"
	"github.com/pxc1984/nnkl-backend/store/models"
	"gorm.io/gorm"
)

func (a *AuthAPI) register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.RespondError(c, http.StatusBadRequest, "invalid request", "bad_request")
		return
	}

	count, err := a.store.CountUsers(c.Request.Context())
	if err != nil {
		api.RespondError(c, http.StatusInternalServerError, "failed to inspect users", "internal_error")
		return
	}

	role := "guest"
	if count == 0 {
		role = "admin"
	}

	passwordHash, err := auth2.HashPassword(req.Password)
	if err != nil {
		api.RespondError(c, http.StatusInternalServerError, "failed to hash password", "internal_error")
		return
	}

	user, err := a.store.CreateUser(c.Request.Context(), models.CreateUserParams{
		Email:        strings.ToLower(strings.TrimSpace(req.Email)),
		Name:         strings.TrimSpace(req.Name),
		Role:         role,
		PasswordHash: passwordHash,
	})
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			api.RespondError(c, http.StatusConflict, "user already exists", "conflict")
			return
		}
		api.RespondError(c, http.StatusInternalServerError, "failed to create user", "internal_error")
		return
	}

	c.JSON(http.StatusCreated, shared.ToUserResponse(user))
}
