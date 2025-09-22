package monitoring

import (
	"sync"
	"time"
)

const responseTimesBufferSize = 1000

type MetricsCollector struct {
	requestCount    int64
	responseTimes   []time.Duration
	responseIndex   int // circular buffer index
	responseCount   int // actual count of items in buffer
	errorCount      int64
	lastRequestTime time.Time
	startTime       time.Time
	mutex           sync.RWMutex

	endpointMetrics map[string]*EndpointMetrics
	statusCodes     map[int]int64
}

type EndpointMetrics struct {
	RequestCount    int64
	TotalTime       time.Duration
	MinTime         time.Duration
	MaxTime         time.Duration
	ErrorCount      int64
	LastRequestTime time.Time
}

type PerformanceMetrics struct {
	TotalRequests       int64                       `json:"total_requests"`
	AverageResponseTime time.Duration               `json:"average_response_time"`
	MinResponseTime     time.Duration               `json:"min_response_time"`
	MaxResponseTime     time.Duration               `json:"max_response_time"`
	RequestsPerSecond   float64                     `json:"requests_per_second"`
	ErrorRate           float64                     `json:"error_rate"`
	Uptime              time.Duration               `json:"uptime"`
	LastRequestTime     time.Time                   `json:"last_request_time"`
	EndpointMetrics     map[string]*EndpointMetrics `json:"endpoint_metrics"`
	StatusCodes         map[int]int64               `json:"status_codes"`
}

func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		responseTimes:   make([]time.Duration, responseTimesBufferSize),
		responseIndex:   0,
		responseCount:   0,
		endpointMetrics: make(map[string]*EndpointMetrics),
		statusCodes:     make(map[int]int64),
		startTime:       time.Now(),
	}
}

func (mc *MetricsCollector) RecordRequest(endpoint string, responseTime time.Duration, statusCode int) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	mc.requestCount++
	mc.lastRequestTime = time.Now()

	if mc.responseCount < responseTimesBufferSize {
		mc.responseTimes[mc.responseCount] = responseTime
		mc.responseCount++
	} else {
		mc.responseTimes[mc.responseIndex] = responseTime
		mc.responseIndex = (mc.responseIndex + 1) % responseTimesBufferSize
	}

	if statusCode >= 400 {
		mc.errorCount++
	}

	mc.statusCodes[statusCode]++

	if mc.endpointMetrics[endpoint] == nil {
		mc.endpointMetrics[endpoint] = &EndpointMetrics{
			MinTime: responseTime,
			MaxTime: responseTime,
		}
	}

	epMetrics := mc.endpointMetrics[endpoint]
	epMetrics.RequestCount++
	epMetrics.TotalTime += responseTime
	epMetrics.LastRequestTime = time.Now()

	if responseTime < epMetrics.MinTime {
		epMetrics.MinTime = responseTime
	}
	if responseTime > epMetrics.MaxTime {
		epMetrics.MaxTime = responseTime
	}

	if statusCode >= 400 {
		epMetrics.ErrorCount++
	}
}

func (mc *MetricsCollector) GetMetrics() *PerformanceMetrics {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	var avgResponseTime time.Duration
	var minResponseTime time.Duration
	var maxResponseTime time.Duration

	if mc.responseCount > 0 {
		total := time.Duration(0)
		// Compute the correct order for the circular buffer
		var firstIdx int
		if mc.responseCount < responseTimesBufferSize {
			firstIdx = 0
		} else {
			firstIdx = mc.responseIndex
		}

		// Get the first value for min/max initialization
		minResponseTime = mc.responseTimes[firstIdx]
		maxResponseTime = mc.responseTimes[firstIdx]

		for i := 0; i < mc.responseCount; i++ {
			idx := (firstIdx + i) % responseTimesBufferSize
			rt := mc.responseTimes[idx]
			total += rt
			if rt < minResponseTime {
				minResponseTime = rt
			}
			if rt > maxResponseTime {
				maxResponseTime = rt
			}
		}

		avgResponseTime = total / time.Duration(mc.responseCount)
	}

	uptime := time.Since(mc.startTime)
	var requestsPerSecond float64
	if uptime.Seconds() > 0 {
		requestsPerSecond = float64(mc.requestCount) / uptime.Seconds()
	}

	var errorRate float64
	if mc.requestCount > 0 {
		errorRate = float64(mc.errorCount) / float64(mc.requestCount) * 100
	}

	endpointMetrics := make(map[string]*EndpointMetrics)
	for endpoint, metrics := range mc.endpointMetrics {
		endpointMetrics[endpoint] = &EndpointMetrics{
			RequestCount:    metrics.RequestCount,
			TotalTime:       metrics.TotalTime,
			MinTime:         metrics.MinTime,
			MaxTime:         metrics.MaxTime,
			ErrorCount:      metrics.ErrorCount,
			LastRequestTime: metrics.LastRequestTime,
		}
	}

	statusCodes := make(map[int]int64)
	for code, count := range mc.statusCodes {
		statusCodes[code] = count
	}

	return &PerformanceMetrics{
		TotalRequests:       mc.requestCount,
		AverageResponseTime: avgResponseTime,
		MinResponseTime:     minResponseTime,
		MaxResponseTime:     maxResponseTime,
		RequestsPerSecond:   requestsPerSecond,
		ErrorRate:           errorRate,
		Uptime:              uptime,
		LastRequestTime:     mc.lastRequestTime,
		EndpointMetrics:     endpointMetrics,
		StatusCodes:         statusCodes,
	}
}

func (mc *MetricsCollector) Reset() {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	mc.requestCount = 0
	// Reset circular buffer state
	mc.responseIndex = 0
	mc.responseCount = 0
	// Optionally clear the buffer (not strictly necessary, but for hygiene)
	for i := range mc.responseTimes {
		mc.responseTimes[i] = 0
	}
	mc.errorCount = 0
	mc.lastRequestTime = time.Time{}
	mc.startTime = time.Now()
	mc.endpointMetrics = make(map[string]*EndpointMetrics)
	mc.statusCodes = make(map[int]int64)
}

func (mc *MetricsCollector) GetEndpointMetrics(endpoint string) *EndpointMetrics {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	if metrics, exists := mc.endpointMetrics[endpoint]; exists {
		return &EndpointMetrics{
			RequestCount:    metrics.RequestCount,
			TotalTime:       metrics.TotalTime,
			MinTime:         metrics.MinTime,
			MaxTime:         metrics.MaxTime,
			ErrorCount:      metrics.ErrorCount,
			LastRequestTime: metrics.LastRequestTime,
		}
	}

	return nil
}
