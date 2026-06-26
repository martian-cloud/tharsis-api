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

// NewHistogramVec returns a new Prometheus HistogramVec with labels.
func NewHistogramVec(name string, help string, start float64, factor float64, count int, labels []string) *prometheus.HistogramVec {
	return promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    name,
		Help:    help,
		Buckets: prometheus.ExponentialBuckets(start, factor, count),
	}, labels)
}

// NewCounter returns a new Prometheus counter.
func NewCounter(name string, help string) prometheus.Counter {
	return promauto.NewCounter(prometheus.CounterOpts{
		Name: name,
		Help: help,
	})
}

// NewCounterVec returns a new Prometheus CounterVec with labels.
func NewCounterVec(name string, help string, labels []string) *prometheus.CounterVec {
	return promauto.NewCounterVec(prometheus.CounterOpts{
		Name: name,
		Help: help,
	}, labels)
}
