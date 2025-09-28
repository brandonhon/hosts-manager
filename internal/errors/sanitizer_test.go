package errors

import (
	"errors"
	"strings"
	"testing"
)

func TestNewSanitizedError(t *testing.T) {
	tests := []struct {
		name         string
		internalErr  error
		userMessage  string
		expectedUser string
	}{
		{
			name:         "basic sanitized error",
			internalErr:  errors.New("internal error with sensitive data"),
			userMessage:  "operation failed",
			expectedUser: "operation failed",
		},
		{
			name:         "empty user message",
			internalErr:  errors.New("internal error"),
			userMessage:  "",
			expectedUser: "",
		},
		{
			name:         "nil internal error",
			internalErr:  nil,
			userMessage:  "user message",
			expectedUser: "user message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sanitizedErr := NewSanitizedError(tt.internalErr, tt.userMessage)

			if sanitizedErr.UserMessage != tt.expectedUser {
				t.Errorf("NewSanitizedError().UserMessage = %q, want %q", sanitizedErr.UserMessage, tt.expectedUser)
			}

			if sanitizedErr.InternalError != tt.internalErr {
				t.Errorf("NewSanitizedError().InternalError = %v, want %v", sanitizedErr.InternalError, tt.internalErr)
			}

			// Test Error() method
			errStr := sanitizedErr.Error()
			if errStr != tt.expectedUser {
				t.Errorf("SanitizedError.Error() = %q, want %q", errStr, tt.expectedUser)
			}

			// Test Unwrap() method
			unwrapped := sanitizedErr.Unwrap()
			if unwrapped != tt.internalErr {
				t.Errorf("SanitizedError.Unwrap() = %v, want %v", unwrapped, tt.internalErr)
			}
		})
	}
}

func TestSanitizeError(t *testing.T) {
	tests := []struct {
		name     string
		input    error
		validate func(error) bool
	}{
		{
			name:  "error with Users path gets sanitized",
			input: errors.New("failed to open file /Users/john/secret/config.txt"),
			validate: func(output error) bool {
				if output == nil {
					return false
				}
				errStr := output.Error()
				return strings.Contains(errStr, "[user]") && !strings.Contains(errStr, "john")
			},
		},
		{
			name:  "error with home path gets sanitized",
			input: errors.New("failed to read /home/user/secret/file.txt"),
			validate: func(output error) bool {
				if output == nil {
					return false
				}
				errStr := output.Error()
				return strings.Contains(errStr, "[user]") && !strings.Contains(errStr, "/home/user/")
			},
		},
		{
			name:  "process ID gets sanitized",
			input: errors.New("process 1234 crashed"),
			validate: func(output error) bool {
				if output == nil {
					return false
				}
				errStr := output.Error()
				return !strings.Contains(errStr, "1234") && strings.Contains(errStr, "[hidden]")
			},
		},
		{
			name:  "clean error without sensitive data unchanged",
			input: errors.New("invalid input format"),
			validate: func(output error) bool {
				if output == nil {
					return false
				}
				return output.Error() == "invalid input format"
			},
		},
		{
			name:  "nil error returns nil",
			input: nil,
			validate: func(output error) bool {
				return output == nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeError(tt.input)

			if !tt.validate(result) {
				if result != nil {
					t.Errorf("SanitizeError(%v) = %q, failed validation", tt.input, result.Error())
				} else {
					t.Errorf("SanitizeError(%v) = nil, failed validation", tt.input)
				}
			}
		})
	}
}

func TestSanitizeErrorMessage(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		validate func(string) bool
	}{
		{
			name:  "message with Users path",
			input: "failed to access /Users/john/secret/config.txt",
			validate: func(output string) bool {
				return strings.Contains(output, "[user]") && !strings.Contains(output, "john")
			},
		},
		{
			name:  "message with home path",
			input: "error reading /home/user/file.txt",
			validate: func(output string) bool {
				return strings.Contains(output, "[user]")
			},
		},
		{
			name:  "message with process info",
			input: "process 1234 crashed",
			validate: func(output string) bool {
				return !strings.Contains(output, "1234") && strings.Contains(output, "[hidden]")
			},
		},
		{
			name:  "clean message unchanged",
			input: "generic error occurred",
			validate: func(output string) bool {
				return output == "generic error occurred"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeErrorMessage(tt.input)

			if !tt.validate(result) {
				t.Errorf("SanitizeErrorMessage(%q) = %q, failed validation", tt.input, result)
			}
		})
	}
}

func TestIsSecuritySensitive(t *testing.T) {
	tests := []struct {
		name     string
		input    error
		expected bool
	}{
		{
			name:     "permission denied error",
			input:    errors.New("permission denied"),
			expected: true,
		},
		{
			name:     "access denied error",
			input:    errors.New("access denied to resource"),
			expected: true,
		},
		{
			name:     "authentication failed error",
			input:    errors.New("authentication failed"),
			expected: true,
		},
		{
			name:     "unauthorized access error",
			input:    errors.New("unauthorized access"),
			expected: true,
		},
		{
			name:     "forbidden error",
			input:    errors.New("forbidden resource"),
			expected: true,
		},
		{
			name:     "regular error",
			input:    errors.New("file not found"),
			expected: false,
		},
		{
			name:     "nil error",
			input:    nil,
			expected: false,
		},
		{
			name:     "case insensitive match",
			input:    errors.New("Permission DENIED"),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsSecuritySensitive(tt.input)

			if result != tt.expected {
				t.Errorf("IsSecuritySensitive(%v) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizeFilePaths(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		output string
	}{
		{
			name:   "simple message unchanged",
			input:  "simple error message",
			output: "simple error message",
		},
		{
			name:   "short paths unchanged",
			input:  "error in /home/user/file.txt",
			output: "error in /home/user/file.txt",
		},
		{
			name:   "path with dots gets sanitized",
			input:  "failed to read /app.config/data/file.txt",
			output: "failed to read /[path]/data/file.txt",
		},
		{
			name:   "long path parts get sanitized",
			input:  "error in /verylongdirectorynamethatexceedstwentycharacters/file.txt",
			output: "error in /[path]/file.txt",
		},
		{
			name:   "Windows path with long parts",
			input:  "failed to access C:\\VeryLongDirectoryNameThatExceedsTwentyCharacters\\file.txt",
			output: "failed to access C:\\[path]\\file.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeFilePaths(tt.input)

			if result != tt.output {
				t.Errorf("sanitizeFilePaths(%q) = %q, want %q", tt.input, result, tt.output)
			}
		})
	}
}

func TestSanitizeUserInfo(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		validate func(string) bool
	}{
		{
			name:  "process ID with pid keyword",
			input: "pid 1234 failed",
			validate: func(output string) bool {
				return !strings.Contains(output, "1234") && strings.Contains(output, "[hidden]")
			},
		},
		{
			name:  "process ID with process keyword",
			input: "process 5678 crashed",
			validate: func(output string) bool {
				return !strings.Contains(output, "5678") && strings.Contains(output, "[hidden]")
			},
		},
		{
			name:  "no process info unchanged",
			input: "generic error occurred",
			validate: func(output string) bool {
				return output == "generic error occurred"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeUserInfo(tt.input)

			if !tt.validate(result) {
				t.Errorf("sanitizeUserInfo(%q) = %q, failed validation", tt.input, result)
			}
		})
	}
}

func TestSanitizeCommonPatterns(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "bind address already in use",
			input:    "bind: address already in use",
			expected: "network resource busy",
		},
		{
			name:     "connection refused",
			input:    "connection refused",
			expected: "connection failed",
		},
		{
			name:     "network unreachable",
			input:    "network is unreachable",
			expected: "network error",
		},
		{
			name:     "operation timed out",
			input:    "operation timed out",
			expected: "operation timeout",
		},
		{
			name:     "broken pipe",
			input:    "broken pipe",
			expected: "connection interrupted",
		},
		{
			name:     "case insensitive match",
			input:    "CONNECTION REFUSED",
			expected: "connection failed",
		},
		{
			name:     "no pattern match unchanged",
			input:    "custom error message",
			expected: "custom error message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeCommonPatterns(tt.input)

			if result != tt.expected {
				t.Errorf("sanitizeCommonPatterns(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestWrapWithSanitization(t *testing.T) {
	tests := []struct {
		name     string
		fn       func() error
		validate func(error) bool
	}{
		{
			name: "function returns sanitizable error",
			fn: func() error {
				return errors.New("failed to read /Users/john/secret.txt")
			},
			validate: func(output error) bool {
				if output == nil {
					return false
				}
				return strings.Contains(output.Error(), "[user]")
			},
		},
		{
			name: "function returns nil",
			fn: func() error {
				return nil
			},
			validate: func(output error) bool {
				return output == nil
			},
		},
		{
			name: "function returns clean error",
			fn: func() error {
				return errors.New("clean error")
			},
			validate: func(output error) bool {
				if output == nil {
					return false
				}
				return output.Error() == "clean error"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WrapWithSanitization(tt.fn)

			if !tt.validate(result) {
				t.Errorf("WrapWithSanitization failed validation, got: %v", result)
			}
		})
	}
}

func TestFormatUserError(t *testing.T) {
	tests := []struct {
		name      string
		operation string
		err       error
		validate  func(error) bool
	}{
		{
			name:      "format error with sanitization",
			operation: "file operation",
			err:       errors.New("failed to read /Users/john/secret.txt"),
			validate: func(output error) bool {
				if output == nil {
					return false
				}
				errStr := output.Error()
				return strings.Contains(errStr, "file operation failed:") && strings.Contains(errStr, "[user]")
			},
		},
		{
			name:      "nil error returns nil",
			operation: "operation",
			err:       nil,
			validate: func(output error) bool {
				return output == nil
			},
		},
		{
			name:      "clean error gets formatted",
			operation: "test",
			err:       errors.New("simple error"),
			validate: func(output error) bool {
				if output == nil {
					return false
				}
				return output.Error() == "test failed: simple error"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatUserError(tt.operation, tt.err)

			if !tt.validate(result) {
				t.Errorf("FormatUserError failed validation, got: %v", result)
			}
		})
	}
}

// Benchmark tests
func BenchmarkSanitizeError(b *testing.B) {
	testErr := errors.New("failed to open file /Users/user/secret/config.txt")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = SanitizeError(testErr)
	}
}

func BenchmarkSanitizeErrorMessage(b *testing.B) {
	testMsg := "failed to open file /Users/user/secret/config.txt"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SanitizeErrorMessage(testMsg)
	}
}

func BenchmarkIsSecuritySensitive(b *testing.B) {
	testErr := errors.New("permission denied for user access")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsSecuritySensitive(testErr)
	}
}

// Edge case tests
func TestSanitizeErrorEdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		input error
	}{
		{
			name:  "very long error message",
			input: errors.New(strings.Repeat("a", 1000) + " /Users/user/secret " + strings.Repeat("b", 1000)),
		},
		{
			name:  "empty error message",
			input: errors.New(""),
		},
		{
			name:  "error with multiple paths",
			input: errors.New("copy from /Users/john/src to /Users/jane/dst failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeError(tt.input)

			// Test that function doesn't crash and returns something reasonable
			if result == nil && tt.input != nil {
				t.Errorf("SanitizeError should not return nil for non-nil input")
			}
		})
	}
}

// Test concurrent access
func TestSanitizeErrorConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrency test in short mode")
	}

	testErr := errors.New("failed to access /Users/user/secret/file.txt")

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			for j := 0; j < 100; j++ {
				result := SanitizeError(testErr)
				if result == nil {
					t.Errorf("SanitizeError returned nil unexpectedly")
					return
				}
			}
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}
