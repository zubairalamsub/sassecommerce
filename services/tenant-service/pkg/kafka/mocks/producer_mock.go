package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
)

type MockKafkaProducer struct {
	mock.Mock
}

func (m *MockKafkaProducer) Publish(ctx context.Context, topic, key string, value []byte) error {
	args := m.Called(ctx, topic, key, value)
	return args.Error(0)
}

func (m *MockKafkaProducer) Close() error {
	args := m.Called()
	return args.Error(0)
}
