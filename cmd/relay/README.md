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
- Support for exporting to multiple destinations
- Support for in-memory bounded buffer per destination
- Detailed telemetry including metrics for each stage in the pipeline (E.g. exposed via prometheus endpoint)
- Many configuration options to build a custom ingestion pipeline with desired behaviors

## Example Configurations

### Configuration 1 : Agents -> Relay -> Collector
![Configuration 1 : Agents -> Relay -> Collector](img/relay-config-1.png)

### Configuration 2 : Agents -> Relay -> Relay -> Collector
![Configuration 2 : Agents -> Relay -> Relay -> Collector](img/relay-config-2.png)

### Configuration 3 : Agents -> Relay -> Multiple Endpoints
![Configuration 2 : Agents -> Relay -> Multiple Endpoints](img/relay-config-3.png)

