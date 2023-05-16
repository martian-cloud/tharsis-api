package job

import "go.opentelemetry.io/otel"

var tracer = otel.Tracer("job")
