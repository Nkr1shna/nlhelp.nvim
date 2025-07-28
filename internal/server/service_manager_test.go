package server

import (
	"testing"
	"time"
)

func TestServiceManager_StartStop(t *testing.T) {
	config := &ServiceManagerConfig{
		MaxRestarts:         3,
		HealthCheckInterval: 100 * time.Millisecond,
		RestartDelay:        50 * time.Millisecond,
	}

	sm := NewServiceManager(&MockRAGAgent{}, &MockVectorDB{}, &MockLLMClient{}, config)

	// Test start
	err := sm.Start()
	if err != nil {
		t.Errorf("unexpected error starting service: %v", err)
	}

	if !sm.IsRunning() {
		t.Errorf("expected service to be running")
	}

	// Test double start
	err = sm.Start()
	if err == nil {
		t.Errorf("expected error when starting already running service")
	}

	// Test stop
	err = sm.Stop()
	if err != nil {
		t.Errorf("unexpected error stopping service: %v", err)
	}

	if sm.IsRunning() {
		t.Errorf("expected service to be stopped")
	}

	// Test double stop
	err = sm.Stop()
	if err != nil {
		t.Errorf("unexpected error stopping already stopped service: %v", err)
	}
}

func TestServiceManager_GetRPCService(t *testing.T) {
	sm := NewServiceManager(&MockRAGAgent{}, &MockVectorDB{}, &MockLLMClient{}, nil)

	// Before start
	rpcService := sm.GetRPCService()
	if rpcService != nil {
		t.Errorf("expected nil RPC service before start")
	}

	// After start
	err := sm.Start()
	if err != nil {
		t.Errorf("unexpected error starting service: %v", err)
	}
	defer sm.Stop()

	rpcService = sm.GetRPCService()
	if rpcService == nil {
		t.Errorf("expected non-nil RPC service after start")
	}
}

func TestServiceManager_HealthMonitoring(t *testing.T) {
	config := &ServiceManagerConfig{
		MaxRestarts:         1,
		HealthCheckInterval: 50 * time.Millisecond,
		RestartDelay:        10 * time.Millisecond,
	}

	// Create mocks that will fail health checks
	ragAgent := &MockRAGAgent{shouldError: true}
	vectorDB := &MockVectorDB{shouldError: false}
	llmClient := &MockLLMClient{shouldError: false}

	sm := NewServiceManager(ragAgent, vectorDB, llmClient, config)

	err := sm.Start()
	if err != nil {
		t.Errorf("unexpected error starting service: %v", err)
	}
	defer sm.Stop()

	// Wait for health check to run
	time.Sleep(100 * time.Millisecond)

	status := sm.GetHealthStatus()
	if status["rag_agent"] {
		t.Errorf("expected rag_agent to be unhealthy")
	}

	if !status["vector_db"] {
		t.Errorf("expected vector_db to be healthy")
	}
}

// MockRAGAgentWithInitError implements RAGAgent with initialization error
type MockRAGAgentWithInitError struct {
	MockRAGAgent
}

func (m *MockRAGAgentWithInitError) Initialize() error {
	return NewRPCError(ErrorCodeInitializationError, "initialization failed")
}

func TestServiceManager_InitializationFailure(t *testing.T) {
	// Create a mock that fails initialization
	ragAgent := &MockRAGAgentWithInitError{}

	sm := NewServiceManager(ragAgent, &MockVectorDB{}, &MockLLMClient{}, nil)

	err := sm.Start()
	if err == nil {
		t.Errorf("expected error when initialization fails")
	}

	if sm.IsRunning() {
		t.Errorf("expected service not to be running after initialization failure")
	}
}

func TestRecoverFromPanic(t *testing.T) {
	// Test normal execution (no panic)
	err := RecoverFromPanic()
	if err != nil {
		t.Errorf("expected no error when no panic occurs, got: %v", err)
	}

	// Note: Testing actual panic recovery is complex in unit tests
	// The panic recovery is tested implicitly through the RPC method tests
	// where the defer statements in the RPC methods will catch any panics
}
