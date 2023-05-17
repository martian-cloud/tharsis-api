package db

import "go.opentelemetry.io/otel"

var tracer = otel.Tracer("db")
