package messaging

import (
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"testing"

	"github.com/segmentio/kafka-go"
	metadataAPI "github.com/segmentio/kafka-go/protocol/metadata"
	"github.com/sirupsen/logrus"
)

// fakeMetadataTransport answers a kafka.Writer's partition-count lookup
// in-process, with no socket and no broker involved. WriteMessages resolves
// this before it does anything else -- even for an Async writer, since the
// Async short-circuit only happens after partitions are resolved -- so this
// alone is enough to drive Publish's success and failure branches
// deterministically without touching the network.
//
// Any other request type (chiefly Produce, which an Async writer issues on a
// background goroutine after WriteMessages has already returned to the
// caller) is answered with an error rather than a type-asserted zero value,
// so that goroutine fails fast instead of retrying against a nonexistent
// broker for several seconds during Close.
type fakeMetadataTransport struct {
	err error
}

func (f *fakeMetadataTransport) RoundTrip(_ context.Context, _ net.Addr, req kafka.Request) (kafka.Response, error) {
	mreq, ok := req.(*metadataAPI.Request)
	if !ok {
		return nil, errors.New("fakeMetadataTransport: only metadata requests are supported")
	}
	if f.err != nil {
		return nil, f.err
	}
	topics := make([]metadataAPI.ResponseTopic, len(mreq.TopicNames))
	for i, name := range mreq.TopicNames {
		topics[i] = metadataAPI.ResponseTopic{
			Name:       name,
			Partitions: []metadataAPI.ResponsePartition{{PartitionIndex: 0, LeaderID: 0}},
		}
	}
	return &metadataAPI.Response{Topics: topics}, nil
}

// newDiscardLogger is shared by this package's test files.
func newDiscardLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	return logger
}

func TestNewProducer_ConfiguresAsyncWriterWithLeastBytesBalancingAndCompletionLogging(t *testing.T) {
	p := NewProducer([]string{"localhost:9092"}, newDiscardLogger())
	defer func() { _ = p.Close() }()

	if !p.writer.Async {
		t.Error("writer is synchronous -- Publish would block the request handler on the broker round-trip")
	}
	if _, ok := p.writer.Balancer.(*kafka.LeastBytes); !ok {
		t.Errorf("balancer = %T, want *kafka.LeastBytes", p.writer.Balancer)
	}
	if p.writer.Completion == nil {
		t.Error("no Completion callback -- async publish failures on this fire-and-forget path would be invisible")
	}
}

func TestProducer_Close_OnAProducerThatNeverPublishedIsSafe(t *testing.T) {
	p := NewProducer([]string{"localhost:9092"}, newDiscardLogger())

	if err := p.Close(); err != nil {
		t.Errorf("Close() on an unused producer = %v, want nil", err)
	}
}

func TestProducer_Publish_WrapsTheUnderlyingTransportFailure(t *testing.T) {
	p := NewProducer([]string{"localhost:9092"}, newDiscardLogger())
	p.writer.Transport = &fakeMetadataTransport{err: errors.New("simulated broker unreachable")}
	defer func() { _ = p.Close() }()

	err := p.Publish(context.Background(), "product-events", "key-1", []byte(`{"event_type":"ProductCreated"}`))

	if err == nil {
		t.Fatal("Publish returned nil despite the broker being unreachable, want a wrapped error")
	}
	if !strings.Contains(err.Error(), "failed to publish message") {
		t.Errorf("error = %q, want it wrapped with %q so callers can distinguish publish failures from other errors", err.Error(), "failed to publish message")
	}
	if !strings.Contains(err.Error(), "simulated broker unreachable") {
		t.Errorf("error = %q, want the underlying transport error preserved for diagnostics", err.Error())
	}
}
