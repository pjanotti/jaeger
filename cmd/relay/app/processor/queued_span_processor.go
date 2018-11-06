// Copyright (c) 2018 The Jaeger Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package processor

import (
	"time"

	"github.com/uber/jaeger-lib/metrics"
	"go.uber.org/zap"

	cApp "github.com/jaegertracing/jaeger/cmd/collector/app"
	"github.com/jaegertracing/jaeger/model"
	"github.com/jaegertracing/jaeger/pkg/queue"
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
	tags := map[string]string{"jgr_type": "QueuedSpanProcessor"}
	sm := options.serviceMetrics.Namespace("", tags)
	hm := options.hostMetrics.Namespace("", tags)
	handlerMetrics := cApp.NewSpanProcessorMetrics(
		sm,
		hm,
		options.extraFormatTypes)
	bm := &processorBatchMetrics{}
	metrics.Init(bm, sm, nil)
	// boundedqueue doesn't like a nil dropped handler
	boundedQueue := queue.NewBoundedQueue(options.queueSize, func(item interface{}) {})
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
	ok := sp.enqueueSpanBatch(mSpans, spanFormat)
	retMe := make([]bool, len(mSpans))
	for i := range mSpans {
		retMe[i] = ok
	}
	return retMe, nil
}

func (sp *queuedSpanProcessor) enqueueSpanBatch(mSpans []*model.Span, spanFormat string) bool {
	sp.batchMetrics.BatchesIncoming.Inc(1)
	sp.metrics.BatchSize.Update(int64(len(mSpans)))
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
		sp.onItemDropped(item)
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
			sp.onItemDropped(item)
		} else {
			// try to put it back at the end of queue for retry at a later time
			addedToQueue := sp.queue.Produce(item)
			if !addedToQueue {
				sp.logger.Error("Failed to process batch and failed to re-enqueue", zap.Int("batch-size", len(oks)))
				sp.onItemDropped(item)
			} else {
				sp.logger.Warn("Failed to process batch, re-enqueued", zap.Int("batch-size", len(oks)))
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
			sp.metrics.SavedOkBySvc.ReportServiceNameForSpan(mSpan)
			sp.metrics.SaveLatency.Record(time.Since(startTime))
			sp.metrics.InQueueLatency.Record(time.Since(item.queuedTime))
		}
	}
}

func (sp *queuedSpanProcessor) onItemDropped(item *queueItem) {
	sp.metrics.SpansDropped.Inc(int64(len(item.mSpans)))
	sp.batchMetrics.BatchesDropped.Inc(1)
	spanCounts := sp.metrics.GetCountsForFormat(item.spanFormat)
	for _, mSpan := range item.mSpans {
		spanCounts.DroppedBySvc.ReportServiceNameForSpan(mSpan)
	}
}
