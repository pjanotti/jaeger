package sender

import "github.com/uber/jaeger-lib/metrics"

type senderMetrics struct {
	// Number of batches incoming to sender
	BatchesIncoming metrics.Counter `metric:"batches.incoming"`

	// Number of successful batches sent
	BatchesSent metrics.Counter `metric:"batches.sent"`

	// Number of failures of sending batches
	BatchesFailedToSend metrics.Counter `metric:"batches.failedToSend"`

	// Number of spans in batches incoming
	SpansIncoming metrics.Counter `metric:"spans.incoming"`

	// Number of successful spans sent
	SpansSent metrics.Counter `metric:"spans.sent"`

	// Number of failures of sending spans
	SpansFailedToSend metrics.Counter `metric:"spans.failedToSend"`
}
