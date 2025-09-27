package errors

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// SanitizedError represents an error with both internal detail and sanitized user message
type SanitizedError struct {
	InternalError error
	UserMessage   string
}

func (e SanitizedError) Error() string {
	return e.UserMessage
}

func (e SanitizedError) Unwrap() error {
	return e.InternalError
}

// NewSanitizedError creates a new sanitized error
func NewSanitizedError(internal error, userMsg string) *SanitizedError {
	return &SanitizedError{
		InternalError: internal,
		UserMessage:   userMsg,
	}
}

// SanitizeError sanitizes an error message for user display
func SanitizeError(err error) error {
	if err == nil {
		return nil
	}

	errMsg := err.Error()
	sanitized := SanitizeErrorMessage(errMsg)
	
	if sanitized != errMsg {
		return NewSanitizedError(err, sanitized)
	}
	
	return err
}

// SanitizeErrorMessage sanitizes a single error message string
func SanitizeErrorMessage(message string) string {
	// Remove full file paths, keep only filenames
	message = sanitizeFilePaths(message)
	
	// Remove potential sensitive environment information
	message = sanitizeEnvironmentInfo(message)
	
	// Remove potential user information
	message = sanitizeUserInfo(message)
	
	// Sanitize common error patterns that might leak information
	message = sanitizeCommonPatterns(message)
	
	return message
}

// safeRegexReplace performs regex replacement with timeout protection
func safeRegexReplace(pattern *regexp.Regexp, input, replacement string, timeout time.Duration) string {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	
	resultCh := make(chan string, 1)
	go func() {
		resultCh <- pattern.ReplaceAllString(input, replacement)
	}()
	
	select {
	case result := <-resultCh:
		return result
	case <-ctx.Done():
		// Timeout occurred, return input unchanged to be safe
		return input
	}
}

// sanitizeFilePaths removes full paths and keeps only relevant filenames using simple string operations
func sanitizeFilePaths(message string) string {
	// Use simple string replacements instead of complex regex to avoid DoS
	// Replace common path patterns with simplified versions
	
	// Unix-style paths
	if strings.Contains(message, "/") {
		parts := strings.Split(message, "/")
		if len(parts) > 2 {
			// Keep only the last part (filename) for paths
			for i := 0; i < len(parts)-1; i++ {
				if strings.Contains(parts[i], ".") || len(parts[i]) > 20 {
					parts[i] = "[path]"
				}
			}
			message = strings.Join(parts, "/")
		}
	}
	
	// Windows-style paths
	if strings.Contains(message, "\\") {
		parts := strings.Split(message, "\\")
		if len(parts) > 2 {
			// Keep only the last part (filename) for paths
			for i := 0; i < len(parts)-1; i++ {
				if strings.Contains(parts[i], ".") || len(parts[i]) > 20 {
					parts[i] = "[path]"
				}
			}
			message = strings.Join(parts, "\\")
		}
	}
	
	return message
}

// sanitizeEnvironmentInfo removes environment-specific information
func sanitizeEnvironmentInfo(message string) string {
	patterns := []struct {
		pattern     *regexp.Regexp
		replacement string
	}{
		{regexp.MustCompile(`(?i)permission denied.*`), "insufficient permissions"},
		{regexp.MustCompile(`(?i)no such file or directory.*`), "file not found"},
		{regexp.MustCompile(`(?i)access is denied.*`), "access denied"},
		{regexp.MustCompile(`(?i)cannot access.*`), "file access error"},
		{regexp.MustCompile(`(?i)operation not permitted.*`), "operation not allowed"},
	}
	
	for _, p := range patterns {
		message = p.pattern.ReplaceAllString(message, p.replacement)
	}
	
	return message
}

// sanitizeUserInfo removes potentially sensitive user information using safe string operations
func sanitizeUserInfo(message string) string {
	// Use simple string replacements instead of regex to avoid DoS attacks
	
	// Replace common user path patterns
	replacements := map[string]string{
		"/Users/":     "/Users/[user]/",
		"/home/":      "/home/[user]/",
		"C:\\Users\\": "C:\\Users\\[user]\\",
	}
	
	for pattern, replacement := range replacements {
		if strings.Contains(message, pattern) {
			parts := strings.Split(message, pattern)
			if len(parts) > 1 {
				// Replace the username part after the path
				for i := 1; i < len(parts); i++ {
					userPart := strings.Split(parts[i], "/")[0]
					userPart = strings.Split(userPart, "\\")[0]
					userPart = strings.Split(userPart, " ")[0]
					parts[i] = strings.Replace(parts[i], userPart, "[user]", 1)
				}
				message = strings.Join(parts, pattern[:len(pattern)-1])
			}
		}
	}
	
	// Simple PID sanitization using string operations
	if strings.Contains(message, "pid ") {
		words := strings.Fields(message)
		for i, word := range words {
			if i > 0 && words[i-1] == "pid" && isNumeric(word) {
				words[i] = "[hidden]"
			}
		}
		message = strings.Join(words, " ")
	}
	
	// Simple process ID sanitization
	if strings.Contains(message, "process ") {
		words := strings.Fields(message)
		for i, word := range words {
			if i > 0 && words[i-1] == "process" && isNumeric(word) {
				words[i] = "[hidden]"
			}
		}
		message = strings.Join(words, " ")
	}
	
	// Simple IP address sanitization (basic pattern)
	words := strings.Fields(message)
	for i, word := range words {
		if isSimpleIPv4(word) {
			words[i] = "[ip-address]"
		}
	}
	message = strings.Join(words, " ")
	
	return message
}

// isNumeric checks if a string contains only digits
func isNumeric(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// isSimpleIPv4 performs a basic IPv4 check without regex
func isSimpleIPv4(s string) bool {
	parts := strings.Split(s, ".")
	if len(parts) != 4 {
		return false
	}
	for _, part := range parts {
		if !isNumeric(part) || len(part) == 0 || len(part) > 3 {
			return false
		}
		// Basic range check (0-255)
		if len(part) == 3 && part[0] > '2' {
			return false
		}
		if len(part) == 3 && part[0] == '2' && part[1] > '5' {
			return false
		}
		if len(part) == 3 && part[0] == '2' && part[1] == '5' && part[2] > '5' {
			return false
		}
	}
	return true
}

// sanitizeCommonPatterns sanitizes common error patterns
func sanitizeCommonPatterns(message string) string {
	// Replace specific error types with generic messages
	replacements := map[string]string{
		"bind: address already in use":     "network resource busy",
		"connection refused":               "connection failed",
		"network is unreachable":          "network error",
		"device or resource busy":         "resource unavailable",
		"operation timed out":             "operation timeout",
		"broken pipe":                     "connection interrupted",
	}
	
	lowerMsg := strings.ToLower(message)
	for pattern, replacement := range replacements {
		if strings.Contains(lowerMsg, pattern) {
			return replacement
		}
	}
	
	return message
}

// WrapWithSanitization wraps a function to automatically sanitize errors
func WrapWithSanitization(fn func() error) error {
	err := fn()
	return SanitizeError(err)
}

// FormatUserError formats an error for safe user display
func FormatUserError(operation string, err error) error {
	if err == nil {
		return nil
	}
	
	sanitized := SanitizeError(err)
	return fmt.Errorf("%s failed: %v", operation, sanitized)
}

// IsSecuritySensitive checks if an error contains security-sensitive information
func IsSecuritySensitive(err error) bool {
	if err == nil {
		return false
	}
	
	errMsg := strings.ToLower(err.Error())
	sensitivePatterns := []string{
		"permission denied",
		"access denied", 
		"unauthorized",
		"forbidden",
		"authentication",
		"credential",
		"password",
		"token",
		"key",
		"certificate",
	}
	
	for _, pattern := range sensitivePatterns {
		if strings.Contains(errMsg, pattern) {
			return true
		}
	}
	
	return false
}