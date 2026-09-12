package telemetry

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total HTTP requests served by a service.",
		},
		[]string{"service", "method", "route", "status"},
	)
	httpDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "http_request_duration_seconds",
			Help: "HTTP request duration served by a service.",
		},
		[]string{"service", "method", "route", "status"},
	)
)

func init() {
	prometheus.MustRegister(httpRequests, httpDuration)
}

// HTTPMetrics records status and duration after each request completes.
func HTTPMetrics(service string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		startedAt := time.Now()
		ctx.Next()

		route := ctx.FullPath()
		if route == "" {
			route = "unmatched"
		}
		status := strconv.Itoa(ctx.Writer.Status())
		httpRequests.WithLabelValues(service, ctx.Request.Method, route, status).Inc()
		httpDuration.WithLabelValues(service, ctx.Request.Method, route, status).Observe(time.Since(startedAt).Seconds())
	}
}

// RegisterMetricsRoute exposes the Prometheus scrape endpoint.
func RegisterMetricsRoute(router gin.IRouter) {
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))
}
