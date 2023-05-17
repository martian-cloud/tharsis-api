package workspace

import "go.opentelemetry.io/otel"

var tracer = otel.Tracer("workspace")
