// Package telemetry implements the first slice of
// docs/07-operations/observability.md's already-decided target
// (OpenTelemetry SDK metrics exported as Prometheus exposition format;
// Grafana dashboards read them) — see adr/0020-observability-prometheus-loki-grafana.md
// for what this covers (HTTP + business metrics) and what it deliberately
// does not (distributed tracing, alerting, per-tenant breakdowns).
package telemetry

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Recorder holds every metric this service exposes at GET /metrics, on a
// private registry (not prometheus.DefaultRegisterer) so nothing else in the
// process can accidentally register into a shared global one. Every
// recording method is nil-receiver-safe so callers that don't care about
// metrics (most unit/integration tests) can pass a nil *Recorder instead of
// threading a real one through test setup.
type Recorder struct {
	registry *prometheus.Registry

	httpRequestsTotal   *prometheus.CounterVec
	httpRequestDuration *prometheus.HistogramVec

	ingestionEventsTotal *prometheus.CounterVec
	caseOutcomesTotal    *prometheus.CounterVec

	actionRequestedTotal *prometheus.CounterVec
	approvalDecidedTotal *prometheus.CounterVec
	actionExecutedTotal  *prometheus.CounterVec
	actionReleasedTotal  *prometheus.CounterVec
	actionEscalatedTotal prometheus.Counter
	connectorCallTotal   *prometheus.CounterVec

	schedulerSweepDuration prometheus.Histogram
	wazuhPollDuration      prometheus.Histogram
}

// NewRecorder builds and registers every metric. Call once at startup.
func NewRecorder() *Recorder {
	reg := prometheus.NewRegistry()
	r := &Recorder{
		registry: reg,
		httpRequestsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "soar_http_requests_total",
			Help: "Total HTTP requests handled, by method, route template and status.",
		}, []string{"method", "route", "status"}),
		httpRequestDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "soar_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds, by method and route template.",
			Buckets: prometheus.DefBuckets,
		}, []string{"method", "route"}),
		ingestionEventsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "soar_ingestion_events_total",
			Help: "Events processed by POST /v1/events, by outcome (accepted, duplicate, rejected, error).",
		}, []string{"outcome"}),
		caseOutcomesTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "soar_case_outcomes_total",
			Help: "Case outcomes from event ingestion, by outcome (created vs correlated onto an existing case).",
		}, []string{"outcome"}),
		actionRequestedTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "soar_action_requested_total",
			Help: "Response actions requested (manually or auto-triggered), by intent.",
		}, []string{"intent"}),
		approvalDecidedTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "soar_approval_decided_total",
			Help: "Approval decisions recorded, by outcome (approved, rejected).",
		}, []string{"outcome"}),
		actionExecutedTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "soar_action_executed_total",
			Help: "Action execute attempts, by outcome (success, denied, error).",
		}, []string{"outcome"}),
		actionReleasedTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "soar_action_released_total",
			Help: "Actions released (rolled back), by mode (manual, scheduled).",
		}, []string{"mode"}),
		actionEscalatedTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "soar_action_release_escalated_total",
			Help: "Actions whose automatic release retries were exhausted and were escalated for manual handling.",
		}),
		connectorCallTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "soar_connector_call_total",
			Help: "Network connector calls, by connector version and outcome (success, error).",
		}, []string{"connector", "outcome"}),
		schedulerSweepDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "soar_scheduler_sweep_duration_seconds",
			Help:    "Duration of each TTL-rollback scheduler sweep.",
			Buckets: prometheus.DefBuckets,
		}),
		wazuhPollDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "soar_wazuhpoll_duration_seconds",
			Help:    "Duration of each Wazuh indexer poll.",
			Buckets: prometheus.DefBuckets,
		}),
	}
	reg.MustRegister(
		r.httpRequestsTotal, r.httpRequestDuration,
		r.ingestionEventsTotal, r.caseOutcomesTotal,
		r.actionRequestedTotal, r.approvalDecidedTotal, r.actionExecutedTotal,
		r.actionReleasedTotal, r.actionEscalatedTotal, r.connectorCallTotal,
		r.schedulerSweepDuration, r.wazuhPollDuration,
	)
	return r
}

// Handler serves the Prometheus exposition format for GET /metrics.
func (r *Recorder) Handler() http.Handler {
	return promhttp.HandlerFor(r.registry, promhttp.HandlerOpts{})
}

// Middleware records one request-count and one duration observation per
// request, labeled by the matched route template (c.FullPath(), not the raw
// path) to keep label cardinality bounded — an unmatched route (e.g. a 404
// for an unknown path) is labeled "unmatched".
func (r *Recorder) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if r == nil {
			c.Next()
			return
		}
		start := time.Now()
		c.Next()
		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}
		status := strconv.Itoa(c.Writer.Status())
		r.httpRequestsTotal.WithLabelValues(c.Request.Method, route, status).Inc()
		r.httpRequestDuration.WithLabelValues(c.Request.Method, route).Observe(time.Since(start).Seconds())
	}
}

func (r *Recorder) IngestionOutcome(outcome string) {
	if r == nil {
		return
	}
	r.ingestionEventsTotal.WithLabelValues(outcome).Inc()
}

func (r *Recorder) CaseOutcome(outcome string) {
	if r == nil {
		return
	}
	r.caseOutcomesTotal.WithLabelValues(outcome).Inc()
}

func (r *Recorder) ActionRequested(intent string) {
	if r == nil {
		return
	}
	r.actionRequestedTotal.WithLabelValues(intent).Inc()
}

func (r *Recorder) ApprovalDecided(outcome string) {
	if r == nil {
		return
	}
	r.approvalDecidedTotal.WithLabelValues(outcome).Inc()
}

func (r *Recorder) ActionExecuted(outcome string) {
	if r == nil {
		return
	}
	r.actionExecutedTotal.WithLabelValues(outcome).Inc()
}

func (r *Recorder) ActionReleased(mode string) {
	if r == nil {
		return
	}
	r.actionReleasedTotal.WithLabelValues(mode).Inc()
}

func (r *Recorder) ActionEscalated() {
	if r == nil {
		return
	}
	r.actionEscalatedTotal.Inc()
}

func (r *Recorder) ConnectorCall(connector, outcome string) {
	if r == nil {
		return
	}
	r.connectorCallTotal.WithLabelValues(connector, outcome).Inc()
}

func (r *Recorder) SchedulerSweep(d time.Duration) {
	if r == nil {
		return
	}
	r.schedulerSweepDuration.Observe(d.Seconds())
}

func (r *Recorder) WazuhPoll(d time.Duration) {
	if r == nil {
		return
	}
	r.wazuhPollDuration.Observe(d.Seconds())
}
