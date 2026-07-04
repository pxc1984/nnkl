package store

import (
	"context"
	"testing"
	"time"

	"github.com/pxc1984/nnkl-backend/store/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInMemoryStoreAuthFlow(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryStore()

	user, err := store.CreateUser(ctx, models.CreateUserParams{
		Email:        "user@example.com",
		Name:         "User",
		Role:         "admin",
		PasswordHash: "hashed",
	})
	require.NoError(t, err)

	fetchedUser, err := store.GetUserByEmail(ctx, "user@example.com")
	require.NoError(t, err)
	assert.Equal(t, user.ID, fetchedUser.ID)

	now := time.Now().UTC()
	session, err := store.CreateSession(ctx, models.CreateSessionParams{
		UserID:           user.ID,
		RefreshTokenHash: "hash-1",
		IP:               "127.0.0.1",
		UserAgent:        "test",
		ExpiresAt:        now.Add(time.Hour),
		LastUsedAt:       now,
	})
	require.NoError(t, err)

	rotatedSession, err := store.UpdateSessionToken(ctx, models.UpdateSessionTokenParams{
		SessionID:        session.ID,
		RefreshTokenHash: "hash-2",
		ExpiresAt:        now.Add(2 * time.Hour),
		LastUsedAt:       now.Add(time.Minute),
	})
	require.NoError(t, err)
	assert.Equal(t, "hash-2", rotatedSession.RefreshTokenHash)

	sessions, err := store.ListUserSessions(ctx, user.ID)
	require.NoError(t, err)
	assert.Len(t, sessions, 1)

	assert.NoError(t, store.DeleteSessionByUserAndHash(ctx, user.ID, "hash-2"))
	sessions, err = store.ListUserSessions(ctx, user.ID)
	require.NoError(t, err)
	assert.Len(t, sessions, 0)
}
