package user

import "go.opentelemetry.io/otel"

var tracer = otel.Tracer("user")
