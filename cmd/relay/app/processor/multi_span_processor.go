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
	cApp "github.com/jaegertracing/jaeger/cmd/collector/app"
	"github.com/jaegertracing/jaeger/model"
	"github.com/jaegertracing/jaeger/pkg/multierror"
)

// MultiSpanProcessor enables processing on multiple processors.
// For each incoming span batch, it calls ProcessSpans method on each span
// processor one-by-one. It aggregates success/failures/errors from all of
// them and reports the result upstream.
type MultiSpanProcessor []cApp.SpanProcessor

// NewMultiSpanProcessor creates a MultiSpanProcessor from the variadic
// list of passed SpanProcessors.
func NewMultiSpanProcessor(procs ...cApp.SpanProcessor) MultiSpanProcessor {
	return procs
}

// ProcessSpans implements the SpanProcessor interface
func (msp MultiSpanProcessor) ProcessSpans(mSpans []*model.Span, spanFormat string) ([]bool, error) {
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
