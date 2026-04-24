package service

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// MockKafkaPublisher is a mock implementation of KafkaPublisher for tests
type MockKafkaPublisher struct {
	mock.Mock
}

func (m *MockKafkaPublisher) Publish(ctx context.Context, topic, key string, value []byte) error {
	args := m.Called(ctx, topic, key, value)
	return args.Error(0)
}
