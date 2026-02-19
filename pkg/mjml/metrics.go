package mjml

import "github.com/zeromicro/go-zero/core/metric"

var (
	renderDuration = metric.NewHistogramVec(&metric.HistogramVecOpts{
		Namespace: "plat_mjml",
		Subsystem: "render",
		Name:      "duration_seconds",
		Help:      "Template render duration in seconds",
		Labels:    []string{"template"},
		Buckets:   []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1},
	})

	renderCacheHits = metric.NewCounterVec(&metric.CounterVecOpts{
		Namespace: "plat_mjml",
		Subsystem: "render",
		Name:      "cache_hits_total",
		Help:      "Render cache hits",
		Labels:    []string{"template"},
	})

	renderCacheMisses = metric.NewCounterVec(&metric.CounterVecOpts{
		Namespace: "plat_mjml",
		Subsystem: "render",
		Name:      "cache_misses_total",
		Help:      "Render cache misses",
		Labels:    []string{"template"},
	})
)
