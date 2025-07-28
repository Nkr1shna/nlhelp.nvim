package server

import (
	"sync"
	"time"
)

// PerformanceMetrics holds performance-related metrics
type PerformanceMetrics struct {
	mu                  sync.RWMutex
	QueryCount          int64         `json:"query_count"`
	AverageResponseTime time.Duration `json:"average_response_time"`
	LastResponseTime    time.Duration `json:"last_response_time"`
	MaxResponseTime     time.Duration `json:"max_response_time"`
	MinResponseTime     time.Duration `json:"min_response_time"`
	ErrorCount          int64         `json:"error_count"`
	LastErrorTime       time.Time     `json:"last_error_time,omitempty"`
	SuccessfulQueries   int64         `json:"successful_queries"`
	FailedQueries       int64         `json:"failed_queries"`
	TotalResponseTime   time.Duration `json:"-"` // Used for calculating average
	StartTime           time.Time     `json:"start_time"`
}

// MetricsCollector collects and manages performance metrics
type MetricsCollector struct {
	metrics *PerformanceMetrics
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		metrics: &PerformanceMetrics{
			StartTime:       time.Now(),
			MinResponseTime: time.Duration(0), // Will be set on first query
		},
	}
}

// RecordQuery records metrics for a query operation
func (mc *MetricsCollector) RecordQuery(duration time.Duration, success bool) {
	mc.metrics.mu.Lock()
	defer mc.metrics.mu.Unlock()

	mc.metrics.QueryCount++
	mc.metrics.LastResponseTime = duration
	mc.metrics.TotalResponseTime += duration

	// Update min/max response times
	if mc.metrics.MinResponseTime == 0 || duration < mc.metrics.MinResponseTime {
		mc.metrics.MinResponseTime = duration
	}
	if duration > mc.metrics.MaxResponseTime {
		mc.metrics.MaxResponseTime = duration
	}

	// Calculate average response time
	mc.metrics.AverageResponseTime = mc.metrics.TotalResponseTime / time.Duration(mc.metrics.QueryCount)

	// Update success/failure counts
	if success {
		mc.metrics.SuccessfulQueries++
	} else {
		mc.metrics.FailedQueries++
		mc.metrics.ErrorCount++
		mc.metrics.LastErrorTime = time.Now()
	}
}

// GetMetrics returns a copy of the current metrics
func (mc *MetricsCollector) GetMetrics() PerformanceMetrics {
	mc.metrics.mu.RLock()
	defer mc.metrics.mu.RUnlock()

	// Return a copy to avoid race conditions
	return PerformanceMetrics{
		QueryCount:          mc.metrics.QueryCount,
		AverageResponseTime: mc.metrics.AverageResponseTime,
		LastResponseTime:    mc.metrics.LastResponseTime,
		MaxResponseTime:     mc.metrics.MaxResponseTime,
		MinResponseTime:     mc.metrics.MinResponseTime,
		ErrorCount:          mc.metrics.ErrorCount,
		LastErrorTime:       mc.metrics.LastErrorTime,
		SuccessfulQueries:   mc.metrics.SuccessfulQueries,
		FailedQueries:       mc.metrics.FailedQueries,
		StartTime:           mc.metrics.StartTime,
	}
}

// Reset resets all metrics
func (mc *MetricsCollector) Reset() {
	mc.metrics.mu.Lock()
	defer mc.metrics.mu.Unlock()

	mc.metrics.QueryCount = 0
	mc.metrics.AverageResponseTime = 0
	mc.metrics.LastResponseTime = 0
	mc.metrics.MaxResponseTime = 0
	mc.metrics.MinResponseTime = 0
	mc.metrics.ErrorCount = 0
	mc.metrics.LastErrorTime = time.Time{}
	mc.metrics.SuccessfulQueries = 0
	mc.metrics.FailedQueries = 0
	mc.metrics.TotalResponseTime = 0
	mc.metrics.StartTime = time.Now()
}

// DetailedHealthStatus extends HealthStatus with more comprehensive information
type DetailedHealthStatus struct {
	HealthStatus
	Metrics      PerformanceMetrics          `json:"metrics"`
	Dependencies map[string]DependencyStatus `json:"dependencies"`
	SystemInfo   SystemInfo                  `json:"system_info"`
	Uptime       time.Duration               `json:"uptime"`
}

// DependencyStatus represents the status of a specific dependency
type DependencyStatus struct {
	Name           string                 `json:"name"`
	Status         string                 `json:"status"`
	LastCheck      time.Time              `json:"last_check"`
	ResponseTime   time.Duration          `json:"response_time,omitempty"`
	Error          string                 `json:"error,omitempty"`
	Version        string                 `json:"version,omitempty"`
	AdditionalInfo map[string]interface{} `json:"additional_info,omitempty"`
}

// SystemInfo holds system-level information
type SystemInfo struct {
	Version   string    `json:"version"`
	BuildTime string    `json:"build_time,omitempty"`
	GoVersion string    `json:"go_version,omitempty"`
	StartTime time.Time `json:"start_time"`
}

// HealthMonitor provides comprehensive health monitoring capabilities
type HealthMonitor struct {
	metricsCollector *MetricsCollector
	systemInfo       SystemInfo
	startTime        time.Time
}

// NewHealthMonitor creates a new health monitor
func NewHealthMonitor() *HealthMonitor {
	return &HealthMonitor{
		metricsCollector: NewMetricsCollector(),
		systemInfo: SystemInfo{
			Version:   "1.0.0", // This could be set from build flags
			StartTime: time.Now(),
		},
		startTime: time.Now(),
	}
}

// GetMetricsCollector returns the metrics collector
func (hm *HealthMonitor) GetMetricsCollector() *MetricsCollector {
	return hm.metricsCollector
}

// GetDetailedHealthStatus performs a comprehensive health check and returns detailed status
func (hm *HealthMonitor) GetDetailedHealthStatus(rpcService *RPCService) *DetailedHealthStatus {
	// Get basic health status
	var basicHealth HealthStatus
	rpcService.HealthCheck(&HealthCheckArgs{}, &basicHealth)

	// Get performance metrics
	metrics := hm.metricsCollector.GetMetrics()

	// Check dependencies with timing and detailed information
	dependencies := make(map[string]DependencyStatus)

	// Check RAG Agent with detailed stats
	if rpcService.ragAgent != nil {
		start := time.Now()
		err := rpcService.ragAgent.HealthCheck()
		responseTime := time.Since(start)

		status := DependencyStatus{
			Name:         "rag_agent",
			LastCheck:    time.Now(),
			ResponseTime: responseTime,
			Version:      "1.0.0",
		}

		if err != nil {
			status.Status = "unhealthy"
			status.Error = err.Error()
		} else {
			status.Status = "healthy"
			// Get RAG agent statistics if available
			if ragStats := rpcService.ragAgent.GetStats(); ragStats != nil {
				status.AdditionalInfo = ragStats
			}
		}

		dependencies["rag_agent"] = status
	} else {
		dependencies["rag_agent"] = DependencyStatus{
			Name:      "rag_agent",
			Status:    "not_initialized",
			LastCheck: time.Now(),
			Error:     "RAG agent not initialized",
		}
	}

	// Check Vector Database (ChromaDB) with connection details
	if rpcService.vectorDB != nil {
		start := time.Now()
		err := rpcService.vectorDB.HealthCheck()
		responseTime := time.Since(start)

		status := DependencyStatus{
			Name:         "vector_db",
			LastCheck:    time.Now(),
			ResponseTime: responseTime,
			Version:      "chromadb",
		}

		if err != nil {
			status.Status = "unhealthy"
			status.Error = err.Error()
		} else {
			status.Status = "healthy"
			status.AdditionalInfo = map[string]interface{}{
				"type":        "chromadb",
				"host":        "localhost",
				"port":        8000,
				"collections": "keybindings",
			}
		}

		dependencies["vector_db"] = status
	} else {
		dependencies["vector_db"] = DependencyStatus{
			Name:      "vector_db",
			Status:    "not_initialized",
			LastCheck: time.Now(),
			Error:     "Vector database not initialized",
		}
	}

	// Check LLM Client (Ollama) with model availability and service status
	if rpcService.llmClient != nil {
		start := time.Now()
		err := rpcService.llmClient.HealthCheck()
		responseTime := time.Since(start)

		status := DependencyStatus{
			Name:         "llm_client",
			LastCheck:    time.Now(),
			ResponseTime: responseTime,
			Version:      "ollama",
		}

		if err != nil {
			status.Status = "unhealthy"
			status.Error = err.Error()
			status.AdditionalInfo = map[string]interface{}{
				"service_url":    "http://localhost:11434",
				"service_status": "unavailable",
			}
		} else {
			status.Status = "healthy"
			additionalInfo := map[string]interface{}{
				"service_url":    "http://localhost:11434",
				"service_status": "running",
			}

			// Try to get model info
			if modelInfo, err := rpcService.llmClient.GetModelInfo(); err == nil {
				status.Version = modelInfo.Name
				additionalInfo["model_name"] = modelInfo.Name
				additionalInfo["model_status"] = modelInfo.Status
				additionalInfo["model_size"] = modelInfo.Size
				additionalInfo["model_version"] = modelInfo.Version
			} else {
				additionalInfo["model_status"] = "no_model_loaded"
				additionalInfo["model_error"] = err.Error()
			}

			status.AdditionalInfo = additionalInfo
		}

		dependencies["llm_client"] = status
	} else {
		dependencies["llm_client"] = DependencyStatus{
			Name:      "llm_client",
			Status:    "not_initialized",
			LastCheck: time.Now(),
			Error:     "LLM client not initialized",
		}
	}

	return &DetailedHealthStatus{
		HealthStatus: basicHealth,
		Metrics:      metrics,
		Dependencies: dependencies,
		SystemInfo:   hm.systemInfo,
		Uptime:       time.Since(hm.startTime),
	}
}
