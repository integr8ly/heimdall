package customMetrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	RegistryCallsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "heimdall_registry_calls_total",
			Help: "Number of registry calls",
		})
	RegistryCallsSuccess = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "heimdall_registry_calls_success",
			Help: "Number of successful registry calls",
		})
	RegistryCallsFailure = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "heimdall_registry_calls_failure",
			Help: "Number of failed registry calls",
		})
)

func init() {
	metrics.Registry.MustRegister(RegistryCallsTotal)
	metrics.Registry.MustRegister(RegistryCallsSuccess)
	metrics.Registry.MustRegister(RegistryCallsFailure)
}
