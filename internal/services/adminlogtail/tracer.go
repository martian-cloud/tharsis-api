package adminlogtail

import "go.opentelemetry.io/otel"

var tracer = otel.Tracer("adminlogtail")
