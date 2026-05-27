package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestHomeHandler(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	homeHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "Hello from") {
		t.Errorf("expected 'Hello from' in body, got: %s", rr.Body.String())
	}
}

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()
	healthHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "healthy") {
		t.Errorf("expected 'healthy' in body, got: %s", rr.Body.String())
	}
}

func TestMetricsEndpoint(t *testing.T) {
	// Trigger some metrics first
	handler := instrumentHandler("/metrics-test", homeHandler)
	handler(httptest.NewRecorder(), httptest.NewRequest("GET", "/metrics-test", nil))

	req := httptest.NewRequest("GET", "/metrics", nil)
	rr := httptest.NewRecorder()
	promhttp.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "http_requests_total") {
		t.Error("expected http_requests_total metric in /metrics output")
	}
	if !strings.Contains(body, "http_request_duration_seconds") {
		t.Error("expected http_request_duration_seconds metric in /metrics output")
	}
	if !strings.Contains(body, "app_info") {
		t.Error("expected app_info metric in /metrics output")
	}
}

func TestInstrumentHandler(t *testing.T) {
	httpRequestsTotal.Reset()

	handler := instrumentHandler("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	handler(rr, req)

	count := testutil.ToFloat64(httpRequestsTotal.WithLabelValues("GET", "/test", "200"))
	if count != 1 {
		t.Errorf("expected counter to be 1, got %f", count)
	}

	handler(httptest.NewRecorder(), httptest.NewRequest("GET", "/test", nil))
	count = testutil.ToFloat64(httpRequestsTotal.WithLabelValues("GET", "/test", "200"))
	if count != 2 {
		t.Errorf("expected counter to be 2, got %f", count)
	}
}
