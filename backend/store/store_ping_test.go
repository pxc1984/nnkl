package store

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInMemoryStore_Ping(t *testing.T) {
	store := NewInMemoryStore()
	assert.NotNil(t, store)
	assert.Nil(t, store.Ping(context.Background()))
}

func TestPostgresStore_Ping(t *testing.T) {
	t.Skip() // кому не лень поднимать постгрю для тест кейсов для постгри) это хакатон after all
	store, err := NewPostgresStore()
	assert.Nil(t, err)
	assert.NotNil(t, store)
	assert.Nil(t, store.Ping(context.Background()))
}
