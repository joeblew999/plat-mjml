package delivery

import "github.com/zeromicro/go-zero/core/metric"

var (
	emailsSent = metric.NewCounterVec(&metric.CounterVecOpts{
		Namespace: "plat_mjml",
		Subsystem: "delivery",
		Name:      "emails_sent_total",
		Help:      "Total emails sent successfully",
		Labels:    []string{"template"},
	})

	emailsFailed = metric.NewCounterVec(&metric.CounterVecOpts{
		Namespace: "plat_mjml",
		Subsystem: "delivery",
		Name:      "emails_failed_total",
		Help:      "Total emails failed permanently",
		Labels:    []string{"template", "reason"},
	})

	emailsRetried = metric.NewCounterVec(&metric.CounterVecOpts{
		Namespace: "plat_mjml",
		Subsystem: "delivery",
		Name:      "emails_retried_total",
		Help:      "Total email delivery retries",
		Labels:    []string{"template"},
	})

	deliveryDuration = metric.NewHistogramVec(&metric.HistogramVecOpts{
		Namespace: "plat_mjml",
		Subsystem: "delivery",
		Name:      "duration_seconds",
		Help:      "Email delivery duration in seconds",
		Labels:    []string{"template"},
		Buckets:   []float64{0.1, 0.5, 1, 2, 5, 10, 30},
	})

	queueDepth = metric.NewGaugeVec(&metric.GaugeVecOpts{
		Namespace: "plat_mjml",
		Subsystem: "queue",
		Name:      "depth",
		Help:      "Current queue depth by status",
		Labels:    []string{"status"},
	})
)
