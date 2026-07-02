package store

import "context"

type InMemoryStore struct{}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{}
}

func (s *InMemoryStore) Backend() string {
	return "memory"
}

func (s *InMemoryStore) Ping(context.Context) error {
	return nil
}

func (s *InMemoryStore) Close() error {
	return nil
}
