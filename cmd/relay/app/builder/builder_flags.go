package builder

import (
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// SenderType indicates the type of sender
type SenderType string

// ConfiguredSenderType indicates the type of sender that was configured by user
var ConfiguredSenderType SenderType

const (
	receiverJaegerTChannelPort = "receiver.jaeger.tchannel-port"
	receiverJaegerHTTPPort     = "receiver.jaeger.http-port"
	receiverZipkinHTTPort      = "receiver.zipkin.http-port"

	queueProcessorNumWorkers     = "queue-processor.num-workers"
	queueProcessorQueueSize      = "queue-processor.queue-size"
	queueProcessorRetryOnFailure = "queue-processor.retry-on-failure"

	senderTypeFlag string = "sender.type"
	// ThriftTChannelSenderType represents a thrift-format tchannel-transport sender
	ThriftTChannelSenderType SenderType = "thrift-tchannel"
	// ThriftHTTPSenderType represents a thrift-format http-transport sender
	ThriftHTTPSenderType = "thrift-http"

	thriftTChannelSenderCollectorHostPorts        = "sender.thrift.tchannel.collector-host-ports"
	thriftTChannelSenderDiscoveryMinPeers         = "sender.thrift.tchannel.discovery.min-peers"
	thriftTChannelSenderDiscoveryConnCheckTimeout = "sender.thrift.tchannel.discovery.conn-check-timeout"

	thriftHTTPSenderCollectorEndpoint = "sender.thrift.http.collector-endpoint"
	thriftHTTPSenderTimeout           = "sender.thrift.http.timeout"
	thriftHTTPSenderHeaders           = "sender.thrift.http.headers"

	// RelayDefaultHealthCheckHTTPPort is the default HTTP Port for health check
	RelayDefaultHealthCheckHTTPPort = 14269

	// defaultQueuedProcessorNumWorkers is the default number of workers consuming from the processor queue
	defaultQueueProcessorNumWorkers = 10
	// defaultQueueProcessorQueueSize is the default maximum number of span batches allowed in the processor's queue
	defaultQueueProcessorQueueSize = 1000

	defaultThriftTChannelSenderDiscoveryMinPeers         = 3
	defaultThriftTChannelSenderDiscoveryConnCheckTimeout = 250 * time.Millisecond

	defaultThriftHTTPSenderTimeout = 5 * time.Second
)

// ReceiverOptions holds configuration for receivers
type ReceiverOptions struct {
	// ReceiverJaegerTchannelPort is the port that the relay receives on for tchannel requests
	ReceiverJaegerTChannelPort int
	// ReceiverJaegerHTTPPort is the port that the relay receives on for http requests
	ReceiverJaegerHTTPPort int
	// ReceiverZipkinHTTPPort is the port that the relay receives on for zipkin http requests
	ReceiverZipkinHTTPPort int
}

// QueueProcessorOptions holds configuration for the queued batch processor
type QueueProcessorOptions struct {
	// NumWorkers is the number of queue workers that dequeue batches and send them out
	NumWorkers int
	// QueueSize is the maximum number of batches allowed in queue at a given time
	QueueSize int
	// Retry indicates whether queue processor should retry span batches in case of processing failure
	RetryOnFailure bool
}

// ThriftTChannelSenderOptions holds configuration for Thrift Tchannel sender
type ThriftTChannelSenderOptions struct {
	CollectorHostPorts        []string
	DiscoveryMinPeers         int
	DiscoveryConnCheckTimeout time.Duration
}

// ThriftHTTPSenderOptions holds configuration for Thrift HTTP sender
type ThriftHTTPSenderOptions struct {
	CollectorEndpoint string
	HTTPTimeout       time.Duration
	Headers           map[string]string
}

func (s *SenderType) String() string {
	return string(*s)
}

// Set sets the value from string
func (s *SenderType) Set(value string) error {
	switch value {
	case string(ThriftTChannelSenderType):
		*s = ThriftTChannelSenderType
	case string(ThriftHTTPSenderType):
		*s = ThriftHTTPSenderType
	default:
		return fmt.Errorf("Unrecognized sender type %s", value)
	}
	return nil
}

// AddFlags adds flags for ReceiverOptions
func AddFlags(flags *flag.FlagSet) {
	addReceiverFlags(flags)
	addQueueProcessorFlags(flags)
	flags.Var(&ConfiguredSenderType, senderTypeFlag, "The type of sender to instantiate")
	addThriftTChannelReporterFlags(flags)
	addThriftHTTPReporterFlags(flags)
}

func addReceiverFlags(flags *flag.FlagSet) {
	flags.Int(receiverJaegerTChannelPort, 14267, "The tchannel port for the Jaeger receiver service")
	flags.Int(receiverJaegerHTTPPort, 14268, "The http port for the Jaeger receiver service")
	flags.Int(receiverZipkinHTTPort, 9411, "The http port for the Zipkin reciver service e.g. 9411")
}

func addQueueProcessorFlags(flags *flag.FlagSet) {
	flags.Int(
		queueProcessorNumWorkers,
		defaultQueueProcessorNumWorkers,
		"The number of workers consuming from the processor queue")
	flags.Int(
		queueProcessorQueueSize,
		defaultQueueProcessorQueueSize,
		"The maximum number of span batches allowed in the processor's queue (batch sizes can vary and depend upon client settings)")
	flags.Bool(
		queueProcessorRetryOnFailure,
		false,
		"Whether queue processor should retry span batch in case of processing failure")
}

func addThriftTChannelReporterFlags(flags *flag.FlagSet) {
	flags.String(
		thriftTChannelSenderCollectorHostPorts,
		"",
		"(with thrift tchannel sender) comma-separated string representing host:ports of a static list of collectors to connect to directly (e.g. when not using service discovery)")
	flags.Int(
		thriftTChannelSenderDiscoveryMinPeers,
		defaultThriftTChannelSenderDiscoveryMinPeers,
		"(with thrift tchannel sender) if using service discovery, the min number of connections to maintain to the backend")
	flags.Duration(
		thriftTChannelSenderDiscoveryConnCheckTimeout,
		defaultThriftTChannelSenderDiscoveryConnCheckTimeout,
		"(with thrift tchannel sender) sets the timeout used when establishing new connections")
}

func addThriftHTTPReporterFlags(flags *flag.FlagSet) {
	flags.String(
		thriftHTTPSenderCollectorEndpoint,
		"",
		"(with thrift HTTP sender) the collector endpoint to send spans to")
	flags.Duration(
		thriftHTTPSenderTimeout,
		defaultThriftHTTPSenderTimeout,
		"(with thrift HTTP sender) sets the timeout used for HTTP client")
	flags.String(
		thriftHTTPSenderHeaders,
		"",
		"(with thrift http sender) comma-separated string representing header-name:header-value key-value pairs to send as headers")
}

// InitFromViper initializes ReceiverOptions with properties from viper
func (rOpts *ReceiverOptions) InitFromViper(v *viper.Viper) *ReceiverOptions {
	rOpts.ReceiverJaegerTChannelPort = v.GetInt(receiverJaegerTChannelPort)
	rOpts.ReceiverJaegerHTTPPort = v.GetInt(receiverJaegerHTTPPort)
	rOpts.ReceiverZipkinHTTPPort = v.GetInt(receiverZipkinHTTPort)
	return rOpts
}

// InitFromViper initializes QueueProcessorOptions with properties from viper
func (qOpts *QueueProcessorOptions) InitFromViper(v *viper.Viper) *QueueProcessorOptions {
	qOpts.NumWorkers = v.GetInt(queueProcessorNumWorkers)
	qOpts.QueueSize = v.GetInt(queueProcessorQueueSize)
	qOpts.RetryOnFailure = v.GetBool(queueProcessorRetryOnFailure)
	return qOpts
}

// InitSenderTypeFromViper initializes senderType with property from viper
func InitSenderTypeFromViper(v *viper.Viper) {
	ConfiguredSenderType.Set(v.GetString(senderTypeFlag))
}

// InitFromViper initializes ThriftTChannelSenderOptions with properties from viper
func (sOpts *ThriftTChannelSenderOptions) InitFromViper(v *viper.Viper) *ThriftTChannelSenderOptions {
	if len(v.GetString(thriftTChannelSenderCollectorHostPorts)) > 0 {
		sOpts.CollectorHostPorts = strings.Split(v.GetString(thriftTChannelSenderCollectorHostPorts), ",")
	}
	sOpts.DiscoveryMinPeers = v.GetInt(thriftTChannelSenderDiscoveryMinPeers)
	sOpts.DiscoveryConnCheckTimeout = v.GetDuration(thriftTChannelSenderDiscoveryConnCheckTimeout)
	return sOpts
}

// InitFromViper initializes ThriftHTTPSenderOptions with properties from viper
func (sOpts *ThriftHTTPSenderOptions) InitFromViper(v *viper.Viper) *ThriftHTTPSenderOptions {
	sOpts.CollectorEndpoint = v.GetString(thriftHTTPSenderCollectorEndpoint)
	sOpts.HTTPTimeout = v.GetDuration(thriftHTTPSenderTimeout)
	sOpts.Headers = make(map[string]string)
	if len(v.GetString(thriftHTTPSenderHeaders)) > 0 {
		headers := strings.Split(v.GetString(thriftHTTPSenderHeaders), ",")
		for _, header := range headers {
			// TODO: for now we silently fail if header cannot be parsed
			// into a key:value pair unambiguously
			kv := strings.Split(header, ":")
			if len(kv) == 2 {
				sOpts.Headers[kv[0]] = kv[1]
			}
		}
	}
	return sOpts
}
