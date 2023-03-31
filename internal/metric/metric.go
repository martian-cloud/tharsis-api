// Package metric package
package metric

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// NewHistogram returns a new Prometheus Histogram for execution time metrics.
func NewHistogram(name string, help string, start float64, factor float64, count int) prometheus.Histogram {
	return promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    name,
		Help:    help,
		Buckets: prometheus.ExponentialBuckets(start, factor, count),
	})
}

// NewCounter returns a new Prometheus counter.
func NewCounter(name string, help string) prometheus.Counter {
	return promauto.NewCounter(prometheus.CounterOpts{
		Name: name,
		Help: help,
	})
}
