package serviceaccount

import "go.opentelemetry.io/otel"

var tracer = otel.Tracer("serviceaccount")
