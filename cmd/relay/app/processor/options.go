package processor

import (
	"time"

	"github.com/uber/jaeger-lib/metrics"
	"go.uber.org/zap"
)

const (
	// DefaultNumWorkers is the default number of workers consuming from the processor queue
	DefaultNumWorkers = 10
	// DefaultQueueSize is the default maximum number of span batches allowed in the processor's queue
	DefaultQueueSize = 1000
)

type options struct {
	logger                   *zap.Logger
	serviceMetrics           metrics.Factory
	hostMetrics              metrics.Factory
	name                     string
	numWorkers               int
	queueSize                int
	backoffDelay             time.Duration
	extraFormatTypes         []string
	retryOnProcessingFailure bool
}

// Option is a function that sets some option on the component.
type Option func(c *options)

// Options is a factory for all available Option's
var Options options

// Logger creates a Option that initializes the logger
func (options) Logger(logger *zap.Logger) Option {
	return func(b *options) {
		b.logger = logger
	}
}

// ServiceMetrics creates an Option that initializes the serviceMetrics metrics factory
func (options) ServiceMetrics(serviceMetrics metrics.Factory) Option {
	return func(b *options) {
		b.serviceMetrics = serviceMetrics
	}
}

// HostMetrics creates an Option that initializes the hostMetrics metrics factory
func (options) HostMetrics(hostMetrics metrics.Factory) Option {
	return func(b *options) {
		b.hostMetrics = hostMetrics
	}
}

// Name creates an Option that initializes the name of the processor
func (options) Name(name string) Option {
	return func(b *options) {
		b.name = name
	}
}

// NumWorkers creates an Option that initializes the number of queue consumers AKA workers
func (options) NumWorkers(numWorkers int) Option {
	return func(b *options) {
		b.numWorkers = numWorkers
	}
}

// QueueSize creates an Option that initializes the queue size
func (options) QueueSize(queueSize int) Option {
	return func(b *options) {
		b.queueSize = queueSize
	}
}

// BackoffDelay creates an Option that initializes the backoff delay
func (options) BackoffDelay(backoffDelay time.Duration) Option {
	return func(b *options) {
		b.backoffDelay = backoffDelay
	}
}

// ExtraFormatTypes creates an Option that initializes the extra list of format types
func (options) ExtraFormatTypes(extraFormatTypes []string) Option {
	return func(b *options) {
		b.extraFormatTypes = extraFormatTypes
	}
}

// RetryOnProcessingFailures creates an Option that initializes the retryOnProcessingFailure boolean
func (options) RetryOnProcessingFailures(retryOnProcessingFailure bool) Option {
	return func(b *options) {
		b.retryOnProcessingFailure = retryOnProcessingFailure
	}
}

func (o options) apply(opts ...Option) options {
	ret := options{}
	for _, opt := range opts {
		opt(&ret)
	}
	if ret.logger == nil {
		ret.logger = zap.NewNop()
	}
	if ret.serviceMetrics == nil {
		ret.serviceMetrics = metrics.NullFactory
	}
	if ret.hostMetrics == nil {
		ret.hostMetrics = metrics.NullFactory
	}
	if ret.numWorkers == 0 {
		ret.numWorkers = DefaultNumWorkers
	}
	if ret.queueSize == 0 {
		ret.queueSize = DefaultQueueSize
	}
	return ret
}
