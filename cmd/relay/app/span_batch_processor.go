package app

import (
	"time"

	cApp "github.com/jaegertracing/jaeger/cmd/collector/app"
	"github.com/jaegertracing/jaeger/model"
	"github.com/jaegertracing/jaeger/pkg/queue"
	"go.uber.org/zap"
)

type spanBatchProcessor struct {
	queue                    *queue.BoundedQueue
	metrics                  *cApp.SpanProcessorMetrics
	logger                   *zap.Logger
	sender                   cApp.SpanProcessor
	numWorkers               int
	retryOnProcessingFailure bool
}

type queueItem struct {
	queuedTime time.Time
	mSpans     []*model.Span
	spanFormat string
}

// NewSpanBatchProcessor returns a NewSpanBatchProcessor that queues and sends out spans
func NewSpanBatchProcessor(
	sender cApp.SpanProcessor,
	opts ...Option,
) cApp.SpanProcessor {
	sp := newSpanBatchProcessor(sender, opts...)

	sp.queue.StartConsumers(sp.numWorkers, func(item interface{}) {
		value := item.(*queueItem)
		sp.processItemFromQueue(value)
	})

	sp.queue.StartLengthReporting(1*time.Second, sp.metrics.QueueLength)

	return sp
}

func newSpanBatchProcessor(
	sender cApp.SpanProcessor,
	opts ...Option,
) *spanBatchProcessor {
	options := Options.apply(opts...)
	handlerMetrics := cApp.NewSpanProcessorMetrics(
		options.serviceMetrics,
		options.hostMetrics,
		options.extraFormatTypes)
	droppedItemHandler := func(item interface{}) {
		batchItem := item.(queueItem)
		handlerMetrics.SpansDropped.Inc(int64(len(batchItem.mSpans)))
	}
	boundedQueue := queue.NewBoundedQueue(options.queueSize, droppedItemHandler)
	sp := spanBatchProcessor{
		queue:                    boundedQueue,
		metrics:                  handlerMetrics,
		logger:                   options.logger,
		numWorkers:               options.numWorkers,
		sender:                   sender,
		retryOnProcessingFailure: options.retryOnProcessingFailure,
	}
	return &sp
}

// Stop halts the span batch processor and all its go-routines.
func (sp *spanBatchProcessor) Stop() {
	sp.queue.Stop()
}

func (sp *spanBatchProcessor) ProcessSpans(mSpans []*model.Span, spanFormat string) ([]bool, error) {
	sp.metrics.BatchSize.Update(int64(len(mSpans)))
	retMe := make([]bool, len(mSpans))
	ok := sp.enqueueSpanBatch(mSpans, spanFormat)
	for i := range mSpans {
		retMe[i] = ok
	}
	return retMe, nil
}

func (sp *spanBatchProcessor) enqueueSpanBatch(mSpans []*model.Span, spanFormat string) bool {
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
	}
	return addedToQueue
}

func (sp *spanBatchProcessor) processItemFromQueue(item *queueItem) {
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
	} else {
		for _, mSpan := range item.mSpans {
			sp.metrics.SavedBySvc.ReportServiceNameForSpan(mSpan)
			sp.metrics.SaveLatency.Record(time.Since(startTime))
			sp.metrics.InQueueLatency.Record(time.Since(item.queuedTime))
		}
	}
}
