package cli

import "go.opentelemetry.io/otel"

var tracer = otel.Tracer("cli")
