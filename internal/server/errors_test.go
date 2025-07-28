package server

import (
	"errors"
	"testing"
)

func TestNewRPCError(t *testing.T) {
	tests := []struct {
		name     string
		code     ErrorCode
		message  string
		details  []string
		expected string
	}{
		{
			name:     "error without details",
			code:     ErrorCodeInvalidQuery,
			message:  "invalid query",
			expected: "RPC Error 4001: invalid query",
		},
		{
			name:     "error with details",
			code:     ErrorCodeInternalError,
			message:  "internal error",
			details:  []string{"database connection failed"},
			expected: "RPC Error 5000: internal error (database connection failed)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewRPCError(tt.code, tt.message, tt.details...)
			if err.Error() != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, err.Error())
			}
		})
	}
}

func TestWrapError(t *testing.T) {
	originalErr := errors.New("original error")
	rpcErr := WrapError(originalErr, ErrorCodeVectorDBError, "vector db failed")

	if rpcErr.Code != ErrorCodeVectorDBError {
		t.Errorf("expected code %d, got %d", ErrorCodeVectorDBError, rpcErr.Code)
	}

	if rpcErr.Message != "vector db failed" {
		t.Errorf("expected message 'vector db failed', got '%s'", rpcErr.Message)
	}

	if rpcErr.Details != "original error" {
		t.Errorf("expected details 'original error', got '%s'", rpcErr.Details)
	}
}

func TestIsClientError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "client error",
			err:      NewRPCError(ErrorCodeInvalidQuery, "invalid query"),
			expected: true,
		},
		{
			name:     "server error",
			err:      NewRPCError(ErrorCodeInternalError, "internal error"),
			expected: false,
		},
		{
			name:     "regular error",
			err:      errors.New("regular error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsClientError(tt.err)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestIsServerError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "server error",
			err:      NewRPCError(ErrorCodeInternalError, "internal error"),
			expected: true,
		},
		{
			name:     "client error",
			err:      NewRPCError(ErrorCodeInvalidQuery, "invalid query"),
			expected: false,
		},
		{
			name:     "regular error",
			err:      errors.New("regular error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsServerError(tt.err)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
