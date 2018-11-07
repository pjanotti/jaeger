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

package builder

import (
	"time"

	"github.com/spf13/viper"
)

// SenderType indicates the type of sender
type SenderType string

const (
	// NullSenderType for performance measurements
	NullSenderType SenderType = "null-sender"
	// ThriftTChannelSenderType represents a thrift-format tchannel-transport sender
	ThriftTChannelSenderType = "thrift-tchannel"
	// ThriftHTTPSenderType represents a thrift-format http-transport sender
	ThriftHTTPSenderType = "thrift-http"
	// InvalidSenderType represents an invalid sender
	InvalidSenderType = "invalid"
)

// ReceiverOptions holds configuration for receivers
type ReceiverOptions struct {
	// JaegerThriftTChannelPort is the port that the relay receives on for jaeger thrift tchannel requests
	JaegerThriftTChannelPort int `mapstructure:"jaeger-thrift-tchannel-port"`
	// ReceiverJaegerHTTPPort is the port that the relay receives on for jaeger thrift http requests
	JaegerThriftHTTPPort int `mapstructure:"jaeger-thrift-http-port"`
	// ZipkinThriftHTTPPort is the port that the relay receives on for zipkin thrift http requests
	ZipkinThriftHTTPPort int `mapstructure:"zipkin-thrift-tchannel-port"`
}

// NewReceiverOptions returns an instance of ReceiverOptions with default values
func NewReceiverOptions() *ReceiverOptions {
	opts := &ReceiverOptions{
		JaegerThriftTChannelPort: 14267,
		JaegerThriftHTTPPort:     14268,
		ZipkinThriftHTTPPort:     9411,
	}
	return opts
}

// JaegerThriftTChannelSenderOptions holds configuration for Jaeger Thrift Tchannel sender
type JaegerThriftTChannelSenderOptions struct {
	CollectorHostPorts        []string      `mapstructure:"collector-host-ports"`
	DiscoveryMinPeers         int           `mapstructure:"discovery-min-peers"`
	DiscoveryConnCheckTimeout time.Duration `mapstructure:"discovery-conn-check-timeout"`
}

// NewJaegerThriftTChannelSenderOptions returns an instance of JaegerThriftTChannelSenderOptions with default values
func NewJaegerThriftTChannelSenderOptions() *JaegerThriftTChannelSenderOptions {
	opts := &JaegerThriftTChannelSenderOptions{
		DiscoveryMinPeers:         3,
		DiscoveryConnCheckTimeout: 250 * time.Millisecond,
	}
	return opts
}

// JaegerThriftHTTPSenderOptions holds configuration for Jaeger Thrift HTTP sender
type JaegerThriftHTTPSenderOptions struct {
	CollectorEndpoint string            `mapstructure:"collector-endpoint"`
	Timeout           time.Duration     `mapstructure:"timeout"`
	Headers           map[string]string `mapstructure:"headers"`
}

// NewJaegerThriftHTTPSenderOptions returns an instance of JaegerThriftHTTPSenderOptions with default values
func NewJaegerThriftHTTPSenderOptions() *JaegerThriftHTTPSenderOptions {
	opts := &JaegerThriftHTTPSenderOptions{
		Timeout: 5 * time.Second,
	}
	return opts
}

// QueuedSpanProcessorOptions holds configuration for the queued span processor
type QueuedSpanProcessorOptions struct {
	// Name is the friendly name of the processor
	Name string
	// NumWorkers is the number of queue workers that dequeue batches and send them out
	NumWorkers int `mapstructure:"num-workers"`
	// QueueSize is the maximum number of batches allowed in queue at a given time
	QueueSize int `mapstructure:"queue-size"`
	// Retry indicates whether queue processor should retry span batches in case of processing failure
	RetryOnFailure bool `mapstructure:"retry-on-failure"`
	// BackoffDelay is the amount of time a worker waits after a failed send before retrying
	BackoffDelay time.Duration `mapstructure:"backoff-delay"`
	// SenderType indicates the type of sender to instantiate
	SenderType   SenderType `mapstructure:"sender-type"`
	SenderConfig interface{}
}

// NewQueuedSpanProcessorOptions returns an instance of QueuedSpanProcessorOptions with default values
func NewQueuedSpanProcessorOptions() *QueuedSpanProcessorOptions {
	opts := &QueuedSpanProcessorOptions{
		NumWorkers:     10,
		QueueSize:      5000,
		RetryOnFailure: true,
		SenderType:     InvalidSenderType,
		BackoffDelay:   5 * time.Second,
	}
	return opts
}

// InitFromViper initializes QueuedSpanProcessorOptions with properties from viper
func (qOpts *QueuedSpanProcessorOptions) InitFromViper(v *viper.Viper) *QueuedSpanProcessorOptions {
	v.Unmarshal(qOpts)
	switch qOpts.SenderType {
	case ThriftTChannelSenderType:
		ttsopts := NewJaegerThriftTChannelSenderOptions()
		v.Sub(string(ThriftTChannelSenderType)).Unmarshal(ttsopts)
		qOpts.SenderConfig = ttsopts
	case ThriftHTTPSenderType:
		thsOpts := NewJaegerThriftHTTPSenderOptions()
		v.Sub(string(ThriftHTTPSenderType)).Unmarshal(thsOpts)
		qOpts.SenderConfig = thsOpts
	}
	return qOpts
}

// MultiSpanProcessorOptions holds configuration for all the span processors
type MultiSpanProcessorOptions struct {
	Receiver   *ReceiverOptions
	Processors []*QueuedSpanProcessorOptions
}

// NewMultiSpanProcessorOptions returns an instance of MultiSpanProcessorOptions with default values
func NewMultiSpanProcessorOptions() *MultiSpanProcessorOptions {
	opts := &MultiSpanProcessorOptions{
		Receiver:   NewReceiverOptions(),
		Processors: make([]*QueuedSpanProcessorOptions, 0),
	}
	return opts
}

// InitFromViper initializes MultiProcessorOptions with properties from viper
func (mOpts *MultiSpanProcessorOptions) InitFromViper(v *viper.Viper) *MultiSpanProcessorOptions {
	recvv := v.Sub("receiver")
	recvv.Unmarshal(mOpts.Receiver)
	procsv := v.Sub("processors")
	for procName := range v.GetStringMap("processors") {
		procv := procsv.Sub(procName)
		procOpts := NewQueuedSpanProcessorOptions()
		procOpts.Name = procName
		procOpts.InitFromViper(procv)
		mOpts.Processors = append(mOpts.Processors, procOpts)
	}
	return mOpts
}
