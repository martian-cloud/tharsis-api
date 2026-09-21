package email

import "go.opentelemetry.io/otel"

var tracer = otel.Tracer("email")
