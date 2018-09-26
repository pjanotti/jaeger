package sender

import (
	"github.com/uber/jaeger-lib/metrics"
	"go.uber.org/zap"

	reporter "github.com/jaegertracing/jaeger/cmd/agent/app/reporter"
	"github.com/jaegertracing/jaeger/model"
	jConv "github.com/jaegertracing/jaeger/model/converter/thrift/jaeger"
	tmodel "github.com/jaegertracing/jaeger/thrift-gen/jaeger"
)

// JaegerThriftTChannelSender takes span batches and sends them
// out on tchannel in thrift encoding
type JaegerThriftTChannelSender struct {
	logger        *zap.Logger
	reporter      reporter.Reporter
	senderMetrics senderMetrics
}

// NewJaegerThriftTChannelSender creates new TChannel-based sender.
func NewJaegerThriftTChannelSender(
	reporter reporter.Reporter,
	mFactory metrics.Factory,
	zlogger *zap.Logger,
) *JaegerThriftTChannelSender {
	sm := senderMetrics{}
	metrics.Init(&sm, mFactory.Namespace("thrift-tchannel", nil), nil)
	return &JaegerThriftTChannelSender{
		logger:        zlogger,
		reporter:      reporter,
		senderMetrics: sm,
	}
}

// ProcessSpans implements SpanProcessor interface
func (s *JaegerThriftTChannelSender) ProcessSpans(mSpans []*model.Span, spanFormat string) ([]bool, error) {
	s.senderMetrics.BatchesIncoming.Inc(1)
	s.senderMetrics.SpansIncoming.Inc(int64(len(mSpans)))
	tBatch := &tmodel.Batch{
		Process: jConv.FromDomainProcess(mSpans[0].Process),
		Spans:   jConv.FromDomain(mSpans),
	}
	oks := make([]bool, len(mSpans))
	if err := s.reporter.EmitBatch(tBatch); err != nil {
		s.logger.Error("Reporter failed to report span batch", zap.Error(err))
		for i := range oks {
			oks[i] = false
		}
		s.senderMetrics.BatchesFailedToSend.Inc(1)
		s.senderMetrics.SpansFailedToSend.Inc(int64(len(oks)))
		return oks, err
	}
	for i := range oks {
		oks[i] = true
	}
	s.senderMetrics.BatchesSent.Inc(1)
	s.senderMetrics.SpansSent.Inc(int64(len(oks)))
	return oks, nil
}
