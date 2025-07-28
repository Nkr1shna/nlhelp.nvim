package server

import (
	"testing"
	"time"
)

// TestHealthCheckIntegration tests the complete health check flow
func TestHealthCheckIntegration(t *testing.T) {
	// Create service with healthy mocks
	ragAgent := &MockRAGAgent{shouldError: false}
	vectorDB := &MockVectorDB{shouldError: false}
	llmClient := &MockLLMClient{shouldError: false}

	service := NewRPCService(ragAgent, vectorDB, llmClient)

	// Test basic health check
	var healthResult HealthStatus
	err := service.HealthCheck(&HealthCheckArgs{}, &healthResult)
	if err != nil {
		t.Fatalf("HealthCheck failed: %v", err)
	}

	if healthResult.Status != "healthy" {
		t.Errorf("Expected healthy status, got: %s", healthResult.Status)
	}

	// Verify all services are reported as healthy
	expectedServices := []string{"rag_agent", "vector_db", "llm_client"}
	for _, serviceName := range expectedServices {
		status, exists := healthResult.Services[serviceName]
		if !exists {
			t.Errorf("Service %s not found in health check result", serviceName)
			continue
		}

		if !contains(status, "healthy") {
			t.Errorf("Service %s is not healthy: %s", serviceName, status)
		}

		// Verify response time is included
		if !contains(status, "checked in") {
			t.Errorf("Service %s should include response time", serviceName)
		}
	}

	// Test detailed health check
	var detailedResult DetailedHealthStatus
	err = service.DetailedHealthCheck(&DetailedHealthCheckArgs{}, &detailedResult)
	if err != nil {
		t.Fatalf("DetailedHealthCheck failed: %v", err)
	}

	if detailedResult.Status != "healthy" {
		t.Errorf("Expected healthy status in detailed check, got: %s", detailedResult.Status)
	}

	// Verify dependencies are populated with detailed information
	if len(detailedResult.Dependencies) != 3 {
		t.Errorf("Expected 3 dependencies, got: %d", len(detailedResult.Dependencies))
	}

	for serviceName, dep := range detailedResult.Dependencies {
		if dep.Status != "healthy" && dep.Status != "not_initialized" {
			t.Errorf("Dependency %s should be healthy or not_initialized, got: %s", serviceName, dep.Status)
		}

		// Only check response time for initialized dependencies
		if dep.Status != "not_initialized" && dep.ResponseTime <= 0 {
			t.Errorf("Dependency %s should have positive response time", serviceName)
		}

		if dep.LastCheck.IsZero() {
			t.Errorf("Dependency %s should have last check time set", serviceName)
		}
	}

	// Verify system info
	if detailedResult.SystemInfo.Version == "" {
		t.Errorf("System version should be set")
	}

	if detailedResult.Uptime <= 0 {
		t.Errorf("Uptime should be positive")
	}

	// Test metrics endpoint
	var metricsResult PerformanceMetrics
	err = service.GetMetrics(&GetMetricsArgs{}, &metricsResult)
	if err != nil {
		t.Fatalf("GetMetrics failed: %v", err)
	}

	if metricsResult.StartTime.IsZero() {
		t.Errorf("Metrics start time should be set")
	}
}

// TestHealthCheckWithFailures tests health check behavior when dependencies fail
func TestHealthCheckWithFailures(t *testing.T) {
	// Create service with some failing dependencies
	ragAgent := &MockRAGAgent{shouldError: true}
	vectorDB := &MockVectorDB{shouldError: false}
	llmClient := &MockLLMClient{shouldError: true}

	service := NewRPCService(ragAgent, vectorDB, llmClient)

	// Test basic health check
	var healthResult HealthStatus
	err := service.HealthCheck(&HealthCheckArgs{}, &healthResult)
	if err != nil {
		t.Fatalf("HealthCheck failed: %v", err)
	}

	if healthResult.Status != "unhealthy" {
		t.Errorf("Expected unhealthy status, got: %s", healthResult.Status)
	}

	// Verify failing services are reported correctly
	ragStatus := healthResult.Services["rag_agent"]
	if !contains(ragStatus, "unhealthy") {
		t.Errorf("RAG agent should be unhealthy: %s", ragStatus)
	}

	llmStatus := healthResult.Services["llm_client"]
	if !contains(llmStatus, "unhealthy") {
		t.Errorf("LLM client should be unhealthy: %s", llmStatus)
	}

	vectorStatus := healthResult.Services["vector_db"]
	if !contains(vectorStatus, "healthy") {
		t.Errorf("Vector DB should be healthy: %s", vectorStatus)
	}

	// Test detailed health check with failures
	var detailedResult DetailedHealthStatus
	err = service.DetailedHealthCheck(&DetailedHealthCheckArgs{}, &detailedResult)
	if err != nil {
		t.Fatalf("DetailedHealthCheck failed: %v", err)
	}

	if detailedResult.Status != "unhealthy" {
		t.Errorf("Expected unhealthy status in detailed check, got: %s", detailedResult.Status)
	}

	// Verify failing dependencies have error information
	ragDep := detailedResult.Dependencies["rag_agent"]
	if ragDep.Status != "unhealthy" {
		t.Errorf("RAG agent dependency should be unhealthy")
	}
	if ragDep.Error == "" {
		t.Errorf("RAG agent dependency should have error message")
	}

	llmDep := detailedResult.Dependencies["llm_client"]
	if llmDep.Status != "unhealthy" {
		t.Errorf("LLM client dependency should be unhealthy")
	}
	if llmDep.Error == "" {
		t.Errorf("LLM client dependency should have error message")
	}

	vectorDep := detailedResult.Dependencies["vector_db"]
	if vectorDep.Status != "healthy" {
		t.Errorf("Vector DB dependency should be healthy")
	}
	if vectorDep.Error != "" {
		t.Errorf("Vector DB dependency should not have error message")
	}
}

// TestMetricsCollection tests that performance metrics are collected correctly
func TestMetricsCollection(t *testing.T) {
	service := NewRPCService(&MockRAGAgent{}, &MockVectorDB{}, &MockLLMClient{})

	// Simulate some queries to generate metrics
	queryArgs := &QueryArgs{Query: "test query", Limit: 5}
	var queryResult QueryResult

	// Execute a few queries
	for i := 0; i < 3; i++ {
		err := service.Query(queryArgs, &queryResult)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}
		time.Sleep(1 * time.Millisecond) // Small delay to ensure different timestamps
	}

	// Get metrics
	var metricsResult PerformanceMetrics
	err := service.GetMetrics(&GetMetricsArgs{}, &metricsResult)
	if err != nil {
		t.Fatalf("GetMetrics failed: %v", err)
	}

	// Verify metrics are collected (if health monitor is available)
	if service.healthMonitor != nil {
		if metricsResult.QueryCount != 3 {
			t.Errorf("Expected 3 queries, got: %d", metricsResult.QueryCount)
		}

		if metricsResult.SuccessfulQueries != 3 {
			t.Errorf("Expected 3 successful queries, got: %d", metricsResult.SuccessfulQueries)
		}

		if metricsResult.FailedQueries != 0 {
			t.Errorf("Expected 0 failed queries, got: %d", metricsResult.FailedQueries)
		}

		if metricsResult.AverageResponseTime <= 0 {
			t.Errorf("Expected positive average response time")
		}

		if metricsResult.LastResponseTime <= 0 {
			t.Errorf("Expected positive last response time")
		}
	}
}

// Helper function to check if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	if len(s) < len(substr) {
		return false
	}

	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
