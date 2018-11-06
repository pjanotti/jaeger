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
