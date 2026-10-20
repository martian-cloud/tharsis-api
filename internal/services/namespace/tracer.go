package namespace

import "go.opentelemetry.io/otel"

var tracer = otel.Tracer("namespace")
