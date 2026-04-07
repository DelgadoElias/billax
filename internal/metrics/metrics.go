package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// HTTP Metrics
var (
	// HTTPRequestsTotal tracks the total number of HTTP requests by method, path, and status code
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "payd_http_requests_total",
			Help: "Total HTTP requests by method, path, and status_code",
		},
		[]string{"method", "path", "status_code"},
	)

	// HTTPRequestDuration tracks request latency in seconds
	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "payd_http_request_duration_seconds",
			Help:    "HTTP request latency histogram by method and path",
			Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5},
		},
		[]string{"method", "path"},
	)

	// HTTPInFlightRequests tracks the current number of in-flight HTTP requests
	HTTPInFlightRequests = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "payd_http_in_flight_requests",
			Help: "Current number of in-flight HTTP requests",
		},
	)
)

// Business Metrics
var (
	// PaymentChargeAttempts tracks payment charge attempts by provider and outcome (success/failure)
	PaymentChargeAttempts = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "payd_payment_charge_attempts_total",
			Help: "Payment charge attempts by provider and outcome",
		},
		[]string{"provider", "outcome"},
	)

	// ActiveSubscriptions tracks the current number of subscriptions by status
	ActiveSubscriptions = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "payd_active_subscriptions",
			Help: "Current number of subscriptions by status",
		},
		[]string{"status"},
	)
)
