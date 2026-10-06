package metrics

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Event-pipeline instrumentation.
//
// The Kafka exporter already answers "is the broker up, and is anyone behind".
// It cannot answer the question that actually cost this platform an outage:
// is a consumer reading messages and then throwing them away?
//
// Every consumer here commits the offset whether processing succeeded or not
// — a failure is logged and the loop moves on. That is a deliberate choice
// (it stops one poison message blocking a partition forever) but it means a
// consumer that fails on *every* message looks identical to a healthy one
// from outside: it reads, it commits, lag stays at zero, throughput is
// normal. When the `version` field on the event envelope changed shape, every
// order event failed to decode for weeks and no broker-level metric moved.
//
// So these metrics count what the broker cannot see: how many events were
// acted on, how many were dropped, and why.
//
// Cardinality is bounded by design. `topic` and `event_type` are both closed
// sets fixed in code, and `reason` is the constants below — no ids, no tenant,
// nothing user-supplied.
//
// `service` is a ConstLabel here, matching the HTTP metrics in this package.
// prometheus.yml also attaches `service` as a target label on every scrape
// job, and with honor_labels unset Prometheus renames the exposed copy to
// `exported_service` on collision. The values are identical, so this is
// redundancy rather than a conflict, and queries should group by the target
// label. Kept for consistency with the rest of the package — dropping it
// would mean changing the HTTP metrics too.

// Reasons an event was read but not acted on. Use these rather than free
// strings; the label is only bounded because the set is.
const (
	// ReasonDecodeError — the envelope or payload would not unmarshal. This
	// is the one that caught the version-field outage.
	ReasonDecodeError = "decode_error"
	// ReasonSignatureInvalid — HMAC verification failed, so the event was
	// spoofed, tampered with, or signed with a key this service does not have.
	ReasonSignatureInvalid = "signature_invalid"
	// ReasonUnknownEventType — a well-formed event this service has no handler
	// for. Usually benign, but a spike means a producer shipped a new event
	// type before its consumer.
	ReasonUnknownEventType = "unknown_event_type"
	// ReasonMissingField — decoded, but a field the handler needs was absent
	// or empty, so it returned early.
	ReasonMissingField = "missing_field"
	// ReasonHandlerError — the handler ran and failed. Downstream problem
	// rather than a message problem.
	ReasonHandlerError = "handler_error"
)

var (
	eventRegisterOnce sync.Once

	eventsConsumedTotal  *prometheus.CounterVec
	eventsFailedTotal    *prometheus.CounterVec
	eventsPublishedTotal *prometheus.CounterVec
	eventProcessingDur   *prometheus.HistogramVec
	eventLastProcessedTS *prometheus.GaugeVec
)

// eventBuckets are wider than the HTTP ones: an event handler often writes to
// a database and calls another service, so the interesting tail is seconds,
// not milliseconds.
var eventBuckets = []float64{0.005, 0.025, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30}

// RegisterEvents initialises the event-pipeline metrics for a service. Safe to
// call more than once; later calls are no-ops. The Event* helpers call it, so
// forgetting it at startup costs nothing.
func RegisterEvents(service string) {
	eventRegisterOnce.Do(func() {
		labels := prometheus.Labels{"service": service}

		eventsConsumedTotal = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name:        "kafka_events_consumed_total",
				Help:        "Events read from Kafka and handled successfully.",
				ConstLabels: labels,
			},
			[]string{"topic", "event_type"},
		)

		eventsFailedTotal = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "kafka_events_failed_total",
				Help: "Events read from Kafka but not acted on, by reason. " +
					"The offset is committed either way, so this is invisible to consumer lag.",
				ConstLabels: labels,
			},
			[]string{"topic", "event_type", "reason"},
		)

		eventsPublishedTotal = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name:        "kafka_events_published_total",
				Help:        "Publish attempts to Kafka, by outcome (success|error).",
				ConstLabels: labels,
			},
			[]string{"topic", "event_type", "outcome"},
		)

		eventProcessingDur = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:        "kafka_event_processing_duration_seconds",
				Help:        "Time spent handling one event, excluding the fetch.",
				Buckets:     eventBuckets,
				ConstLabels: labels,
			},
			[]string{"topic", "event_type"},
		)

		eventLastProcessedTS = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "kafka_event_last_processed_timestamp_seconds",
				Help: "Unix time this service last handled an event on the topic. " +
					"Staleness on a normally-busy topic is the signal a counter rate cannot give you.",
				ConstLabels: labels,
			},
			[]string{"topic"},
		)

		prometheus.MustRegister(
			eventsConsumedTotal,
			eventsFailedTotal,
			eventsPublishedTotal,
			eventProcessingDur,
			eventLastProcessedTS,
		)
	})
}

// EventConsumed records one event handled successfully, and how long it took.
//
// eventType may be empty when the envelope failed to decode far enough to know
// it; it is normalised to "unknown" so the series still exists.
func EventConsumed(service, topic, eventType string, d time.Duration) {
	RegisterEvents(service)
	eventType = orUnknown(eventType)

	eventsConsumedTotal.WithLabelValues(topic, eventType).Inc()
	eventProcessingDur.WithLabelValues(topic, eventType).Observe(d.Seconds())
	eventLastProcessedTS.WithLabelValues(topic).Set(float64(time.Now().Unix()))
}

// EventDropped records an event that was read from the broker and not acted
// on. reason should be one of the Reason* constants.
//
// This is the counter to alert on. Because the consume loops commit the offset
// regardless of outcome, a service dropping every message it receives is
// indistinguishable from a healthy one at the broker: lag stays at zero.
func EventDropped(service, topic, eventType, reason string) {
	RegisterEvents(service)
	eventsFailedTotal.WithLabelValues(topic, orUnknown(eventType), reason).Inc()
	// Deliberately does not touch eventLastProcessedTS. A topic whose events
	// are all being dropped should look stale, because in every sense that
	// matters it is.
}

// EventPublished records a publish attempt. Pass the error from the produce
// call, or nil.
func EventPublished(service, topic, eventType string, err error) {
	RegisterEvents(service)
	outcome := "success"
	if err != nil {
		outcome = "error"
	}
	eventsPublishedTotal.WithLabelValues(topic, orUnknown(eventType), outcome).Inc()
}

func orUnknown(eventType string) string {
	if eventType == "" {
		return "unknown"
	}
	return eventType
}

// InitTopic pre-creates the zero-valued drop series for a topic, so a consumer
// that has never dropped anything reports 0 rather than nothing at all.
//
// This matters because a CounterVec emits no output whatsoever until one of
// its label combinations has been touched — not even a HELP line. Without
// this, a dashboard cannot tell a healthy consumer from one that was never
// instrumented: both render as "No data".
//
// Call it once per topic at consumer construction. Only the drop counters are
// seeded; kafka_events_consumed_total appears on its own the moment real
// traffic arrives, and a zero there would be indistinguishable from a topic
// nobody publishes to.
func InitTopic(service, topic string) {
	RegisterEvents(service)
	for _, reason := range []string{
		ReasonDecodeError,
		ReasonSignatureInvalid,
		ReasonUnknownEventType,
		ReasonMissingField,
		ReasonHandlerError,
	} {
		eventsFailedTotal.WithLabelValues(topic, "unknown", reason).Add(0)
	}

	// Seed the staleness gauge with the subscribe time.
	//
	// A GaugeVec child does not exist until something Sets it, so without
	// this a consumer that comes up and then processes nothing at all leaves
	// the series absent — and an alert shaped like
	// `time() - kafka_event_last_processed_timestamp_seconds > N` cannot fire
	// on a series that is not there. The silent case would be the one it
	// missed. Seeding at subscribe starts the clock immediately.
	eventLastProcessedTS.WithLabelValues(topic).Set(float64(time.Now().Unix()))
}
