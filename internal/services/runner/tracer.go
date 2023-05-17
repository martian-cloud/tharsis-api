package runner

import "go.opentelemetry.io/otel"

var tracer = otel.Tracer("runner")
