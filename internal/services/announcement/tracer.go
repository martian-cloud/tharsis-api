package announcement

import "go.opentelemetry.io/otel"

var tracer = otel.Tracer("announcement")
