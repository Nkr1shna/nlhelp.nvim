package server

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"nvim-smart-keybind-search/internal/interfaces"
)

// ServiceManager manages the lifecycle of the RPC service and its dependencies
type ServiceManager struct {
	rpcService *RPCService
	ragAgent   interfaces.RAGAgent
	vectorDB   interfaces.VectorDB
	llmClient  interfaces.LLMClient

	// Service state
	mu           sync.RWMutex
	isRunning    bool
	restartCount int
	maxRestarts  int

	// Context for graceful shutdown
	ctx    context.Context
	cancel context.CancelFunc

	// Health monitoring
	healthCheckInterval time.Duration
	lastHealthCheck     time.Time
	healthStatus        map[string]bool
}

// ServiceManagerConfig holds configuration for the service manager
type ServiceManagerConfig struct {
	MaxRestarts         int
	HealthCheckInterval time.Duration
	RestartDelay        time.Duration
}

// DefaultServiceManagerConfig returns default configuration
func DefaultServiceManagerConfig() *ServiceManagerConfig {
	return &ServiceManagerConfig{
		MaxRestarts:         5,
		HealthCheckInterval: 30 * time.Second,
		RestartDelay:        5 * time.Second,
	}
}

// NewServiceManager creates a new service manager
func NewServiceManager(ragAgent interfaces.RAGAgent, vectorDB interfaces.VectorDB, llmClient interfaces.LLMClient, config *ServiceManagerConfig) *ServiceManager {
	if config == nil {
		config = DefaultServiceManagerConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &ServiceManager{
		ragAgent:            ragAgent,
		vectorDB:            vectorDB,
		llmClient:           llmClient,
		maxRestarts:         config.MaxRestarts,
		healthCheckInterval: config.HealthCheckInterval,
		ctx:                 ctx,
		cancel:              cancel,
		healthStatus:        make(map[string]bool),
	}
}

// Start initializes and starts the service
func (sm *ServiceManager) Start() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.isRunning {
		return fmt.Errorf("service is already running")
	}

	// Initialize dependencies
	if err := sm.initializeDependencies(); err != nil {
		return fmt.Errorf("failed to initialize dependencies: %w", err)
	}

	// Create RPC service
	sm.rpcService = NewRPCService(sm.ragAgent, sm.vectorDB, sm.llmClient)

	sm.isRunning = true
	sm.restartCount = 0

	// Start health monitoring
	go sm.healthMonitor()

	log.Println("Service manager started successfully")
	return nil
}

// Stop gracefully shuts down the service
func (sm *ServiceManager) Stop() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !sm.isRunning {
		return nil
	}

	// Cancel context to stop background goroutines
	sm.cancel()

	// Close dependencies
	if sm.vectorDB != nil {
		if err := sm.vectorDB.Close(); err != nil {
			log.Printf("Error closing vector database: %v", err)
		}
	}

	if sm.llmClient != nil {
		if err := sm.llmClient.Close(); err != nil {
			log.Printf("Error closing LLM client: %v", err)
		}
	}

	sm.isRunning = false
	log.Println("Service manager stopped")
	return nil
}

// GetRPCService returns the RPC service instance
func (sm *ServiceManager) GetRPCService() *RPCService {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.rpcService
}

// IsRunning returns whether the service is currently running
func (sm *ServiceManager) IsRunning() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.isRunning
}

// GetHealthStatus returns the current health status of all components
func (sm *ServiceManager) GetHealthStatus() map[string]bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	status := make(map[string]bool)
	for k, v := range sm.healthStatus {
		status[k] = v
	}
	return status
}

// initializeDependencies initializes all service dependencies
func (sm *ServiceManager) initializeDependencies() error {
	// Initialize RAG Agent
	if sm.ragAgent != nil {
		if err := sm.ragAgent.Initialize(); err != nil {
			return WrapError(err, ErrorCodeInitializationError, "failed to initialize RAG agent")
		}
	}

	// Initialize Vector Database
	if sm.vectorDB != nil {
		if err := sm.vectorDB.Initialize(); err != nil {
			return WrapError(err, ErrorCodeInitializationError, "failed to initialize vector database")
		}
	}

	// Initialize LLM Client
	if sm.llmClient != nil {
		if err := sm.llmClient.Initialize(); err != nil {
			return WrapError(err, ErrorCodeInitializationError, "failed to initialize LLM client")
		}
	}

	return nil
}

// healthMonitor runs periodic health checks and handles service recovery
func (sm *ServiceManager) healthMonitor() {
	ticker := time.NewTicker(sm.healthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-sm.ctx.Done():
			return
		case <-ticker.C:
			sm.performHealthCheck()
		}
	}
}

// performHealthCheck checks the health of all components
func (sm *ServiceManager) performHealthCheck() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !sm.isRunning {
		return
	}

	sm.lastHealthCheck = time.Now()
	allHealthy := true

	// Check RAG Agent
	if sm.ragAgent != nil {
		if err := sm.ragAgent.HealthCheck(); err != nil {
			sm.healthStatus["rag_agent"] = false
			allHealthy = false
			log.Printf("RAG agent health check failed: %v", err)
		} else {
			sm.healthStatus["rag_agent"] = true
		}
	}

	// Check Vector Database
	if sm.vectorDB != nil {
		if err := sm.vectorDB.HealthCheck(); err != nil {
			sm.healthStatus["vector_db"] = false
			allHealthy = false
			log.Printf("Vector database health check failed: %v", err)
		} else {
			sm.healthStatus["vector_db"] = true
		}
	}

	// Check LLM Client
	if sm.llmClient != nil {
		if err := sm.llmClient.HealthCheck(); err != nil {
			sm.healthStatus["llm_client"] = false
			allHealthy = false
			log.Printf("LLM client health check failed: %v", err)
		} else {
			sm.healthStatus["llm_client"] = true
		}
	}

	// If any component is unhealthy, attempt restart
	if !allHealthy {
		sm.attemptRestart()
	}
}

// attemptRestart attempts to restart failed components
func (sm *ServiceManager) attemptRestart() {
	if sm.restartCount >= sm.maxRestarts {
		log.Printf("Maximum restart attempts (%d) reached, service will remain degraded", sm.maxRestarts)
		return
	}

	sm.restartCount++
	log.Printf("Attempting service restart (%d/%d)", sm.restartCount, sm.maxRestarts)

	// Attempt to reinitialize failed components
	go func() {
		// Wait before restart attempt
		time.Sleep(5 * time.Second)

		sm.mu.Lock()
		defer sm.mu.Unlock()

		// Try to reinitialize dependencies
		if err := sm.initializeDependencies(); err != nil {
			log.Printf("Failed to reinitialize dependencies during restart: %v", err)
			return
		}

		// Recreate RPC service with reinitialized dependencies
		sm.rpcService = NewRPCService(sm.ragAgent, sm.vectorDB, sm.llmClient)

		log.Printf("Service restart attempt %d completed", sm.restartCount)
	}()
}

// RecoverFromPanic recovers from panics in RPC methods and returns appropriate errors
func RecoverFromPanic() error {
	if r := recover(); r != nil {
		log.Printf("Recovered from panic in RPC method: %v", r)
		return NewRPCError(ErrorCodeInternalError, "internal server error", fmt.Sprintf("panic: %v", r))
	}
	return nil
}
