package activity

import "go.opentelemetry.io/otel"

var tracer = otel.Tracer("core/activity")
