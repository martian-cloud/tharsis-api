package registry

import "go.opentelemetry.io/otel"

var tracer = otel.Tracer("registry")
