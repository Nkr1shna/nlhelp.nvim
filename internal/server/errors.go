package server

import (
	"fmt"
	"log"
)

// ErrorCode represents standardized error codes for RPC responses
type ErrorCode int

const (
	// Client errors (4xx equivalent)
	ErrorCodeInvalidRequest ErrorCode = 4000
	ErrorCodeInvalidQuery   ErrorCode = 4001
	ErrorCodeQueryTooLong   ErrorCode = 4002
	ErrorCodeRateLimited    ErrorCode = 4003

	// Server errors (5xx equivalent)
	ErrorCodeInternalError       ErrorCode = 5000
	ErrorCodeServiceUnavailable  ErrorCode = 5001
	ErrorCodeRAGAgentError       ErrorCode = 5002
	ErrorCodeVectorDBError       ErrorCode = 5003
	ErrorCodeLLMError            ErrorCode = 5004
	ErrorCodeInitializationError ErrorCode = 5005
)

// RPCError represents a structured error response
type RPCError struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Details string    `json:"details,omitempty"`
}

// Error implements the error interface
func (e *RPCError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("RPC Error %d: %s (%s)", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("RPC Error %d: %s", e.Code, e.Message)
}

// NewRPCError creates a new structured RPC error
func NewRPCError(code ErrorCode, message string, details ...string) *RPCError {
	err := &RPCError{
		Code:    code,
		Message: message,
	}
	if len(details) > 0 {
		err.Details = details[0]
	}
	return err
}

// WrapError wraps a regular error into an RPC error with appropriate code
func WrapError(err error, code ErrorCode, message string) *RPCError {
	return &RPCError{
		Code:    code,
		Message: message,
		Details: err.Error(),
	}
}

// LogError logs an error with appropriate level based on error code
func LogError(err error, context string) {
	if rpcErr, ok := err.(*RPCError); ok {
		if rpcErr.Code >= 5000 {
			log.Printf("ERROR [%s]: %v", context, err)
		} else {
			log.Printf("WARN [%s]: %v", context, err)
		}
	} else {
		log.Printf("ERROR [%s]: %v", context, err)
	}
}

// IsClientError returns true if the error is a client-side error
func IsClientError(err error) bool {
	if rpcErr, ok := err.(*RPCError); ok {
		return rpcErr.Code >= 4000 && rpcErr.Code < 5000
	}
	return false
}

// IsServerError returns true if the error is a server-side error
func IsServerError(err error) bool {
	if rpcErr, ok := err.(*RPCError); ok {
		return rpcErr.Code >= 5000
	}
	return false
}
