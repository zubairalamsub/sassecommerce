package metrics

import (
	"errors"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// counterValue reads one counter series straight from the default gatherer.
//
// Deliberately not prometheus/client_golang's testutil: importing it here adds
// a test-only dependency to go.mod, and shared/go is consumed by all fourteen
// services, so its dependency surface is everyone's problem.
func counterValue(t *testing.T, name string, want map[string]string) float64 {
	t.Helper()
	mfs, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	for _, mf := range mfs {
		if mf.GetName() != name {
			continue
		}
		for _, m := range mf.GetMetric() {
			labels := map[string]string{}
			for _, lp := range m.GetLabel() {
				labels[lp.GetName()] = lp.GetValue()
			}
			match := true
			for k, v := range want {
				if labels[k] != v {
					match = false
					break
				}
			}
			if !match {
				continue
			}
			if c := m.GetCounter(); c != nil {
				return c.GetValue()
			}
			if g := m.GetGauge(); g != nil {
				return g.GetValue()
			}
		}
	}
	return 0
}

func metricExists(t *testing.T, name string) bool {
	t.Helper()
	mfs, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	for _, mf := range mfs {
		if mf.GetName() == name {
			return mf.GetHelp() != ""
		}
	}
	return false
}

// The failure these metrics exist for: a consumer that reads every message and
// drops it looks healthy at the broker, because the offset is committed either
// way and lag stays at zero. So the drop counter has to move when nothing else
// does.
func TestEventDroppedIsCountedByReason(t *testing.T) {
	RegisterEvents("test-service")
	lbl := map[string]string{"topic": "order-events", "event_type": "OrderShipped", "reason": ReasonDecodeError}

	before := counterValue(t, "kafka_events_failed_total", lbl)
	EventDropped("test-service", "order-events", "OrderShipped", ReasonDecodeError)
	EventDropped("test-service", "order-events", "OrderShipped", ReasonDecodeError)

	if got := counterValue(t, "kafka_events_failed_total", lbl) - before; got != 2 {
		t.Errorf("drops counted = %v, want 2", got)
	}

	// A different reason must be its own series, or "why" is unanswerable.
	other := counterValue(t, "kafka_events_failed_total",
		map[string]string{"topic": "order-events", "event_type": "OrderShipped", "reason": ReasonHandlerError})
	if other != 0 {
		t.Errorf("handler_error = %v, want 0; reasons are bleeding into each other", other)
	}
}

// A dropped event must not refresh the last-processed clock. A topic whose
// events are all being discarded should read as stale, because it is.
func TestEventDroppedDoesNotRefreshLastProcessed(t *testing.T) {
	RegisterEvents("test-service")
	lbl := map[string]string{"topic": "stale-topic"}

	EventConsumed("test-service", "stale-topic", "Thing", time.Millisecond)
	afterConsume := counterValue(t, "kafka_event_last_processed_timestamp_seconds", lbl)
	if afterConsume == 0 {
		t.Fatal("last-processed timestamp not set by a successful consume")
	}

	EventDropped("test-service", "stale-topic", "Thing", ReasonDecodeError)

	if got := counterValue(t, "kafka_event_last_processed_timestamp_seconds", lbl); got != afterConsume {
		t.Errorf("last-processed moved on a drop (%v -> %v); a fully-broken topic would look fresh", afterConsume, got)
	}
}

func TestEventConsumedRecordsCountAndDuration(t *testing.T) {
	RegisterEvents("test-service")
	lbl := map[string]string{"topic": "user-events", "event_type": "UserRegistered"}

	before := counterValue(t, "kafka_events_consumed_total", lbl)
	EventConsumed("test-service", "user-events", "UserRegistered", 250*time.Millisecond)

	if got := counterValue(t, "kafka_events_consumed_total", lbl) - before; got != 1 {
		t.Errorf("consumed = %v, want 1 more", got)
	}
	if !metricExists(t, "kafka_event_processing_duration_seconds") {
		t.Error("no duration histogram exposed")
	}
}

func TestEventPublishedSplitsSuccessFromError(t *testing.T) {
	RegisterEvents("test-service")

	EventPublished("test-service", "order-events", "OrderCreated", nil)
	EventPublished("test-service", "order-events", "OrderCreated", errors.New("broker unreachable"))

	ok := counterValue(t, "kafka_events_published_total",
		map[string]string{"topic": "order-events", "event_type": "OrderCreated", "outcome": "success"})
	bad := counterValue(t, "kafka_events_published_total",
		map[string]string{"topic": "order-events", "event_type": "OrderCreated", "outcome": "error"})
	if ok < 1 || bad < 1 {
		t.Errorf("success=%v error=%v, want at least 1 of each", ok, bad)
	}
}

// An envelope that fails to decode may not yield an event type at all. The
// series still has to exist, or the drop is invisible.
func TestUnknownEventTypeStillRecorded(t *testing.T) {
	RegisterEvents("test-service")

	EventDropped("test-service", "order-events", "", ReasonDecodeError)

	got := counterValue(t, "kafka_events_failed_total",
		map[string]string{"topic": "order-events", "event_type": "unknown", "reason": ReasonDecodeError})
	if got < 1 {
		t.Error("an empty event type was not recorded under \"unknown\"")
	}
}

// The dashboard and the alert rules key on these names. Pin them here rather
// than discovering a rename in production.
func TestExposedMetricNames(t *testing.T) {
	RegisterEvents("test-service")
	EventConsumed("test-service", "t", "E", time.Millisecond)
	EventDropped("test-service", "t", "E", ReasonHandlerError)
	EventPublished("test-service", "t", "E", nil)

	for _, name := range []string{
		"kafka_events_consumed_total",
		"kafka_events_failed_total",
		"kafka_events_published_total",
		"kafka_event_processing_duration_seconds",
		"kafka_event_last_processed_timestamp_seconds",
	} {
		if !metricExists(t, name) {
			t.Errorf("metric %s is not exposed with help text; the dashboard and alerts reference it", name)
		}
	}
}

// Register() must bring the event metrics up with it. Without this, a healthy
// consumer exposes no kafka_events_* families until the first event happens to
// arrive — which reads exactly like a service nobody instrumented, and leaves
// the alert rules with no series to evaluate.
func TestRegisterAlsoRegistersEventMetrics(t *testing.T) {
	Register("test-service")

	for _, name := range []string{
		"kafka_events_consumed_total",
		"kafka_events_failed_total",
		"kafka_events_published_total",
	} {
		if !metricExists(t, name) {
			t.Errorf("%s is not exposed after Register(); a healthy service would look uninstrumented", name)
		}
	}
}

// InitTopic has to create the series, not merely register the collectors. A
// CounterVec emits nothing at all until a label combination is touched, so
// without this a healthy consumer is indistinguishable from an uninstrumented
// one, and the staleness gauge an alert reads would not exist.
func TestInitTopicCreatesZeroSeries(t *testing.T) {
	InitTopic("test-service", "fresh-topic")

	for _, reason := range []string{ReasonDecodeError, ReasonSignatureInvalid, ReasonHandlerError} {
		lbl := map[string]string{"topic": "fresh-topic", "event_type": "unknown", "reason": reason}
		if !seriesPresent(t, "kafka_events_failed_total", lbl) {
			t.Errorf("no zero-valued drop series for reason=%s; the panel would read \"No data\"", reason)
		}
	}
	if counterValue(t, "kafka_event_last_processed_timestamp_seconds", map[string]string{"topic": "fresh-topic"}) == 0 {
		t.Error("staleness gauge not seeded; a consumer that processes nothing would have no series to alert on")
	}
}

func seriesPresent(t *testing.T, name string, want map[string]string) bool {
	t.Helper()
	mfs, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	for _, mf := range mfs {
		if mf.GetName() != name {
			continue
		}
		for _, m := range mf.GetMetric() {
			labels := map[string]string{}
			for _, lp := range m.GetLabel() {
				labels[lp.GetName()] = lp.GetValue()
			}
			ok := true
			for k, v := range want {
				if labels[k] != v {
					ok = false
					break
				}
			}
			if ok {
				return true
			}
		}
	}
	return false
}
