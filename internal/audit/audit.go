package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"hosts-manager/pkg/platform"
)

// EventType represents the type of audit event
type EventType string

const (
	EventHostsAdd      EventType = "hosts_add"
	EventHostsDelete   EventType = "hosts_delete"
	EventHostsModify   EventType = "hosts_modify"
	EventHostsEnable   EventType = "hosts_enable"
	EventHostsDisable  EventType = "hosts_disable"
	EventBackupCreate  EventType = "backup_create"
	EventBackupRestore EventType = "backup_restore"
	EventBackupDelete  EventType = "backup_delete"
	EventConfigEdit    EventType = "config_edit"
	EventImportFile    EventType = "import_file"
	EventExportFile    EventType = "export_file"
	EventPrivilegeEsc  EventType = "privilege_escalation"
	EventValidationFail EventType = "validation_failure"
	EventSecurityViol  EventType = "security_violation"
	EventFileAccess    EventType = "file_access"
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
	Timestamp   time.Time              `json:"timestamp"`
	EventType   EventType              `json:"event_type"`
	Severity    Severity               `json:"severity"`
	UserID      int                    `json:"user_id"`
	Username    string                 `json:"username"`
	ProcessID   int                    `json:"process_id"`
	Operation   string                 `json:"operation"`
	Resource    string                 `json:"resource"`
	Success     bool                   `json:"success"`
	ErrorMsg    string                 `json:"error_message,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
	IPAddress   string                 `json:"ip_address,omitempty"`
	UserAgent   string                 `json:"user_agent,omitempty"`
	SessionID   string                 `json:"session_id,omitempty"`
}

// Logger handles security audit logging
type Logger struct {
	logPath    string
	enabled    bool
	minLevel   Severity
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
		logPath:  logPath,
		enabled:  true,
		minLevel: SeverityInfo,
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

	// Append to audit log file with secure permissions
	file, err := os.OpenFile(l.logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return fmt.Errorf("failed to open audit log: %w", err)
	}
	defer file.Close()

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
		Operation: operation,
		Resource:  resource,
		Success:   false,
		ErrorMsg:  reason,
		Details:   details,
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
		"input_type": inputType,
		"input_data": input,
		"failure_reason": reason,
	}

	event := AuditEvent{
		EventType: EventValidationFail,
		Severity:  SeverityWarning,
		Operation: "input_validation",
		Resource:  inputType,
		Success:   false,
		ErrorMsg:  reason,
		Details:   details,
	}

	l.Log(event)
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

	l.Log(event)
}

// LogFileOperation logs file access operations
func (l *Logger) LogFileOperation(operation, filePath string, success bool, errorMsg string) {
	severity := SeverityInfo
	if !success {
		severity = SeverityError
	}

	details := map[string]interface{}{
		"file_path": filePath,
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

	l.Log(event)
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
		"ip_address": ip,
		"hostnames": hostnames,
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

	l.Log(event)
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
		"backup_path": backupPath,
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

	l.Log(event)
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
	defer file.Close()

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