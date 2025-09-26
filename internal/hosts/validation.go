package hosts

import (
	"fmt"
	"net"
	"regexp"
	"strings"
	"unicode"

	"hosts-manager/internal/audit"
)

var (
	// RFC-compliant hostname validation
	hostnameRegex = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?)*$`)

	// Dangerous patterns to reject
	dangerousHostnamePatterns = []*regexp.Regexp{
		regexp.MustCompile(`\.\./`),                    // Path traversal
		regexp.MustCompile(`[<>\"'&]`),                 // HTML/Script injection chars
		regexp.MustCompile(`[\x00-\x1f\x7f-\x9f]`),     // Control characters
		regexp.MustCompile(`^\.+$`),                    // Only dots
		regexp.MustCompile(`\.$`),                      // Trailing dot (may cause issues)
		regexp.MustCompile(`\s`),                       // Whitespace
	}
)

// ValidateIP performs comprehensive IP address validation
func ValidateIP(ip string) error {
	if ip == "" {
		logValidationFailure(ip, "ip_address", "IP address cannot be empty")
		return fmt.Errorf("IP address cannot be empty")
	}

	// Basic format validation
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		logValidationFailure(ip, "ip_address", "invalid IP address format")
		return fmt.Errorf("invalid IP address format: %s", ip)
	}

	// Convert to standard format for consistent checking
	ipStr := parsedIP.String()

	// Security checks for potentially dangerous IP addresses
	if err := validateIPSecurity(parsedIP); err != nil {
		return fmt.Errorf("IP address security validation failed: %w", err)
	}

	// Check for suspicious patterns
	if strings.Contains(ipStr, "..") {
		return fmt.Errorf("IP address contains suspicious pattern: %s", ip)
	}

	return nil
}

// validateIPSecurity checks for security-sensitive IP ranges
func validateIPSecurity(ip net.IP) error {
	// Allow localhost entries - these are common and legitimate
	if ip.IsLoopback() {
		return nil
	}

	// Allow private networks - these are also legitimate for local development
	if isPrivateIP(ip) {
		return nil
	}

	// Allow public IPs
	if !ip.IsPrivate() && !ip.IsLoopback() && !ip.IsMulticast() && !ip.IsUnspecified() {
		return nil
	}

	// Reject special-use addresses that could be problematic
	if ip.IsMulticast() {
		return fmt.Errorf("multicast IP addresses not allowed: %s", ip.String())
	}

	if ip.IsUnspecified() {
		return fmt.Errorf("unspecified IP addresses not allowed: %s", ip.String())
	}

	// Check for IPv6 special addresses
	if ip.To4() == nil { // IPv6
		ipStr := ip.String()

		// Reject IPv6 special addresses that could be problematic
		dangerousIPv6Prefixes := []string{
			"ff", // Multicast (already caught above, but double-check)
			"fe80", // Link-local (could be problematic in some contexts)
		}

		for _, prefix := range dangerousIPv6Prefixes {
			if strings.HasPrefix(strings.ToLower(ipStr), prefix) {
				return fmt.Errorf("potentially problematic IPv6 address: %s", ipStr)
			}
		}
	}

	return nil
}

// isPrivateIP checks if an IP is in private ranges (more comprehensive than Go's IsPrivate)
func isPrivateIP(ip net.IP) bool {
	// Standard private ranges
	if ip.IsPrivate() {
		return true
	}

	// Additional ranges to consider private/local
	privateRanges := []string{
		"127.0.0.0/8",     // Loopback
		"169.254.0.0/16",  // Link-local
		"::1/128",         // IPv6 loopback
		"fc00::/7",        // IPv6 unique local
		"fe80::/10",       // IPv6 link-local
	}

	for _, cidr := range privateRanges {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if network.Contains(ip) {
			return true
		}
	}

	return false
}

// ValidateHostname performs comprehensive hostname validation
func ValidateHostname(hostname string) error {
	if hostname == "" {
		logValidationFailure(hostname, "hostname", "hostname cannot be empty")
		return fmt.Errorf("hostname cannot be empty")
	}

	// Length validation
	if len(hostname) > 253 {
		logValidationFailure(hostname, "hostname", "hostname too long (max 253 characters)")
		return fmt.Errorf("hostname too long (max 253 characters): %s", hostname)
	}

	// Basic format validation using RFC-compliant regex
	if !hostnameRegex.MatchString(hostname) {
		logValidationFailure(hostname, "hostname", "invalid hostname format")
		return fmt.Errorf("invalid hostname format: %s", hostname)
	}

	// Security validation
	if err := validateHostnameSecurity(hostname); err != nil {
		return fmt.Errorf("hostname security validation failed: %w", err)
	}

	// Validate each label (part between dots)
	labels := strings.Split(hostname, ".")
	for _, label := range labels {
		if err := validateHostnameLabel(label); err != nil {
			return fmt.Errorf("invalid hostname label '%s': %w", label, err)
		}
	}

	return nil
}

// validateHostnameSecurity checks for security issues in hostnames
func validateHostnameSecurity(hostname string) error {
	// Check against dangerous patterns
	for _, pattern := range dangerousHostnamePatterns {
		if pattern.MatchString(hostname) {
			return fmt.Errorf("hostname contains dangerous pattern: %s", hostname)
		}
	}

	// Check for suspicious Unicode characters
	for _, r := range hostname {
		if r > unicode.MaxASCII {
			return fmt.Errorf("hostname contains non-ASCII characters (potential IDN attack): %s", hostname)
		}

		// Reject various control and special characters
		if unicode.IsControl(r) || unicode.IsSpace(r) {
			return fmt.Errorf("hostname contains control or space characters: %s", hostname)
		}
	}

	// Check for homograph attacks (similar-looking characters)
	if containsHomographs(hostname) {
		return fmt.Errorf("hostname may contain homograph attack characters: %s", hostname)
	}

	// Reject overly long labels that could cause buffer overflows
	for _, label := range strings.Split(hostname, ".") {
		if len(label) > 63 {
			return fmt.Errorf("hostname label too long (max 63 characters): %s", label)
		}
	}

	return nil
}

// validateHostnameLabel validates individual hostname labels
func validateHostnameLabel(label string) error {
	if label == "" {
		return fmt.Errorf("empty label")
	}

	if len(label) > 63 {
		return fmt.Errorf("label too long (max 63 characters)")
	}

	// Labels cannot start or end with hyphens
	if strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
		return fmt.Errorf("label cannot start or end with hyphen")
	}

	// Ensure all characters are valid
	for _, r := range label {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-') {
			return fmt.Errorf("label contains invalid character: %c", r)
		}
	}

	return nil
}

// containsHomographs checks for potential homograph attack characters
func containsHomographs(hostname string) bool {
	// Simple check for common homograph characters
	// In a production system, this would be more comprehensive
	suspiciousChars := []rune{
		0x430, // Cyrillic 'a'
		0x043e, // Cyrillic 'o'
		0x0440, // Cyrillic 'p'
		0x0435, // Cyrillic 'e'
		// Add more as needed
	}

	for _, char := range hostname {
		for _, suspicious := range suspiciousChars {
			if char == suspicious {
				return true
			}
		}
	}

	return false
}

// ValidateComment validates comments for security issues
func ValidateComment(comment string) error {
	if comment == "" {
		return nil // Empty comments are allowed
	}

	// Length validation
	if len(comment) > 500 {
		return fmt.Errorf("comment too long (max 500 characters)")
	}

	// Check for control characters and potential script injection
	for _, r := range comment {
		if unicode.IsControl(r) && r != '\t' && r != '\n' && r != '\r' {
			return fmt.Errorf("comment contains control characters")
		}
	}

	// Check for potential script injection patterns
	dangerousPatterns := []string{
		"<script",
		"javascript:",
		"data:",
		"vbscript:",
		"onload=",
		"onerror=",
		"eval(",
		"setTimeout(",
		"setInterval(",
	}

	commentLower := strings.ToLower(comment)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(commentLower, pattern) {
			return fmt.Errorf("comment contains potentially dangerous content")
		}
	}

	return nil
}

// ValidateEntry performs comprehensive validation of a hosts entry
func ValidateEntry(entry Entry) error {
	// Validate IP address
	if err := ValidateIP(entry.IP); err != nil {
		return fmt.Errorf("invalid IP address: %w", err)
	}

	// Validate hostnames
	if len(entry.Hostnames) == 0 {
		return fmt.Errorf("entry must have at least one hostname")
	}

	for _, hostname := range entry.Hostnames {
		if err := ValidateHostname(hostname); err != nil {
			return fmt.Errorf("invalid hostname: %w", err)
		}
	}

	// Validate comment
	if err := ValidateComment(entry.Comment); err != nil {
		return fmt.Errorf("invalid comment: %w", err)
	}

	// Validate category name
	if entry.Category != "" {
		if err := validateCategoryName(entry.Category); err != nil {
			return fmt.Errorf("invalid category: %w", err)
		}
	}

	return nil
}

// validateCategoryName validates category names
func validateCategoryName(category string) error {
	if category == "" {
		return fmt.Errorf("category name cannot be empty")
	}

	if len(category) > 50 {
		return fmt.Errorf("category name too long (max 50 characters)")
	}

	// Category names should be simple and safe
	categoryRegex := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !categoryRegex.MatchString(category) {
		return fmt.Errorf("category name contains invalid characters (only a-z, A-Z, 0-9, _, - allowed)")
	}

	return nil
}
// logValidationFailure logs validation failures for security monitoring
func logValidationFailure(input, inputType, reason string) {
	// Create audit logger (ignore errors for logging)
	if logger, err := audit.NewLogger(); err == nil {
		logger.LogValidationFailure(input, inputType, reason)
	}
}
