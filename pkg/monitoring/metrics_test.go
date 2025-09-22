package monitoring

import (
	"testing"
	"time"
)

func TestMetricsCollector_RecordRequest(t *testing.T) {
	collector := NewMetricsCollector()

	collector.RecordRequest("/api/v1/ingest", 100*time.Millisecond, 201)
	collector.RecordRequest("/api/v1/stats", 50*time.Millisecond, 200)
	collector.RecordRequest("/api/v1/ingest", 150*time.Millisecond, 400)

	metrics := collector.GetMetrics()

	if metrics.TotalRequests != 3 {
		t.Errorf("Expected 3 total requests, got %d", metrics.TotalRequests)
	}

	if metrics.ErrorRate < 33.0 || metrics.ErrorRate > 34.0 {
		t.Errorf("Expected error rate ~33.33%%, got %.2f%%", metrics.ErrorRate)
	}

	if len(metrics.EndpointMetrics) != 2 {
		t.Errorf("Expected 2 endpoints, got %d", len(metrics.EndpointMetrics))
	}

	ingestMetrics := metrics.EndpointMetrics["/api/v1/ingest"]
	if ingestMetrics == nil {
		t.Fatal("Expected ingest endpoint metrics, got nil")
		return
	}

	if ingestMetrics.RequestCount != 2 {
		t.Errorf("Expected 2 ingest requests, got %d", ingestMetrics.RequestCount)
	}

	if ingestMetrics.ErrorCount != 1 {
		t.Errorf("Expected 1 ingest error, got %d", ingestMetrics.ErrorCount)
	}
}

func TestMetricsCollector_ResponseTimes(t *testing.T) {
	collector := NewMetricsCollector()

	collector.RecordRequest("/test", 100*time.Millisecond, 200)
	collector.RecordRequest("/test", 200*time.Millisecond, 200)
	collector.RecordRequest("/test", 300*time.Millisecond, 200)

	metrics := collector.GetMetrics()

	if metrics.AverageResponseTime != 200*time.Millisecond {
		t.Errorf("Expected average response time 200ms, got %v", metrics.AverageResponseTime)
	}

	if metrics.MinResponseTime != 100*time.Millisecond {
		t.Errorf("Expected min response time 100ms, got %v", metrics.MinResponseTime)
	}

	if metrics.MaxResponseTime != 300*time.Millisecond {
		t.Errorf("Expected max response time 300ms, got %v", metrics.MaxResponseTime)
	}
}

func TestMetricsCollector_RequestsPerSecond(t *testing.T) {
	collector := NewMetricsCollector()

	for i := 0; i < 10; i++ {
		collector.RecordRequest("/test", 50*time.Millisecond, 200)
		time.Sleep(10 * time.Millisecond)
	}

	metrics := collector.GetMetrics()

	if metrics.RequestsPerSecond < 50 || metrics.RequestsPerSecond > 200 {
		t.Errorf("Expected RPS around 100, got %.2f", metrics.RequestsPerSecond)
	}
}

func TestMetricsCollector_StatusCodes(t *testing.T) {
	collector := NewMetricsCollector()

	collector.RecordRequest("/test", 100*time.Millisecond, 200)
	collector.RecordRequest("/test", 100*time.Millisecond, 201)
	collector.RecordRequest("/test", 100*time.Millisecond, 400)
	collector.RecordRequest("/test", 100*time.Millisecond, 500)

	metrics := collector.GetMetrics()

	if metrics.StatusCodes[200] != 1 {
		t.Errorf("Expected 1 status 200, got %d", metrics.StatusCodes[200])
	}

	if metrics.StatusCodes[201] != 1 {
		t.Errorf("Expected 1 status 201, got %d", metrics.StatusCodes[201])
	}

	if metrics.StatusCodes[400] != 1 {
		t.Errorf("Expected 1 status 400, got %d", metrics.StatusCodes[400])
	}

	if metrics.StatusCodes[500] != 1 {
		t.Errorf("Expected 1 status 500, got %d", metrics.StatusCodes[500])
	}
}

func TestMetricsCollector_Reset(t *testing.T) {
	collector := NewMetricsCollector()

	collector.RecordRequest("/test", 100*time.Millisecond, 200)
	collector.RecordRequest("/test", 200*time.Millisecond, 400)

	metrics := collector.GetMetrics()
	if metrics.TotalRequests != 2 {
		t.Errorf("Expected 2 requests before reset, got %d", metrics.TotalRequests)
	}

	collector.Reset()

	metrics = collector.GetMetrics()
	if metrics.TotalRequests != 0 {
		t.Errorf("Expected 0 requests after reset, got %d", metrics.TotalRequests)
	}

	if len(metrics.EndpointMetrics) != 0 {
		t.Errorf("Expected 0 endpoints after reset, got %d", len(metrics.EndpointMetrics))
	}

	if len(metrics.StatusCodes) != 0 {
		t.Errorf("Expected 0 status codes after reset, got %d", len(metrics.StatusCodes))
	}
}

func TestMetricsCollector_GetEndpointMetrics(t *testing.T) {
	collector := NewMetricsCollector()

	collector.RecordRequest("/api/v1/ingest", 100*time.Millisecond, 201)
	collector.RecordRequest("/api/v1/ingest", 200*time.Millisecond, 400)

	metrics := collector.GetEndpointMetrics("/api/v1/ingest")
	if metrics == nil {
		t.Fatal("metrics is nil, possible nil pointer dereference")
		return
	}

	if metrics.RequestCount != 2 {
		t.Errorf("Expected 2 requests, got %d", metrics.RequestCount)
	}

	if metrics.ErrorCount != 1 {
		t.Errorf("Expected 1 error, got %d", metrics.ErrorCount)
	}

	if metrics.MinTime != 100*time.Millisecond {
		t.Errorf("Expected min time 100ms, got %v", metrics.MinTime)
	}

	if metrics.MaxTime != 200*time.Millisecond {
		t.Errorf("Expected max time 200ms, got %v", metrics.MaxTime)
	}

	metrics = collector.GetEndpointMetrics("/nonexistent")
	if metrics != nil {
		t.Error("Expected nil for nonexistent endpoint")
	}
}

func TestMetricsCollector_ConcurrentAccess(t *testing.T) {
	collector := NewMetricsCollector()

	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				collector.RecordRequest("/test", 50*time.Millisecond, 200)
			}
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	metrics := collector.GetMetrics()
	expectedRequests := int64(10 * 100)

	if metrics.TotalRequests != expectedRequests {
		t.Errorf("Expected %d total requests, got %d", expectedRequests, metrics.TotalRequests)
	}
}

func BenchmarkMetricsCollector_RecordRequest(b *testing.B) {
	collector := NewMetricsCollector()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			collector.RecordRequest("/benchmark", 100*time.Millisecond, 200)
		}
	})
}

func BenchmarkMetricsCollector_GetMetrics(b *testing.B) {
	collector := NewMetricsCollector()

	// Pre-populate with data
	for i := 0; i < 1000; i++ {
		collector.RecordRequest("/benchmark", 100*time.Millisecond, 200)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.GetMetrics()
	}
}
