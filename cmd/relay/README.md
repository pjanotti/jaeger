# Jaeger Relay Component

The `Jaeger Relay` is a component that receives spans from an upstream
tracing component (e.g. jaeger agent) and relays those spans to downstream tracing
component(s) (e.g. jaeger collector or the jaeger relay). The relay supports
receiving and exporting spans in a number of protocols, multi-destination
support, and per-destination in-memory buffering with retry. This component enables
ingestion pipeline configurations for several use-cases.

The relay provides several features:
- Support for incoming data in a number of protocols, currently:
  - Zipkin (Thrift encoding over HTTP)
  - Jaeger (Thrift encoding over HTTP)
  - Jaeger (Thrift encoding over tChannel)
- Support for exporting data over a number of protocols, currently:
  - Jaeger (Thrift encoding over HTTP)
  - Jaeger (Thrift encoding over tChannel)
- Support for exporting to multiple destinations, with separate in-memory bounded
  buffer per destination. Option to retry in case of failure to export (e.g. remote
  endpoint unavailable), with configurable back-off delay
- Detailed telemetry including metrics for each stage in the pipeline (E.g. exposed via
  prometheus endpoint)
- Many configuration options to build a custom data pipeline with desired behaviors

## Example Configurations
Below we show a few configurations possible. The module itself
is quite flexible and configuration options allow for many more
flexible data pipelines to be configured.

### Configuration 1 : Agents -> Relay -> Collector
![Configuration 1 : Agents -> Relay -> Collector](img/relay-config-1.png)

### Configuration 2 : Zipkin Tasks -> Relay -> Collector
The relay is capable of receiving spans in zipkin format, thus
pretending to be `Zipkin collector` and internally adapting to
Jaeger format before sending out to the Jaeger collector.
![Configuration 2 : Zipkin Tasks -> Relay -> Collector](img/relay-config-2.png)

### Configuration 3 : Agents -> Relay -> Relay -> Collector
![Configuration 3 : Agents -> Relay -> Relay -> Collector](img/relay-config-3.png)

### Configuration 4 : Agents -> Relay -> Multiple Endpoints
![Configuration 4 : Agents -> Relay -> Multiple Endpoints](img/relay-config-4.png)

## Roadmap

There are many features that can be added into the relay:
- Support receive and export in the proto model over grpc
- Support for per export target filtering
- Support other exporters (e.g. S3, Kinesis, etc.)
- Support for mutual TLS with remote destination

