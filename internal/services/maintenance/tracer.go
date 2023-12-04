package maintenance

import "go.opentelemetry.io/otel"

var tracer = otel.Tracer("maintenance")
