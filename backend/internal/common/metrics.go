package common

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metriky jsou package-level proměnné — singleton, bezpečné pro concurrent přístup.
// promauto.New* = automatická registrace do default Prometheus registry.

var (
	// TelemetryProcessed počítá zprávy úspěšně zpracované Processorem
	TelemetryProcessed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "tracking_telemetry_processed_total",
		Help: "Total number of telemetry messages processed by processor",
	})

	// AlertsGenerated počítá vygenerované alerty dle typu
	AlertsGenerated = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "tracking_alerts_generated_total",
		Help: "Total number of alerts generated, by alert type",
	}, []string{"type"})

	// ProcessingDuration měří latenci zpracování jedné telemetrické zprávy
	ProcessingDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "tracking_processing_duration_seconds",
		Help:    "Duration of telemetry processing in seconds",
		Buckets: prometheus.DefBuckets,
	})

	// IngestRequests počítá příchozí gRPC requesty do Ingest service
	IngestRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "tracking_ingest_requests_total",
		Help: "Total number of gRPC requests received by ingest service",
	}, []string{"success"})
)

// StartMetricsServer spustí HTTP server na daném portu a obsluhuje /metrics endpoint.
// Voláno v goroutině z main() každé služby která potřebuje metriky.
func StartMetricsServer(port string) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	_ = http.ListenAndServe(":"+port, mux)
}
