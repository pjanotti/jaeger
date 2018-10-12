// Copyright (c) 2017 Uber Technologies, Inc.
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

package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/gorilla/mux"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	jMetrics "github.com/uber/jaeger-lib/metrics"
	tchannel "github.com/uber/tchannel-go"
	"github.com/uber/tchannel-go/thrift"
	"go.uber.org/zap"

	tchReporter "github.com/jaegertracing/jaeger/cmd/agent/app/reporter/tchannel"
	cApp "github.com/jaegertracing/jaeger/cmd/collector/app"
	zs "github.com/jaegertracing/jaeger/cmd/collector/app/sanitizer/zipkin"
	"github.com/jaegertracing/jaeger/cmd/collector/app/zipkin"
	"github.com/jaegertracing/jaeger/cmd/env"
	"github.com/jaegertracing/jaeger/cmd/flags"
	"github.com/jaegertracing/jaeger/cmd/relay/app/builder"
	"github.com/jaegertracing/jaeger/cmd/relay/app/processor"
	"github.com/jaegertracing/jaeger/cmd/relay/app/sender"
	"github.com/jaegertracing/jaeger/pkg/config"
	"github.com/jaegertracing/jaeger/pkg/healthcheck"
	"github.com/jaegertracing/jaeger/pkg/metrics"
	pMetrics "github.com/jaegertracing/jaeger/pkg/metrics"
	"github.com/jaegertracing/jaeger/pkg/recoveryhandler"
	"github.com/jaegertracing/jaeger/pkg/version"
	jc "github.com/jaegertracing/jaeger/thrift-gen/jaeger"
	zc "github.com/jaegertracing/jaeger/thrift-gen/zipkincore"
)

const (
	serviceName                = "jaeger-collector"
	defaultHealthCheckHTTPPort = 14269
)

func main() {
	var signalsChannel = make(chan os.Signal)
	signal.Notify(signalsChannel, os.Interrupt, syscall.SIGTERM)

	v := viper.New()
	var command = &cobra.Command{
		Use:   "jaeger-relay",
		Short: "Jaeger relay is a utility program which relays span data.",
		Long: `Jaeger relay is a utility program that can receive data over
		jaeger-thrift (tchannel/http) or zipkin-thrift (http) and relay to
		jaeger-collector using a choice of thrift-tchannel or thrift-http sender.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			err := flags.TryLoadConfigFile(v)
			if err != nil {
				return err
			}

			sFlags := new(flags.SharedFlags).InitFromViper(v)
			logger, err := sFlags.NewLogger(zap.NewProductionConfig())
			if err != nil {
				return err
			}
			hc, err := sFlags.NewHealthCheck(logger)
			if err != nil {
				logger.Fatal("Could not start the health check server.", zap.Error(err))
				return err
			}

			mBldr := new(pMetrics.Builder).InitFromViper(v)
			baseFactory, err := mBldr.CreateMetricsFactory("jaeger_relay")
			if err != nil {
				logger.Fatal("Cannot create metrics factory.", zap.Error(err))
				return err
			}

			// construct multi-processor from config options
			multiProcessorOpts := builder.NewMultiSpanProcessorOptions().InitFromViper(v)
			processors := make([]cApp.SpanProcessor, 0)
			for _, popt := range multiProcessorOpts.Processors {
				proc, err := buildQueuedSpanProcessor(
					logger,
					baseFactory,
					popt,
				)
				if err != nil {
					logger.Fatal("Cannot build processor.", zap.Error(err))
					return err
				} else {
					processors = append(processors, proc)
				}
			}
			multiSpanProcessor := processor.NewMultiSpanProcessor(processors...)

			receiverOpts := multiProcessorOpts.Receiver

			// construct receiver and configure to send to processor
			jaegerBatchesHandler := cApp.NewJaegerSpanHandler(logger, multiSpanProcessor)
			zSanitizer := zs.NewChainedSanitizer(
				zs.NewSpanDurationSanitizer(),
				zs.NewSpanStartTimeSanitizer(),
				zs.NewParentIDSanitizer(),
				zs.NewErrorTagSanitizer(),
			)
			zipkinSpansHandler := cApp.NewZipkinSpanHandler(logger, multiSpanProcessor, zSanitizer)

			// register (jaeger, zipkin) thrift with tchannel-based server
			ch, err := tchannel.NewChannel(serviceName, &tchannel.ChannelOptions{})
			if err != nil {
				logger.Fatal("Unable to create new TChannel", zap.Error(err))
			}
			server := thrift.NewServer(ch)
			server.Register(jc.NewTChanCollectorServer(jaegerBatchesHandler))
			server.Register(zc.NewTChanZipkinCollectorServer(zipkinSpansHandler))

			portStr := ":" + strconv.Itoa(receiverOpts.JaegerThriftTChannelPort)
			listener, err := net.Listen("tcp", portStr)
			if err != nil {
				logger.Fatal("Unable to start listening on tchannel", zap.Error(err))
			} else {
				logger.Info("Starting to listen on tchannel",
					zap.String("port", portStr),
					zap.String("service-name", serviceName))
			}
			ch.Serve(listener)

			// register (jaeger, zipkin) thrift with http-based server
			r := mux.NewRouter()
			apiHandler := cApp.NewAPIHandler(jaegerBatchesHandler)
			apiHandler.RegisterRoutes(r)
			if h := mBldr.Handler(); h != nil {
				logger.Info("Registering metrics handler with HTTP server", zap.String("route", mBldr.HTTPRoute))
				r.Handle(mBldr.HTTPRoute, h)
			}
			httpPortStr := ":" + strconv.Itoa(receiverOpts.JaegerThriftHTTPPort)
			recoveryHandler := recoveryhandler.NewRecoveryHandler(logger, true)

			// Go!
			go startZipkinHTTPAPI(logger, receiverOpts.ZipkinThriftHTTPPort, zipkinSpansHandler, recoveryHandler)
			logger.Info("Starting HTTP server", zap.Int("http-port", receiverOpts.JaegerThriftHTTPPort))
			go func() {
				if err := http.ListenAndServe(httpPortStr, recoveryHandler(r)); err != nil {
					logger.Fatal("Could not launch service", zap.Error(err))
				}
				hc.Set(healthcheck.Unavailable)
			}()

			hc.Ready()
			<-signalsChannel
			logger.Info("Shutting down")

			logger.Info("Shutdown complete")
			return nil
		},
	}

	command.AddCommand(version.Command())
	command.AddCommand(env.Command())

	flags.SetDefaultHealthCheckPort(defaultHealthCheckHTTPPort)

	config.AddFlags(
		v,
		command,
		flags.AddConfigFileFlag,
		flags.AddFlags,
		metrics.AddFlags,
	)

	if err := command.Execute(); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

func buildQueuedSpanProcessor(
	logger *zap.Logger,
	baseFactory jMetrics.Factory,
	opts *builder.QueuedSpanProcessorOptions,
) (cApp.SpanProcessor, error) {
	logger.Info("Constructing queue processor with name", zap.String("name", opts.Name))
	hostname, _ := os.Hostname()
	tags := map[string]string{
		"jgr_processor": opts.Name,
		"jgr_host":      hostname,
	}
	metricsFactory := baseFactory.Namespace("", tags)

	// build span batch sender from configured options
	var spanSender cApp.SpanProcessor
	switch opts.SenderType {
	case builder.ThriftTChannelSenderType:
		logger.Info("Initializing thrift-tChannel sender")
		thriftTChannelSenderOpts := opts.SenderConfig.(*builder.JaegerThriftTChannelSenderOptions)
		tchrepbuilder := &tchReporter.Builder{
			CollectorHostPorts: thriftTChannelSenderOpts.CollectorHostPorts,
			DiscoveryMinPeers:  thriftTChannelSenderOpts.DiscoveryMinPeers,
			ConnCheckTimeout:   thriftTChannelSenderOpts.DiscoveryConnCheckTimeout,
		}
		tchreporter, err := tchrepbuilder.CreateReporter(metricsFactory, logger)
		if err != nil {
			logger.Fatal("Cannot create tchannel reporter.", zap.Error(err))
			return nil, err
		}
		spanSender = sender.NewJaegerThriftTChannelSender(
			tchreporter, metricsFactory, logger)
	case builder.ThriftHTTPSenderType:
		thriftHTTPSenderOpts := opts.SenderConfig.(*builder.JaegerThriftHTTPSenderOptions)
		logger.Info("Initializing thrift-HTTP sender",
			zap.String("url", thriftHTTPSenderOpts.CollectorEndpoint))
		spanSender = sender.NewJaegerThriftHTTPSender(
			thriftHTTPSenderOpts.CollectorEndpoint,
			thriftHTTPSenderOpts.Headers,
			metricsFactory,
			logger,
			sender.HTTPTimeout(thriftHTTPSenderOpts.Timeout),
		)
	default:
		logger.Fatal("Unrecognized sender type configured")
	}

	// build queued span processor with underlying sender
	queuedSpanProcessor := processor.NewQueuedSpanProcessor(
		spanSender,
		processor.Options.Logger(logger),
		processor.Options.ServiceMetrics(metricsFactory),
		processor.Options.HostMetrics(metricsFactory),
		processor.Options.Name(opts.Name),
		processor.Options.NumWorkers(opts.NumWorkers),
		processor.Options.QueueSize(opts.QueueSize),
		processor.Options.RetryOnProcessingFailures(opts.RetryOnFailure),
		processor.Options.BackoffDelay(opts.BackoffDelay),
	)
	return queuedSpanProcessor, nil
}

func startZipkinHTTPAPI(
	logger *zap.Logger,
	zipkinPort int,
	zipkinSpansHandler cApp.ZipkinSpansHandler,
	recoveryHandler func(http.Handler) http.Handler,
) {
	if zipkinPort != 0 {
		zHandler := zipkin.NewAPIHandler(zipkinSpansHandler)
		r := mux.NewRouter()
		zHandler.RegisterRoutes(r)

		httpPortStr := ":" + strconv.Itoa(zipkinPort)
		logger.Info("Listening for Zipkin HTTP traffic", zap.Int("zipkin.http-port", zipkinPort))

		if err := http.ListenAndServe(httpPortStr, recoveryHandler(r)); err != nil {
			logger.Fatal("Could not launch service", zap.Error(err))
		}
	}
}
