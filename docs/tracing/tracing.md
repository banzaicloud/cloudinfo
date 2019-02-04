#### Tracing

The cloudinfo application contains code instrumentation that makes possible tracing trough opencensus / jaeger

For the time being the tracing solution only instruments the background processes in the application (collecting cloud information from the providers)

#### Enable tracing

The application comes with tracing disabled by default. Tracing can be configured through the following environment variables:

| Envvar name                                | default value                                           |
| -------------------------------------------| --------------------------------------------------------| 
|  INSTRUMENTATION_JAEGER_ENABLED            | false                                                   |
|  INSTRUMENTATION_JAEGER_COLLECTORENDPOINT  | http://localhost:14268/api/traces?format=jaeger.thrift  |
|  INSTRUMENTATION_JAEGER_AGENTENDPOINT      | localhost:6832                                          |
|  INSTRUMENTATION_JAEGER_USERNAME           |                                                         |
|  INSTRUMENTATION_JAEGER_PASSWORD           |                                                         |

#### Requirements

Cludinfo tracing reports to a jaeger installation that needs to be reachable by the application. The related configuration entries need to be set accordingly.

#### Reference

https://opencensus.io/quickstart/go/tracing/
https://opencensus.io/exporters/supported-exporters/go/jaeger/
https://www.jaegertracing.io/docs/1.9/
