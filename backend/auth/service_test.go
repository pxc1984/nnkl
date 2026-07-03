package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPasswordHashing(t *testing.T) {
	hash, err := HashPassword("supersecret")
	require.NoError(t, err)
	assert.NotEqual(t, "supersecret", hash)
	assert.NoError(t, CheckPassword("supersecret", hash))
	assert.Error(t, CheckPassword("wrong", hash))
}

func TestAccessTokenRoundTrip(t *testing.T) {
	manager := NewManager("test-secret", time.Minute, time.Hour)
	now := time.Now().UTC()
	token, expiresAt, err := manager.GenerateAccessToken("user-1", "session-1", "admin", now)
	require.NoError(t, err)
	assert.Equal(t, now.Add(time.Minute), expiresAt)

	claims, err := manager.ParseAccessToken(token)
	require.NoError(t, err)
	assert.Equal(t, "user-1", claims.Subject)
	assert.Equal(t, "session-1", claims.SessionID)
	assert.Equal(t, "admin", claims.Role)
}
