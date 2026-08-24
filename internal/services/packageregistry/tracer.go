package packageregistry

import "go.opentelemetry.io/otel"

var tracer = otel.Tracer("services/packageregistry")
