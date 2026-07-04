package auth

import (
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pxc1984/nnkl-backend/api"
	shared "github.com/pxc1984/nnkl-backend/api/v1/shared"
	"github.com/pxc1984/nnkl-backend/store/models"
)

func (a *AuthAPI) me(c *gin.Context) {
	user, _ := api.CurrentUserFromContext(c)
	c.JSON(http.StatusOK, shared.ToUserResponse(user))
}

func (a *AuthAPI) updateMe(c *gin.Context) {
	user, _ := api.CurrentUserFromContext(c)

	name := strings.TrimSpace(c.PostForm("name"))

	avatarFile, avatarHeader, err := c.Request.FormFile("avatar")
	isAvatarUpload := err == nil

	var params models.UpdateUserParams
	if name != "" && name != user.Name {
		params.Name = &name
	}
	if isAvatarUpload {
		defer avatarFile.Close()
		data := make([]byte, avatarHeader.Size)
		if _, err := avatarFile.Read(data); err != nil {
			api.RespondError(c, http.StatusBadRequest, "failed to read avatar file", "bad_request")
			return
		}
		// Basic validation: reject files larger than 5 MB.
		if len(data) > 5*1024*1024 {
			api.RespondError(c, http.StatusRequestEntityTooLarge, "avatar must be less than 5 MB", "payload_too_large")
			return
		}
		// Only allow common image types.
		ext := strings.ToLower(filepath.Ext(avatarHeader.Filename))
		switch ext {
		case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		default:
			api.RespondError(c, http.StatusBadRequest, "unsupported avatar format (allowed: jpg, png, gif, webp)", "bad_request")
			return
		}
		params.AvatarData = data
		// Clear external URL when uploading a local avatar.
		emptyURL := ""
		params.AvatarURL = &emptyURL
	} else if c.Request.MultipartForm != nil && c.Request.MultipartForm.File != nil {
		// Non-file field error — avatar field exists but failed to read.
		if _, ok := c.Request.MultipartForm.File["avatar"]; ok {
			if err != nil {
				api.RespondError(c, http.StatusBadRequest, "failed to read avatar file", "bad_request")
				return
			}
		}
	}

	// Nothing to update.
	if params.Name == nil && params.AvatarData == nil {
		c.JSON(http.StatusOK, shared.ToUserResponse(user))
		return
	}

	updated, err := a.store.UpdateUser(c.Request.Context(), user.ID, params)
	if err != nil {
		api.RespondError(c, http.StatusInternalServerError, "failed to update profile", "internal_error")
		return
	}

	c.JSON(http.StatusOK, shared.ToUserResponse(updated))
}

func (a *AuthAPI) getAvatar(c *gin.Context) {
	user, _ := api.CurrentUserFromContext(c)
	serveAvatarBytes(c, user)
}

// getUserAvatar is public (no auth) — browser <img> tags can load it.
// URL: GET /api/v1/user/:id/avatar
func (a *AuthAPI) getUserAvatar(c *gin.Context) {
	user, err := a.store.GetUserByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		api.RespondError(c, http.StatusNotFound, "user not found", "not_found")
		return
	}
	serveAvatarBytes(c, user)
}

func serveAvatarBytes(c *gin.Context, user *models.User) {
	if user == nil || len(user.AvatarData) == 0 {
		api.RespondError(c, http.StatusNotFound, "no avatar set", "not_found")
		return
	}

	contentType := mime.TypeByExtension(filepath.Ext(c.Query("ext")))
	if contentType == "" {
		contentType = http.DetectContentType(user.AvatarData)
	}
	c.Data(http.StatusOK, contentType, user.AvatarData)
}
