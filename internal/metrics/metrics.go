package metrics

import (
    "net/http"

    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
    Approvals  prometheus.Counter
    Executions prometheus.Counter
    Failures   prometheus.Counter
    registry   *prometheus.Registry
}

func New(namespace string) *Metrics {
    reg := prometheus.NewRegistry()

    approvals := prometheus.NewCounter(prometheus.CounterOpts{
        Namespace: namespace,
        Name:      "approvals_total",
        Help:      "Total approvals submitted by the daemon.",
    })
    executions := prometheus.NewCounter(prometheus.CounterOpts{
        Namespace: namespace,
        Name:      "executions_total",
        Help:      "Total executions submitted by the daemon.",
    })
    failures := prometheus.NewCounter(prometheus.CounterOpts{
        Namespace: namespace,
        Name:      "failures_total",
        Help:      "Total failed operations in the daemon.",
    })

    reg.MustRegister(approvals, executions, failures)

    return &Metrics{
        Approvals:  approvals,
        Executions: executions,
        Failures:   failures,
        registry:   reg,
    }
}

func (m *Metrics) Handler() http.Handler {
    return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

func (m *Metrics) IncApprovals() {
    if m != nil {
        m.Approvals.Inc()
    }
}

func (m *Metrics) IncExecutions() {
    if m != nil {
        m.Executions.Inc()
    }
}

func (m *Metrics) IncFailures() {
    if m != nil {
        m.Failures.Inc()
    }
}
