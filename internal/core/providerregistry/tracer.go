package providerregistry

import "go.opentelemetry.io/otel"

var tracer = otel.Tracer("core/providerregistry")
