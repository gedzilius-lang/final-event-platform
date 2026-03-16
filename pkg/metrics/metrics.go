// Package metrics provides a minimal Prometheus-compatible /metrics handler.
// Uses only stdlib — no prometheus/client_golang dependency.
// Emits: service_up, go_goroutines, go_info.
// Full instrumentation (request counters, latency histograms) can be layered on
// by importing prometheus/client_golang once added to go.work.
package metrics

import (
	"fmt"
	"net/http"
	"runtime"
)

// Handler returns an HTTP handler for GET /metrics that emits
// minimal Prometheus text format sufficient for M5 health checks.
func Handler(serviceName string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		var ms runtime.MemStats
		runtime.ReadMemStats(&ms)

		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		fmt.Fprintf(w, "# HELP service_up Service is running (1 = up)\n")
		fmt.Fprintf(w, "# TYPE service_up gauge\n")
		fmt.Fprintf(w, "service_up{service=%q} 1\n", serviceName)

		fmt.Fprintf(w, "# HELP go_goroutines Number of goroutines\n")
		fmt.Fprintf(w, "# TYPE go_goroutines gauge\n")
		fmt.Fprintf(w, "go_goroutines{service=%q} %d\n", serviceName, runtime.NumGoroutine())

		fmt.Fprintf(w, "# HELP go_memstats_alloc_bytes Heap bytes allocated\n")
		fmt.Fprintf(w, "# TYPE go_memstats_alloc_bytes gauge\n")
		fmt.Fprintf(w, "go_memstats_alloc_bytes{service=%q} %d\n", serviceName, ms.Alloc)

		fmt.Fprintf(w, "# HELP go_info Go runtime version\n")
		fmt.Fprintf(w, "# TYPE go_info gauge\n")
		fmt.Fprintf(w, "go_info{service=%q,version=%q} 1\n", serviceName, runtime.Version())
	}
}
