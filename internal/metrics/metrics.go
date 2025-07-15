package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Registry struct {
	registry        *prometheus.Registry
	approvalsTotal  prometheus.Counter
	executionsTotal prometheus.Counter
	failuresTotal   prometheus.Counter
}

func NewRegistry(namespace string) *Registry {
	reg := prometheus.NewRegistry()
	reg.MustRegister(prometheus.NewGoCollector())
	reg.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	approvals := prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "approvals_total",
		Help:      "Total approvals sent",
	})
	executions := prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "executions_total",
		Help:      "Total executeBatch calls sent",
	})
	failures := prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "failures_total",
		Help:      "Total transaction failures",
	})

	reg.MustRegister(approvals, executions, failures)

	return &Registry{
		registry:        reg,
		approvalsTotal:  approvals,
		executionsTotal: executions,
		failuresTotal:   failures,
	}
}

func (r *Registry) Handler() http.Handler {
	return promhttp.HandlerFor(r.registry, promhttp.HandlerOpts{})
}

func (r *Registry) IncApprovals() {
	r.approvalsTotal.Inc()
}

func (r *Registry) IncExecutions() {
	r.executionsTotal.Inc()
}

func (r *Registry) IncFailures() {
	r.failuresTotal.Inc()
}

func register(reg *prometheus.Registry, collector prometheus.Collector) {
	if err := reg.Register(collector); err != nil {
		if _, ok := err.(prometheus.AlreadyRegisteredError); ok {
			return
		}
	}
}
