package sender

import "github.com/uber/jaeger-lib/metrics"

type senderMetrics struct {
	// Number of batches incoming to sender
	BatchesRequestToSend metrics.Counter `metric:"batches.requestToSend"`

	// Number of successful batches sent
	BatchesSent metrics.Counter `metric:"batches.sent"`

	// Number of failures of sending batches
	BatchesFailedToSend metrics.Counter `metric:"batches.failedToSend"`

	// Number of spans in batches incoming
	SpansRequestToSend metrics.Counter `metric:"spans.requestToSend"`

	// Number of successful spans sent
	SpansSent metrics.Counter `metric:"spans.sent"`

	// Number of failures of sending spans
	SpansFailedToSend metrics.Counter `metric:"spans.failedToSend"`
}
