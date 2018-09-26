package processor

import "github.com/uber/jaeger-lib/metrics"

type processorBatchMetrics struct {
	// Number of batches incoming to processor
	BatchesIncoming metrics.Counter `metric:"batches.incoming"`

	// Number of batches put into queue
	BatchesEnqueued metrics.Counter `metric:"batches.enqueued"`

	// Number of batches put into queue
	BatchesFailedToEnqueue metrics.Counter `metric:"batches.failed.enqueue"`

	// Number of attempts to process batches
	BatchesProcessingTotalAttempts metrics.Counter `metric:"batches.processing.attempts.total"`

	// Number of successful attempts to process batches
	BatchesProcessingSuccessfulAttempts metrics.Counter `metric:"batches.processing.attempts.success"`

	// Number of failed attempts to process batches
	BatchesProcessingFailedAttempts metrics.Counter `metric:"batches.processing.attempts.fail"`

	// Number of batches that were dropped by cleanup policy
	BatchesDropped metrics.Counter `metric:"batches.dropped"`
}
