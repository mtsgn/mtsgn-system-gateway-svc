package handlers

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricsHandler handles metrics endpoint requests
func MetricsHandler() http.Handler {
	return promhttp.Handler()
}
