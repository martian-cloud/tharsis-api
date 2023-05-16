package run

import "go.opentelemetry.io/otel"

var tracer = otel.Tracer("run")
