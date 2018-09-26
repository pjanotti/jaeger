package app

import (
	cApp "github.com/jaegertracing/jaeger/cmd/collector/app"
	"github.com/jaegertracing/jaeger/model"
	"github.com/jaegertracing/jaeger/pkg/multierror"
)

// MultiSpanBatchProcessor provides serial span processing on one or more processors.
// If more than one expensive processor are needed, one or more of them should be
// wrapped and hidden behind a channel.
type MultiSpanBatchProcessor []cApp.SpanProcessor

// NewMultiSpanBatchProcessor creates a MultiSpanBatchProcessor from the variadic
// list of passed Processors.
func NewMultiSpanBatchProcessor(procs ...cApp.SpanProcessor) MultiSpanBatchProcessor {
	return procs
}

// ProcessSpans implements the SpanProcessor interface
func (msp MultiSpanBatchProcessor) ProcessSpans(mSpans []*model.Span, spanFormat string) ([]bool, error) {
	var errors []error
	allOks := make([]bool, len(mSpans))
	for i := range allOks {
		allOks[i] = true
	}
	for _, sp := range msp {
		if oks, err := sp.ProcessSpans(mSpans, spanFormat); err != nil {
			errors = append(errors, err)
		} else {
			for i := range allOks {
				allOks[i] = allOks[i] && oks[i]
			}
		}
	}
	return allOks, multierror.Wrap(errors)
}
