package processor

import (
	"time"

	cApp "github.com/jaegertracing/jaeger/cmd/collector/app"
	"github.com/jaegertracing/jaeger/model"
	"github.com/jaegertracing/jaeger/pkg/queue"
	"github.com/uber/jaeger-lib/metrics"
	"go.uber.org/zap"
)

type queuedSpanProcessor struct {
	queue                    *queue.BoundedQueue
	metrics                  *cApp.SpanProcessorMetrics
	batchMetrics             *processorBatchMetrics
	logger                   *zap.Logger
	sender                   cApp.SpanProcessor
	numWorkers               int
	retryOnProcessingFailure bool
	backoffDelay             time.Duration
	stopCh                   chan struct{}
}

type queueItem struct {
	queuedTime time.Time
	mSpans     []*model.Span
	spanFormat string
}

// NewQueuedSpanProcessor returns a span processor that maintains a bounded
// in-memory queue of span batches, and sends out span batches using the
// provided sender
func NewQueuedSpanProcessor(
	sender cApp.SpanProcessor,
	opts ...Option,
) cApp.SpanProcessor {
	sp := newQueuedSpanProcessor(sender, opts...)

	sp.queue.StartConsumers(sp.numWorkers, func(item interface{}) {
		value := item.(*queueItem)
		sp.processItemFromQueue(value)
	})

	sp.queue.StartLengthReporting(1*time.Second, sp.metrics.QueueLength)

	return sp
}

func newQueuedSpanProcessor(
	sender cApp.SpanProcessor,
	opts ...Option,
) *queuedSpanProcessor {
	options := Options.apply(opts...)
	handlerMetrics := cApp.NewSpanProcessorMetrics(
		options.serviceMetrics,
		options.hostMetrics,
		options.extraFormatTypes)
	bm := &processorBatchMetrics{}
	metrics.Init(bm, options.serviceMetrics.Namespace("queued-processor", nil), nil)
	droppedItemHandler := func(item interface{}) {
		batchItem := item.(queueItem)
		handlerMetrics.SpansDropped.Inc(int64(len(batchItem.mSpans)))
		bm.BatchesDropped.Inc(1)
	}
	boundedQueue := queue.NewBoundedQueue(options.queueSize, droppedItemHandler)
	return &queuedSpanProcessor{
		queue:                    boundedQueue,
		metrics:                  handlerMetrics,
		batchMetrics:             bm,
		logger:                   options.logger,
		numWorkers:               options.numWorkers,
		sender:                   sender,
		retryOnProcessingFailure: options.retryOnProcessingFailure,
		backoffDelay:             options.backoffDelay,
		stopCh:                   make(chan struct{}),
	}
}

// Stop halts the span processor and all its go-routines.
func (sp *queuedSpanProcessor) Stop() {
	close(sp.stopCh)
	sp.queue.Stop()
}

// ProcessSpans implements the SpanProcessor interface
func (sp *queuedSpanProcessor) ProcessSpans(mSpans []*model.Span, spanFormat string) ([]bool, error) {
	sp.batchMetrics.BatchesIncoming.Inc(1)
	sp.metrics.BatchSize.Update(int64(len(mSpans)))
	retMe := make([]bool, len(mSpans))
	ok := sp.enqueueSpanBatch(mSpans, spanFormat)
	for i := range mSpans {
		retMe[i] = ok
	}
	return retMe, nil
}

func (sp *queuedSpanProcessor) enqueueSpanBatch(mSpans []*model.Span, spanFormat string) bool {
	spanCounts := sp.metrics.GetCountsForFormat(spanFormat)
	for _, mSpan := range mSpans {
		spanCounts.ReceivedBySvc.ReportServiceNameForSpan(mSpan)
	}

	item := &queueItem{
		queuedTime: time.Now(),
		mSpans:     mSpans,
		spanFormat: spanFormat,
	}
	addedToQueue := sp.queue.Produce(item)
	if !addedToQueue {
		sp.metrics.ErrorBusy.Inc(int64(len(mSpans)))
		sp.metrics.SpansDropped.Inc(int64(len(mSpans)))
		sp.batchMetrics.BatchesFailedToEnqueue.Inc(1)
		sp.batchMetrics.BatchesDropped.Inc(1)
	} else {
		sp.batchMetrics.BatchesEnqueued.Inc(1)
	}
	return addedToQueue
}

func (sp *queuedSpanProcessor) processItemFromQueue(item *queueItem) {
	startTime := time.Now()
	oks, err := sp.sender.ProcessSpans(item.mSpans, item.spanFormat)
	fail := err != nil
	if !fail {
		for _, ok := range oks {
			fail = fail || !ok
		}
	}
	if fail {
		sp.metrics.SpansFailedToWrite.Inc(int64(len(item.mSpans)))
		sp.batchMetrics.BatchesProcessingFailedAttempts.Inc(1)
		if !sp.retryOnProcessingFailure {
			// throw away the batch
			sp.logger.Error("Failed to process batch, discarding", zap.Int("batch-size", len(oks)))
			sp.metrics.SpansDropped.Inc(int64(len(item.mSpans)))
		} else {
			// try to put it back at the end of queue for retry at a later time
			addedToQueue := sp.queue.Produce(item)
			if !addedToQueue {
				sp.logger.Error("Failed to process batch and failed to re-enqueue", zap.Int("batch-size", len(oks)))
				sp.metrics.ErrorBusy.Inc(int64(len(item.mSpans)))
				sp.metrics.SpansDropped.Inc(int64(len(item.mSpans)))
			} else {
				sp.logger.Error("Failed to process batch, re-enqueued", zap.Int("batch-size", len(oks)))
			}
		}
		// back-off for configured delay, but get interrupted when shutting down
		if sp.backoffDelay > 0 {
			sp.logger.Warn("Backing off before next attempt",
				zap.Duration("backoff-delay", sp.backoffDelay))
			select {
			case <-sp.stopCh:
				sp.logger.Info("Interrupted due to shutdown")
				break
			case <-time.After(sp.backoffDelay):
				sp.logger.Info("Resume processing")
				break
			}
		}
	} else {
		sp.batchMetrics.BatchesProcessingSuccessfulAttempts.Inc(1)
		for _, mSpan := range item.mSpans {
			sp.metrics.SavedBySvc.ReportServiceNameForSpan(mSpan)
			sp.metrics.SaveLatency.Record(time.Since(startTime))
			sp.metrics.InQueueLatency.Record(time.Since(item.queuedTime))
		}
	}
}
