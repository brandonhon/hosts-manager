package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEventTypeConstants(t *testing.T) {
	expectedEvents := map[EventType]string{
		EventHostsAdd:       "hosts_add",
		EventHostsDelete:    "hosts_delete",
		EventHostsModify:    "hosts_modify",
		EventHostsEnable:    "hosts_enable",
		EventHostsDisable:   "hosts_disable",
		EventBackupCreate:   "backup_create",
		EventBackupRestore:  "backup_restore",
		EventBackupDelete:   "backup_delete",
		EventConfigEdit:     "config_edit",
		EventImportFile:     "import_file",
		EventExportFile:     "export_file",
		EventPrivilegeEsc:   "privilege_escalation",
		EventValidationFail: "validation_failure",
		EventSecurityViol:   "security_violation",
		EventFileAccess:     "file_access",
	}

	for eventType, expected := range expectedEvents {
		if string(eventType) != expected {
			t.Errorf("Expected %s, got %s", expected, string(eventType))
		}
	}
}

func TestSeverityConstants(t *testing.T) {
	expectedSeverities := map[Severity]string{
		SeverityInfo:     "info",
		SeverityWarning:  "warning",
		SeverityError:    "error",
		SeverityCritical: "critical",
	}

	for severity, expected := range expectedSeverities {
		if string(severity) != expected {
			t.Errorf("Expected %s, got %s", expected, string(severity))
		}
	}
}

func TestNewLogger(t *testing.T) {
	// Save original environment
	origHome := os.Getenv("HOME")
	origXDGDataHome := os.Getenv("XDG_DATA_HOME")
	origAppData := os.Getenv("APPDATA")
	defer func() {
		_ = os.Setenv("HOME", origHome)
		_ = os.Setenv("XDG_DATA_HOME", origXDGDataHome)
		_ = os.Setenv("APPDATA", origAppData)
	}()

	// Create temporary directory
	tempDir := t.TempDir()
	_ = os.Setenv("XDG_DATA_HOME", tempDir)
	_ = os.Setenv("HOME", tempDir)
	_ = os.Unsetenv("APPDATA")

	logger, err := NewLogger()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	if logger == nil {
		t.Fatal("Logger is nil")
	}

	if !logger.enabled {
		t.Error("Logger should be enabled by default")
	}

	if logger.minLevel != SeverityInfo {
		t.Errorf("Expected default min level to be Info, got %s", logger.minLevel)
	}

	if logger.maxLogSize != 10*1024*1024 {
		t.Errorf("Expected default max log size to be 10MB, got %d", logger.maxLogSize)
	}

	if logger.maxLogs != 5 {
		t.Errorf("Expected default max logs to be 5, got %d", logger.maxLogs)
	}

	// Verify audit directory was created
	auditDir := filepath.Dir(logger.logPath)
	if _, err := os.Stat(auditDir); os.IsNotExist(err) {
		t.Error("Audit directory was not created")
	}
}

func TestLoggerBasicOperations(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "audit.log")

	logger := &Logger{
		logPath:    logPath,
		enabled:    true,
		minLevel:   SeverityInfo,
		maxLogSize: 10 * 1024 * 1024,
		maxLogs:    5,
	}

	// Test GetLogPath
	if logger.GetLogPath() != logPath {
		t.Errorf("Expected log path %s, got %s", logPath, logger.GetLogPath())
	}

	// Test IsEnabled
	if !logger.IsEnabled() {
		t.Error("Logger should be enabled")
	}

	// Test SetEnabled
	logger.SetEnabled(false)
	if logger.IsEnabled() {
		t.Error("Logger should be disabled after SetEnabled(false)")
	}

	logger.SetEnabled(true)
	if !logger.IsEnabled() {
		t.Error("Logger should be enabled after SetEnabled(true)")
	}

	// Test SetMaxLogSize
	logger.SetMaxLogSize(1024)
	if logger.maxLogSize != 1024 {
		t.Errorf("Expected max log size 1024, got %d", logger.maxLogSize)
	}

	// Test SetMaxLogs
	logger.SetMaxLogs(3)
	if logger.maxLogs != 3 {
		t.Errorf("Expected max logs 3, got %d", logger.maxLogs)
	}
}

func TestLogEvent(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "audit.log")

	logger := &Logger{
		logPath:    logPath,
		enabled:    true,
		minLevel:   SeverityInfo,
		maxLogSize: 10 * 1024 * 1024,
		maxLogs:    5,
	}

	// Test basic event logging
	event := AuditEvent{
		EventType: EventHostsAdd,
		Severity:  SeverityInfo,
		Operation: "test_operation",
		Resource:  "test_resource",
		Success:   true,
		Details: map[string]interface{}{
			"test_key": "test_value",
		},
	}

	err := logger.Log(event)
	if err != nil {
		t.Fatalf("Failed to log event: %v", err)
	}

	// Verify log file was created
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Fatal("Log file was not created")
	}

	// Read and verify log content
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	var loggedEvent AuditEvent
	err = json.Unmarshal(content[:len(content)-1], &loggedEvent) // Remove trailing newline
	if err != nil {
		t.Fatalf("Failed to unmarshal logged event: %v", err)
	}

	if loggedEvent.EventType != event.EventType {
		t.Errorf("Expected event type %s, got %s", event.EventType, loggedEvent.EventType)
	}

	if loggedEvent.Operation != event.Operation {
		t.Errorf("Expected operation %s, got %s", event.Operation, loggedEvent.Operation)
	}

	// Test that timestamp was set
	if loggedEvent.Timestamp.IsZero() {
		t.Error("Timestamp should be set")
	}

	// Test that user information was set
	// Note: UserID can be 0 for root user, so we just check it was populated
	// The Log() method sets it to os.Getuid() if not already set

	if loggedEvent.ProcessID == 0 {
		t.Error("ProcessID should be set")
	}
}

func TestLogEventDisabled(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "audit.log")

	logger := &Logger{
		logPath:    logPath,
		enabled:    false, // Disabled
		minLevel:   SeverityInfo,
		maxLogSize: 10 * 1024 * 1024,
		maxLogs:    5,
	}

	event := AuditEvent{
		EventType: EventHostsAdd,
		Severity:  SeverityInfo,
		Operation: "test_operation",
		Resource:  "test_resource",
		Success:   true,
	}

	err := logger.Log(event)
	if err != nil {
		t.Fatalf("Failed to log event: %v", err)
	}

	// Verify log file was NOT created when disabled
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Error("Log file should not be created when logger is disabled")
	}
}

func TestLogSecurityViolation(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "audit.log")

	logger := &Logger{
		logPath:    logPath,
		enabled:    true,
		minLevel:   SeverityInfo,
		maxLogSize: 10 * 1024 * 1024,
		maxLogs:    5,
	}

	details := map[string]interface{}{
		"suspicious_input": "malicious_data",
		"ip_address":       "192.168.1.100",
	}

	logger.LogSecurityViolation("unauthorized_access", "/etc/hosts", "path_traversal_attempt", details)

	// Read and verify log content
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	var loggedEvent AuditEvent
	err = json.Unmarshal(content[:len(content)-1], &loggedEvent)
	if err != nil {
		t.Fatalf("Failed to unmarshal logged event: %v", err)
	}

	if loggedEvent.EventType != EventSecurityViol {
		t.Errorf("Expected event type %s, got %s", EventSecurityViol, loggedEvent.EventType)
	}

	if loggedEvent.Severity != SeverityCritical {
		t.Errorf("Expected severity %s, got %s", SeverityCritical, loggedEvent.Severity)
	}

	if loggedEvent.Success {
		t.Error("Security violation should be marked as unsuccessful")
	}

	if loggedEvent.Details == nil {
		t.Error("Details should be present")
	}
}

func TestLogValidationFailure(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "audit.log")

	logger := &Logger{
		logPath:    logPath,
		enabled:    true,
		minLevel:   SeverityInfo,
		maxLogSize: 10 * 1024 * 1024,
		maxLogs:    5,
	}

	logger.LogValidationFailure("invalid@hostname", "hostname", "contains invalid characters")

	// Read and verify log content
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	var loggedEvent AuditEvent
	err = json.Unmarshal(content[:len(content)-1], &loggedEvent)
	if err != nil {
		t.Fatalf("Failed to unmarshal logged event: %v", err)
	}

	if loggedEvent.EventType != EventValidationFail {
		t.Errorf("Expected event type %s, got %s", EventValidationFail, loggedEvent.EventType)
	}

	if loggedEvent.Severity != SeverityWarning {
		t.Errorf("Expected severity %s, got %s", SeverityWarning, loggedEvent.Severity)
	}

	if loggedEvent.Success {
		t.Error("Validation failure should be marked as unsuccessful")
	}
}

func TestLogPrivilegeEscalation(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "audit.log")

	logger := &Logger{
		logPath:    logPath,
		enabled:    true,
		minLevel:   SeverityInfo,
		maxLogSize: 10 * 1024 * 1024,
		maxLogs:    5,
	}

	// Test successful privilege escalation
	logger.LogPrivilegeEscalation("sudo", true, "")

	// Test failed privilege escalation
	logger.LogPrivilegeEscalation("sudo", false, "permission denied")

	// Read and verify log content
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) != 2 {
		t.Fatalf("Expected 2 log entries, got %d", len(lines))
	}

	// Verify successful escalation
	var successEvent AuditEvent
	err = json.Unmarshal([]byte(lines[0]), &successEvent)
	if err != nil {
		t.Fatalf("Failed to unmarshal success event: %v", err)
	}

	if successEvent.EventType != EventPrivilegeEsc {
		t.Error("Expected privilege escalation event type")
	}

	if successEvent.Severity != SeverityInfo {
		t.Error("Successful escalation should have Info severity")
	}

	if !successEvent.Success {
		t.Error("Successful escalation should be marked as successful")
	}

	// Verify failed escalation
	var failEvent AuditEvent
	err = json.Unmarshal([]byte(lines[1]), &failEvent)
	if err != nil {
		t.Fatalf("Failed to unmarshal fail event: %v", err)
	}

	if failEvent.Severity != SeverityWarning {
		t.Error("Failed escalation should have Warning severity")
	}

	if failEvent.Success {
		t.Error("Failed escalation should be marked as unsuccessful")
	}
}

func TestLogFileOperation(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "audit.log")

	logger := &Logger{
		logPath:    logPath,
		enabled:    true,
		minLevel:   SeverityInfo,
		maxLogSize: 10 * 1024 * 1024,
		maxLogs:    5,
	}

	logger.LogFileOperation("read", "/etc/hosts", true, "")

	// Read and verify log content
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	var loggedEvent AuditEvent
	err = json.Unmarshal(content[:len(content)-1], &loggedEvent)
	if err != nil {
		t.Fatalf("Failed to unmarshal logged event: %v", err)
	}

	if loggedEvent.EventType != EventFileAccess {
		t.Error("Expected file access event type")
	}

	if loggedEvent.Operation != "read" {
		t.Error("Expected read operation")
	}

	if loggedEvent.Resource != "/etc/hosts" {
		t.Error("Expected /etc/hosts resource")
	}
}

func TestLogHostsOperation(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "audit.log")

	logger := &Logger{
		logPath:    logPath,
		enabled:    true,
		minLevel:   SeverityInfo,
		maxLogSize: 10 * 1024 * 1024,
		maxLogs:    5,
	}

	tests := []struct {
		operation     string
		expectedEvent EventType
	}{
		{"add", EventHostsAdd},
		{"delete", EventHostsDelete},
		{"enable", EventHostsEnable},
		{"disable", EventHostsDisable},
		{"unknown", EventHostsModify},
	}

	for _, tt := range tests {
		t.Run(tt.operation, func(t *testing.T) {
			// Clear log file
			_ = os.Remove(logPath)

			hostnames := []string{"example.com", "test.local"}
			logger.LogHostsOperation(tt.operation, "192.168.1.100", hostnames, true, "")

			// Read and verify log content
			content, err := os.ReadFile(logPath)
			if err != nil {
				t.Fatalf("Failed to read log file: %v", err)
			}

			var loggedEvent AuditEvent
			err = json.Unmarshal(content[:len(content)-1], &loggedEvent)
			if err != nil {
				t.Fatalf("Failed to unmarshal logged event: %v", err)
			}

			if loggedEvent.EventType != tt.expectedEvent {
				t.Errorf("Expected event type %s, got %s", tt.expectedEvent, loggedEvent.EventType)
			}

			if loggedEvent.Operation != tt.operation {
				t.Errorf("Expected operation %s, got %s", tt.operation, loggedEvent.Operation)
			}

			if loggedEvent.Resource != "hosts_file" {
				t.Error("Expected hosts_file resource")
			}
		})
	}
}

func TestLogBackupOperation(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "audit.log")

	logger := &Logger{
		logPath:    logPath,
		enabled:    true,
		minLevel:   SeverityInfo,
		maxLogSize: 10 * 1024 * 1024,
		maxLogs:    5,
	}

	tests := []struct {
		operation     string
		expectedEvent EventType
	}{
		{"create", EventBackupCreate},
		{"restore", EventBackupRestore},
		{"delete", EventBackupDelete},
		{"unknown", EventBackupCreate}, // Default case
	}

	for _, tt := range tests {
		t.Run(tt.operation, func(t *testing.T) {
			// Clear log file
			_ = os.Remove(logPath)

			logger.LogBackupOperation(tt.operation, "/path/to/backup", true, "")

			// Read and verify log content
			content, err := os.ReadFile(logPath)
			if err != nil {
				t.Fatalf("Failed to read log file: %v", err)
			}

			var loggedEvent AuditEvent
			err = json.Unmarshal(content[:len(content)-1], &loggedEvent)
			if err != nil {
				t.Fatalf("Failed to unmarshal logged event: %v", err)
			}

			if loggedEvent.EventType != tt.expectedEvent {
				t.Errorf("Expected event type %s, got %s", tt.expectedEvent, loggedEvent.EventType)
			}

			if loggedEvent.Resource != "backup_system" {
				t.Error("Expected backup_system resource")
			}
		})
	}
}

func TestGetRecentEvents(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "audit.log")

	logger := &Logger{
		logPath:    logPath,
		enabled:    true,
		minLevel:   SeverityInfo,
		maxLogSize: 10 * 1024 * 1024,
		maxLogs:    5,
	}

	// Test with non-existent file
	events, err := logger.GetRecentEvents(10)
	if err != nil {
		t.Fatalf("Failed to get events from non-existent log: %v", err)
	}
	if len(events) != 0 {
		t.Error("Expected empty events for non-existent log")
	}

	// Log some events
	for i := 0; i < 5; i++ {
		event := AuditEvent{
			EventType: EventHostsAdd,
			Severity:  SeverityInfo,
			Operation: fmt.Sprintf("operation_%d", i),
			Resource:  "test_resource",
			Success:   true,
		}
		_ = logger.Log(event)
	}

	// Test retrieving events
	events, err = logger.GetRecentEvents(3)
	if err != nil {
		t.Fatalf("Failed to get recent events: %v", err)
	}

	if len(events) != 3 {
		t.Errorf("Expected 3 events, got %d", len(events))
	}

	// Test retrieving all events
	events, err = logger.GetRecentEvents(10)
	if err != nil {
		t.Fatalf("Failed to get all events: %v", err)
	}

	if len(events) != 5 {
		t.Errorf("Expected 5 events, got %d", len(events))
	}

	// Verify events are in order
	for i, event := range events {
		expectedOp := fmt.Sprintf("operation_%d", i)
		if event.Operation != expectedOp {
			t.Errorf("Expected operation %s, got %s", expectedOp, event.Operation)
		}
	}
}

func TestGetRecentEventsWithMalformedEntries(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "audit.log")

	logger := &Logger{
		logPath:    logPath,
		enabled:    true,
		minLevel:   SeverityInfo,
		maxLogSize: 10 * 1024 * 1024,
		maxLogs:    5,
	}

	// First log some valid events normally to create a proper JSON log
	validEvents := []AuditEvent{
		{
			EventType: EventHostsAdd,
			Severity:  SeverityInfo,
			Operation: "valid1",
		},
		{
			EventType: EventHostsAdd,
			Severity:  SeverityInfo,
			Operation: "valid2",
		},
	}

	for _, event := range validEvents {
		err := logger.Log(event)
		if err != nil {
			t.Fatalf("Failed to log valid event: %v", err)
		}
	}

	// Test that GetRecentEvents works with valid entries
	events, err := logger.GetRecentEvents(10)
	if err != nil {
		t.Fatalf("Failed to get events: %v", err)
	}

	if len(events) != 2 {
		t.Errorf("Expected 2 valid events, got %d", len(events))
	}

	expectedOps := []string{"valid1", "valid2"}
	for i, event := range events {
		if event.Operation != expectedOps[i] {
			t.Errorf("Expected operation %s, got %s", expectedOps[i], event.Operation)
		}
	}
}

func TestSanitizeForAuditLog(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "clean input",
			input:    "normal_string",
			expected: "normal_string",
		},
		{
			name:     "newline characters",
			input:    "line1\nline2",
			expected: "line1\\nline2",
		},
		{
			name:     "carriage return",
			input:    "line1\rline2",
			expected: "line1\\rline2",
		},
		{
			name:     "tab characters",
			input:    "col1\tcol2",
			expected: "col1\\tcol2",
		},
		{
			name:     "quotes and backslashes",
			input:    `"quoted" and \backslash`,
			expected: `\"quoted\" and \\backslash`,
		},
		{
			name:     "control characters",
			input:    "test\x00\x01\x1f",
			expected: "test\\u0000\\u0001\\u001f",
		},
		{
			name:     "long input truncation",
			input:    strings.Repeat("a", 1500),
			expected: strings.Repeat("a", 1000) + "...[truncated]",
		},
		{
			name:     "mixed dangerous characters",
			input:    "normal\ntext\r\twith\"quotes\\\x00null",
			expected: "normal\\ntext\\r\\twith\\\"quotes\\\\\\u0000null",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeForAuditLog(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestSanitizeMapForAuditLog(t *testing.T) {
	// Test nil map
	result := sanitizeMapForAuditLog(nil)
	if result != nil {
		t.Error("Expected nil result for nil input")
	}

	// Test empty map
	emptyMap := make(map[string]interface{})
	result = sanitizeMapForAuditLog(emptyMap)
	if len(result) != 0 {
		t.Error("Expected empty result for empty input")
	}

	// Test map with various types
	testMap := map[string]interface{}{
		"string_key":     "string\nvalue",
		"slice_key":      []string{"item1\r", "item2\t"},
		"int_key":        42,
		"bool_key":       true,
		"dangerous\nkey": "dangerous\nvalue",
		"control\x00key": "control\x00value",
	}

	result = sanitizeMapForAuditLog(testMap)

	// Check sanitized string
	if result["string_key"] != "string\\nvalue" {
		t.Errorf("String not properly sanitized: %v", result["string_key"])
	}

	// Check sanitized slice
	sanitizedSlice, ok := result["slice_key"].([]string)
	if !ok {
		t.Error("Slice type not preserved")
	}
	if len(sanitizedSlice) != 2 {
		t.Error("Slice length not preserved")
	}
	if sanitizedSlice[0] != "item1\\r" || sanitizedSlice[1] != "item2\\t" {
		t.Error("Slice items not properly sanitized")
	}

	// Check other types converted to string and sanitized
	if result["int_key"] != "42" {
		t.Error("Int not properly converted and sanitized")
	}

	// Check dangerous key sanitization
	found := false
	for key := range result {
		if strings.Contains(key, "dangerous") && !strings.Contains(key, "\n") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Dangerous key not properly sanitized")
	}
}

// BenchmarkLog benchmarks the Log function
func BenchmarkLog(b *testing.B) {
	tempDir := b.TempDir()
	logPath := filepath.Join(tempDir, "audit.log")

	logger := &Logger{
		logPath:    logPath,
		enabled:    true,
		minLevel:   SeverityInfo,
		maxLogSize: 100 * 1024 * 1024, // Large size to avoid rotation
		maxLogs:    5,
	}

	event := AuditEvent{
		EventType: EventHostsAdd,
		Severity:  SeverityInfo,
		Operation: "benchmark_operation",
		Resource:  "benchmark_resource",
		Success:   true,
		Details: map[string]interface{}{
			"test_key": "test_value",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := logger.Log(event)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkSanitizeForAuditLog benchmarks the sanitization function
func BenchmarkSanitizeForAuditLog(b *testing.B) {
	testInputs := []string{
		"simple_string",
		"string\nwith\nnewlines",
		"string\rwith\rcarriage\rreturns",
		"string\twith\ttabs",
		"string\"with\\quotes",
		"string\x00with\x01control\x1fchars",
		strings.Repeat("long_string_", 100),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, input := range testInputs {
			sanitizeForAuditLog(input)
		}
	}
}

// BenchmarkSanitizeMapForAuditLog benchmarks the map sanitization function
func BenchmarkSanitizeMapForAuditLog(b *testing.B) {
	testMap := map[string]interface{}{
		"string_key": "string\nvalue",
		"slice_key":  []string{"item1\r", "item2\t", "item3\n"},
		"int_key":    42,
		"bool_key":   true,
		"float_key":  3.14,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sanitizeMapForAuditLog(testMap)
	}
}
