package audit

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/brandonhon/hosts-manager/pkg/platform"
)

// EventType represents the type of audit event
type EventType string

const (
	EventHostsAdd       EventType = "hosts_add"
	EventHostsDelete    EventType = "hosts_delete"
	EventHostsModify    EventType = "hosts_modify"
	EventHostsEnable    EventType = "hosts_enable"
	EventHostsDisable   EventType = "hosts_disable"
	EventBackupCreate   EventType = "backup_create"
	EventBackupRestore  EventType = "backup_restore"
	EventBackupDelete   EventType = "backup_delete"
	EventConfigEdit     EventType = "config_edit"
	EventImportFile     EventType = "import_file"
	EventExportFile     EventType = "export_file"
	EventPrivilegeEsc   EventType = "privilege_escalation"
	EventValidationFail EventType = "validation_failure"
	EventSecurityViol   EventType = "security_violation"
	EventFileAccess     EventType = "file_access"
)

// Severity represents the severity level of an audit event
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityError    Severity = "error"
	SeverityCritical Severity = "critical"
)

// AuditEvent represents a single audit event
type AuditEvent struct {
	Timestamp time.Time              `json:"timestamp"`
	EventType EventType              `json:"event_type"`
	Severity  Severity               `json:"severity"`
	UserID    int                    `json:"user_id"`
	Username  string                 `json:"username"`
	ProcessID int                    `json:"process_id"`
	Operation string                 `json:"operation"`
	Resource  string                 `json:"resource"`
	Success   bool                   `json:"success"`
	ErrorMsg  string                 `json:"error_message,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
	IPAddress string                 `json:"ip_address,omitempty"`
	UserAgent string                 `json:"user_agent,omitempty"`
	SessionID string                 `json:"session_id,omitempty"`
}

// Logger handles security audit logging
type Logger struct {
	logPath    string
	enabled    bool
	minLevel   Severity
	maxLogSize int64 // Maximum size in bytes before rotation
	maxLogs    int   // Maximum number of rotated logs to keep
}

// NewLogger creates a new audit logger
func NewLogger() (*Logger, error) {
	p := platform.New()
	logDir := filepath.Join(p.GetDataDir(), "audit")

	// Create audit log directory with secure permissions
	if err := os.MkdirAll(logDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create audit log directory: %w", err)
	}

	logPath := filepath.Join(logDir, "audit.log")

	return &Logger{
		logPath:    logPath,
		enabled:    true,
		minLevel:   SeverityInfo,
		maxLogSize: 10 * 1024 * 1024, // 10MB default
		maxLogs:    5,                // Keep 5 rotated logs
	}, nil
}

// Log records an audit event
func (l *Logger) Log(event AuditEvent) error {
	if !l.enabled {
		return nil
	}

	// Set default timestamp if not provided
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	// Set user information if not provided
	if event.UserID == 0 {
		event.UserID = os.Getuid()
	}
	if event.Username == "" {
		if user := os.Getenv("USER"); user != "" {
			event.Username = user
		} else if user := os.Getenv("USERNAME"); user != "" {
			event.Username = user
		}
	}
	if event.ProcessID == 0 {
		event.ProcessID = os.Getpid()
	}

	// Serialize event to JSON
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to serialize audit event: %w", err)
	}

	// Check if log rotation is needed
	if err := l.rotateIfNeeded(); err != nil {
		// Log rotation failure shouldn't prevent logging, but we should note it
		fmt.Fprintf(os.Stderr, "Warning: audit log rotation failed: %v\n", err)
	}

	// Append to audit log file with secure permissions
	file, err := os.OpenFile(l.logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return fmt.Errorf("failed to open audit log: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close audit log file: %v\n", err)
		}
	}()

	// Write event with newline
	if _, err := file.WriteString(string(eventJSON) + "\n"); err != nil {
		return fmt.Errorf("failed to write audit event: %w", err)
	}

	// Flush to disk immediately for security events
	if event.Severity == SeverityCritical || event.Severity == SeverityError {
		if err := file.Sync(); err != nil {
			return fmt.Errorf("failed to sync audit log: %w", err)
		}
	}

	return nil
}

// LogSecurityViolation logs a security violation event
func (l *Logger) LogSecurityViolation(operation, resource, reason string, details map[string]interface{}) {
	event := AuditEvent{
		EventType: EventSecurityViol,
		Severity:  SeverityCritical,
		Operation: sanitizeForAuditLog(operation),
		Resource:  sanitizeForAuditLog(resource),
		Success:   false,
		ErrorMsg:  sanitizeForAuditLog(reason),
		Details:   sanitizeMapForAuditLog(details),
	}

	if err := l.Log(event); err != nil {
		// If we can't log security violations, write to stderr as fallback
		fmt.Fprintf(os.Stderr, "AUDIT LOG FAILURE: %v - Original violation: %s on %s: %s\n",
			err, operation, resource, reason)
	}
}

// LogValidationFailure logs input validation failures
func (l *Logger) LogValidationFailure(input, inputType, reason string) {
	details := map[string]interface{}{
		"input_type":     inputType,
		"input_data":     input,
		"failure_reason": reason,
	}

	event := AuditEvent{
		EventType: EventValidationFail,
		Severity:  SeverityWarning,
		Operation: "input_validation",
		Resource:  sanitizeForAuditLog(inputType),
		Success:   false,
		ErrorMsg:  sanitizeForAuditLog(reason),
		Details:   sanitizeMapForAuditLog(details),
	}

	_ = l.Log(event) // Intentionally ignore error for audit logging
}

// LogPrivilegeEscalation logs privilege escalation attempts
func (l *Logger) LogPrivilegeEscalation(operation string, success bool, errorMsg string) {
	severity := SeverityInfo
	if !success {
		severity = SeverityWarning
	}

	event := AuditEvent{
		EventType: EventPrivilegeEsc,
		Severity:  severity,
		Operation: operation,
		Resource:  "system_privileges",
		Success:   success,
		ErrorMsg:  errorMsg,
	}

	_ = l.Log(event) // Intentionally ignore error for audit logging
}

// LogFileOperation logs file access operations
func (l *Logger) LogFileOperation(operation, filePath string, success bool, errorMsg string) {
	severity := SeverityInfo
	if !success {
		severity = SeverityError
	}

	details := map[string]interface{}{
		"file_path":      filePath,
		"operation_type": operation,
	}

	event := AuditEvent{
		EventType: EventFileAccess,
		Severity:  severity,
		Operation: operation,
		Resource:  filePath,
		Success:   success,
		ErrorMsg:  errorMsg,
		Details:   details,
	}

	_ = l.Log(event) // Intentionally ignore error for audit logging
}

// LogHostsOperation logs hosts file modification operations
func (l *Logger) LogHostsOperation(operation string, ip string, hostnames []string, success bool, errorMsg string) {
	var eventType EventType
	switch operation {
	case "add":
		eventType = EventHostsAdd
	case "delete":
		eventType = EventHostsDelete
	case "enable":
		eventType = EventHostsEnable
	case "disable":
		eventType = EventHostsDisable
	default:
		eventType = EventHostsModify
	}

	severity := SeverityInfo
	if !success {
		severity = SeverityError
	}

	details := map[string]interface{}{
		"ip_address":     ip,
		"hostnames":      hostnames,
		"operation_type": operation,
	}

	event := AuditEvent{
		EventType: eventType,
		Severity:  severity,
		Operation: operation,
		Resource:  "hosts_file",
		Success:   success,
		ErrorMsg:  errorMsg,
		Details:   details,
	}

	_ = l.Log(event) // Intentionally ignore error for audit logging
}

// LogBackupOperation logs backup-related operations
func (l *Logger) LogBackupOperation(operation, backupPath string, success bool, errorMsg string) {
	var eventType EventType
	switch operation {
	case "create":
		eventType = EventBackupCreate
	case "restore":
		eventType = EventBackupRestore
	case "delete":
		eventType = EventBackupDelete
	default:
		eventType = EventBackupCreate
	}

	severity := SeverityInfo
	if !success {
		severity = SeverityError
	}

	details := map[string]interface{}{
		"backup_path":    backupPath,
		"operation_type": operation,
	}

	event := AuditEvent{
		EventType: eventType,
		Severity:  severity,
		Operation: operation,
		Resource:  "backup_system",
		Success:   success,
		ErrorMsg:  errorMsg,
		Details:   details,
	}

	_ = l.Log(event) // Intentionally ignore error for audit logging
}

// GetLogPath returns the path to the audit log file
func (l *Logger) GetLogPath() string {
	return l.logPath
}

// IsEnabled returns whether audit logging is enabled
func (l *Logger) IsEnabled() bool {
	return l.enabled
}

// SetEnabled enables or disables audit logging
func (l *Logger) SetEnabled(enabled bool) {
	l.enabled = enabled
}

// GetRecentEvents retrieves recent audit events (for security monitoring)
func (l *Logger) GetRecentEvents(limit int) ([]AuditEvent, error) {
	file, err := os.Open(l.logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []AuditEvent{}, nil
		}
		return nil, fmt.Errorf("failed to open audit log: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close audit log file: %v\n", err)
		}
	}()

	var events []AuditEvent
	decoder := json.NewDecoder(file)

	for decoder.More() && len(events) < limit {
		var event AuditEvent
		if err := decoder.Decode(&event); err != nil {
			continue // Skip malformed entries
		}
		events = append(events, event)
	}

	return events, nil
}

// rotateIfNeeded checks if the current log file exceeds the size limit and rotates it
func (l *Logger) rotateIfNeeded() error {
	// Check current log file size
	info, err := os.Stat(l.logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No log file yet, no rotation needed
		}
		return fmt.Errorf("failed to stat audit log: %w", err)
	}

	// If size is under limit, no rotation needed
	if info.Size() < l.maxLogSize {
		return nil
	}

	// Perform rotation
	return l.rotateLog()
}

// rotateLog performs the actual log rotation
func (l *Logger) rotateLog() error {
	logDir := filepath.Dir(l.logPath)
	logBasename := filepath.Base(l.logPath)

	// Remove the oldest log if we have too many
	oldestLog := filepath.Join(logDir, fmt.Sprintf("%s.%d", logBasename, l.maxLogs))
	if _, err := os.Stat(oldestLog); err == nil {
		if err := os.Remove(oldestLog); err != nil {
			return fmt.Errorf("failed to remove oldest log: %w", err)
		}
	}

	// Shift existing rotated logs
	for i := l.maxLogs - 1; i >= 1; i-- {
		oldName := filepath.Join(logDir, fmt.Sprintf("%s.%d", logBasename, i))
		newName := filepath.Join(logDir, fmt.Sprintf("%s.%d", logBasename, i+1))

		if _, err := os.Stat(oldName); err == nil {
			if err := os.Rename(oldName, newName); err != nil {
				return fmt.Errorf("failed to rotate log %s to %s: %w", oldName, newName, err)
			}
		}
	}

	// Move current log to .1
	rotatedName := filepath.Join(logDir, fmt.Sprintf("%s.1", logBasename))
	if err := os.Rename(l.logPath, rotatedName); err != nil {
		return fmt.Errorf("failed to rotate current log: %w", err)
	}

	// Compress rotated log to save space
	if err := l.compressLog(rotatedName); err != nil {
		// Compression failure is not critical, just log it
		fmt.Fprintf(os.Stderr, "Warning: failed to compress rotated log %s: %v\n", rotatedName, err)
	}

	return nil
}

// compressLog compresses a rotated log file with streaming and size limits
func (l *Logger) compressLog(logPath string) error {
	// Check file size before compression to prevent memory exhaustion
	fileInfo, err := os.Stat(logPath)
	if err != nil {
		return fmt.Errorf("failed to stat log file: %w", err)
	}

	// Set maximum file size for compression (100MB)
	const maxCompressionSize = 100 * 1024 * 1024
	if fileInfo.Size() > maxCompressionSize {
		return fmt.Errorf("log file too large for compression: %d bytes (max: %d)", fileInfo.Size(), maxCompressionSize)
	}

	// Create compressed file first
	compressedPath := logPath + ".gz"
	compressedFile, err := os.Create(compressedPath)
	if err != nil {
		return fmt.Errorf("failed to create compressed log: %w", err)
	}

	// Create gzip writer with optimized compression settings
	gzipWriter, err := gzip.NewWriterLevel(compressedFile, gzip.BestSpeed)
	if err != nil {
		_ = compressedFile.Close()
		return fmt.Errorf("failed to create gzip writer: %w", err)
	}

	// Read and compress the original file
	originalFile, err := os.Open(logPath)
	if err != nil {
		_ = gzipWriter.Close()
		_ = compressedFile.Close()
		return fmt.Errorf("failed to open log for compression: %w", err)
	}

	// Stream the file in chunks to avoid loading entire file in memory
	const bufferSize = 64 * 1024 // 64KB chunks
	buffer := make([]byte, bufferSize)

	for {
		n, err := originalFile.Read(buffer)
		if err != nil {
			if err == io.EOF {
				break
			}
			_ = originalFile.Close()
			_ = gzipWriter.Close()
			_ = compressedFile.Close()
			return fmt.Errorf("failed to read from original log: %w", err)
		}

		if _, writeErr := gzipWriter.Write(buffer[:n]); writeErr != nil {
			_ = originalFile.Close()
			_ = gzipWriter.Close()
			_ = compressedFile.Close()
			return fmt.Errorf("failed to write compressed data: %w", writeErr)
		}
	}

	// Close original file first (important for Windows)
	if err := originalFile.Close(); err != nil {
		_ = gzipWriter.Close()
		_ = compressedFile.Close()
		return fmt.Errorf("failed to close original file: %w", err)
	}

	// Finalize compression
	if err := gzipWriter.Close(); err != nil {
		_ = compressedFile.Close()
		return fmt.Errorf("failed to finalize compression: %w", err)
	}

	if err := compressedFile.Close(); err != nil {
		return fmt.Errorf("failed to close compressed file: %w", err)
	}

	// Verify compressed file was created successfully
	if compressedInfo, statErr := os.Stat(compressedPath); statErr != nil || compressedInfo.Size() == 0 {
		return fmt.Errorf("compressed file verification failed")
	}

	// Remove original uncompressed file only after successful compression and file closure
	if err := os.Remove(logPath); err != nil {
		return fmt.Errorf("failed to remove original log after compression: %w", err)
	}

	return nil
}

// SetMaxLogSize sets the maximum log size before rotation
func (l *Logger) SetMaxLogSize(size int64) {
	l.maxLogSize = size
}

// SetMaxLogs sets the maximum number of rotated logs to keep
func (l *Logger) SetMaxLogs(count int) {
	l.maxLogs = count
}

// sanitizeForAuditLog sanitizes input to prevent log injection attacks
func sanitizeForAuditLog(input string) string {
	// Remove or replace dangerous characters that could be used for log injection
	var result strings.Builder
	maxLength := 1000 // Limit length to prevent excessive log sizes

	for i, r := range input {
		if i >= maxLength {
			result.WriteString("...[truncated]")
			break
		}

		// Replace control characters and log injection patterns
		switch {
		case r == '\n':
			result.WriteString("\\n")
		case r == '\r':
			result.WriteString("\\r")
		case r == '\t':
			result.WriteString("\\t")
		case unicode.IsControl(r):
			result.WriteString(fmt.Sprintf("\\u%04x", r))
		case r == '"':
			result.WriteString("\\\"")
		case r == '\\':
			result.WriteString("\\\\")
		default:
			result.WriteRune(r)
		}
	}

	return result.String()
}

// sanitizeMapForAuditLog sanitizes a map of interface{} values for audit logging
func sanitizeMapForAuditLog(data map[string]interface{}) map[string]interface{} {
	if data == nil {
		return nil
	}

	sanitized := make(map[string]interface{})
	for key, value := range data {
		sanitizedKey := sanitizeForAuditLog(key)

		switch v := value.(type) {
		case string:
			sanitized[sanitizedKey] = sanitizeForAuditLog(v)
		case []string:
			sanitizedSlice := make([]string, len(v))
			for i, s := range v {
				sanitizedSlice[i] = sanitizeForAuditLog(s)
			}
			sanitized[sanitizedKey] = sanitizedSlice
		default:
			// For other types, convert to string and sanitize
			sanitized[sanitizedKey] = sanitizeForAuditLog(fmt.Sprintf("%v", v))
		}
	}

	return sanitized
}
