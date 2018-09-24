package sender

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/apache/thrift/lib/go/thrift"
	"github.com/jaegertracing/jaeger/model"
	jConv "github.com/jaegertracing/jaeger/model/converter/thrift/jaeger"
	tmodel "github.com/jaegertracing/jaeger/thrift-gen/jaeger"
	"github.com/uber/jaeger-lib/metrics"
	"go.uber.org/zap"
)

// Default timeout for http request in seconds
const defaultHTTPTimeout = time.Second * 5

// SpanThriftHTTPSender forwards spans to a http server.
type SpanThriftHTTPSender struct {
	url           string
	headers       map[string]string
	client        *http.Client
	logger        *zap.Logger
	senderMetrics senderMetrics
}

// HTTPOption sets a parameter for the HttpCollector
type HTTPOption func(s *SpanThriftHTTPSender)

// HTTPTimeout sets maximum timeout for http request.
func HTTPTimeout(duration time.Duration) HTTPOption {
	return func(s *SpanThriftHTTPSender) { s.client.Timeout = duration }
}

// HTTPRoundTripper configures the underlying Transport on the *http.Client
// that is used
func HTTPRoundTripper(transport http.RoundTripper) HTTPOption {
	return func(s *SpanThriftHTTPSender) {
		s.client.Transport = transport
	}
}

// NewSpanThriftHTTPSender returns a new HTTP-backend span sender. url should be an http
// url of the collector to handle POST request, typically something like:
//     http://hostname:14268/api/traces?format=jaeger.thrift
func NewSpanThriftHTTPSender(
	url string,
	headers map[string]string,
	mFactory metrics.Factory,
	zlogger *zap.Logger,
	options ...HTTPOption,
) *SpanThriftHTTPSender {
	sm := senderMetrics{}
	metrics.Init(&sm, mFactory.Namespace("span-thrift-http-sender", nil), nil)
	s := &SpanThriftHTTPSender{
		url:           url,
		headers:       headers,
		client:        &http.Client{Timeout: defaultHTTPTimeout},
		logger:        zlogger,
		senderMetrics: sm,
	}

	for _, option := range options {
		option(s)
	}
	return s
}

// ProcessSpans implements SpanProcessor interface
func (s *SpanThriftHTTPSender) ProcessSpans(mSpans []*model.Span, spanFormat string) ([]bool, error) {
	s.senderMetrics.BatchesIncoming.Inc(1)
	s.senderMetrics.SpansIncoming.Inc(int64(len(mSpans)))
	tBatch := &tmodel.Batch{
		Process: jConv.FromDomainProcess(mSpans[0].Process),
		Spans:   jConv.FromDomain(mSpans),
	}
	body, err := serializeThrift(tBatch)
	if err != nil {
		return s.fail(len(mSpans), err)
	}
	req, err := http.NewRequest("POST", s.url, body)
	if err != nil {
		return s.fail(len(mSpans), err)
	}
	req.Header.Set("Content-Type", "application/x-thrift")
	for k, v := range s.headers {
		req.Header.Set(k, v)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return s.fail(len(mSpans), err)
	}
	io.Copy(ioutil.Discard, resp.Body)
	resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		return s.fail(len(mSpans), fmt.Errorf("error from collector: %d", resp.StatusCode))
	}
	oks := make([]bool, len(mSpans))
	for i := range oks {
		oks[i] = true
	}
	s.senderMetrics.BatchesSent.Inc(1)
	s.senderMetrics.SpansSent.Inc(int64(len(oks)))
	return oks, nil
}

func serializeThrift(obj thrift.TStruct) (*bytes.Buffer, error) {
	t := thrift.NewTMemoryBuffer()
	p := thrift.NewTBinaryProtocolTransport(t)
	if err := obj.Write(p); err != nil {
		return nil, err
	}
	return t.Buffer, nil
}

func (s *SpanThriftHTTPSender) fail(msgsCount int, err error) ([]bool, error) {
	s.logger.Error("Sender failed with error", zap.Error(err))
	oks := make([]bool, msgsCount)
	for i := range oks {
		oks[i] = false
	}
	s.senderMetrics.BatchesFailedToSend.Inc(1)
	s.senderMetrics.SpansFailedToSend.Inc(int64(msgsCount))
	return oks, err
}
