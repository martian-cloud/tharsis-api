package graphql

import "go.opentelemetry.io/otel"

var tracer = otel.Tracer("graphql")
