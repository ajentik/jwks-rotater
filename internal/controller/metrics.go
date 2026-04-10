package controller

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	rotationTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "jwks_rotation_total",
			Help: "Total number of key rotations performed",
		},
		[]string{"namespace", "name"},
	)

	rotationErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "jwks_rotation_errors_total",
			Help: "Total number of rotation failures",
		},
		[]string{"namespace", "name"},
	)

	activeKeys = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "jwks_active_keys",
			Help: "Current number of keys in the JWKS",
		},
		[]string{"namespace", "name"},
	)

	oldestKeyAge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "jwks_oldest_key_age_seconds",
			Help: "Age of the oldest key in seconds",
		},
		[]string{"namespace", "name"},
	)

	reconcileDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "jwks_reconcile_duration_seconds",
			Help:    "Reconciliation loop duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"controller"},
	)
)

func init() {
	metrics.Registry.MustRegister(
		rotationTotal,
		rotationErrorsTotal,
		activeKeys,
		oldestKeyAge,
		reconcileDuration,
	)
}
