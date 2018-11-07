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

import "github.com/uber/jaeger-lib/metrics"

type processorBatchMetrics struct {
	// Number of batches incoming to processor
	BatchesIncoming metrics.Counter `metric:"batches.incoming"`

	// Number of batches put into queue
	BatchesEnqueued metrics.Counter `metric:"batches.enqueued"`

	// Number of successful attempts to process batches
	BatchesProcessingSuccessfulAttempts metrics.Counter `metric:"batches.processing.attempts.success"`

	// Number of failed attempts to process batches
	BatchesProcessingFailedAttempts metrics.Counter `metric:"batches.processing.attempts.fail"`

	// Number of batches that were dropped by cleanup policy
	BatchesDropped metrics.Counter `metric:"batches.dropped"`
}
