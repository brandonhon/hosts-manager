package hosts

import (
	"strings"
	"testing"
)

// TestValidateIP tests IP address validation
func TestValidateIP(t *testing.T) {
	tests := []struct {
		name      string
		ip        string
		expectErr bool
	}{
		// Valid IPv4 addresses
		{name: "localhost IPv4", ip: "127.0.0.1", expectErr: false},
		{name: "private IPv4", ip: "192.168.1.1", expectErr: false},
		{name: "private IPv4 10.x", ip: "10.0.0.1", expectErr: false},
		{name: "private IPv4 172.x", ip: "172.16.0.1", expectErr: false},
		{name: "public IPv4", ip: "8.8.8.8", expectErr: false},
		{name: "edge case 0.0.0.0", ip: "0.0.0.0", expectErr: true}, // Unspecified
		{name: "edge case 255.255.255.255", ip: "255.255.255.255", expectErr: false},

		// Valid IPv6 addresses
		{name: "localhost IPv6", ip: "::1", expectErr: false},
		{name: "full IPv6", ip: "2001:db8:85a3::8a2e:370:7334", expectErr: false},
		{name: "compressed IPv6", ip: "2001:db8::1", expectErr: false},
		{name: "link-local IPv6", ip: "fe80::1", expectErr: false}, // Should be allowed with warning
		{name: "private IPv6", ip: "fc00::1", expectErr: false},

		// Invalid addresses
		{name: "empty string", ip: "", expectErr: true},
		{name: "invalid format", ip: "999.999.999.999", expectErr: true},
		{name: "incomplete IPv4", ip: "192.168.1", expectErr: true},
		{name: "too many octets", ip: "192.168.1.1.1", expectErr: true},
		{name: "text instead of IP", ip: "not.an.ip", expectErr: true},
		{name: "IPv4 with letters", ip: "192.168.a.1", expectErr: true},
		{name: "suspicious pattern", ip: "127.0.0..1", expectErr: true},

		// Multicast addresses (should be rejected)
		{name: "IPv4 multicast", ip: "224.0.0.1", expectErr: true},
		{name: "IPv6 multicast", ip: "ff02::1", expectErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateIP(tt.ip)

			if tt.expectErr && err == nil {
				t.Errorf("ValidateIP(%q) expected error but got none", tt.ip)
			}
			if !tt.expectErr && err != nil {
				t.Errorf("ValidateIP(%q) unexpected error: %v", tt.ip, err)
			}
		})
	}
}

// TestValidateHostname tests hostname validation
func TestValidateHostname(t *testing.T) {
	tests := []struct {
		name      string
		hostname  string
		expectErr bool
	}{
		// Valid hostnames
		{name: "simple hostname", hostname: "localhost", expectErr: false},
		{name: "FQDN", hostname: "api.example.com", expectErr: false},
		{name: "subdomain", hostname: "sub.api.example.com", expectErr: false},
		{name: "with numbers", hostname: "host1.example.com", expectErr: false},
		{name: "with hyphens", hostname: "my-host.example.com", expectErr: false},
		{name: "single character", hostname: "a", expectErr: false},
		{name: "max length label", hostname: strings.Repeat("a", 63) + ".com", expectErr: false},

		// Invalid hostnames
		{name: "empty string", hostname: "", expectErr: true},
		{name: "too long", hostname: strings.Repeat("a", 254), expectErr: true},
		{name: "starts with hyphen", hostname: "-invalid.com", expectErr: true},
		{name: "ends with hyphen", hostname: "invalid-.com", expectErr: true},
		{name: "label too long", hostname: strings.Repeat("a", 64) + ".com", expectErr: true},
		{name: "trailing dot", hostname: "example.com.", expectErr: true},
		{name: "consecutive dots", hostname: "example..com", expectErr: true},
		{name: "starts with dot", hostname: ".example.com", expectErr: true},
		{name: "only dots", hostname: "...", expectErr: true},
		{name: "with spaces", hostname: "my host.com", expectErr: true},
		{name: "with underscores", hostname: "my_host.com", expectErr: true},
		{name: "with special chars", hostname: "host@example.com", expectErr: true},
		{name: "path traversal", hostname: "../etc/passwd", expectErr: true},
		{name: "script injection", hostname: "<script>alert(1)</script>", expectErr: true},
		{name: "control characters", hostname: "host\x00.com", expectErr: true},
		{name: "non-ASCII", hostname: "тест.com", expectErr: true}, // Cyrillic characters
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateHostname(tt.hostname)

			if tt.expectErr && err == nil {
				t.Errorf("ValidateHostname(%q) expected error but got none", tt.hostname)
			}
			if !tt.expectErr && err != nil {
				t.Errorf("ValidateHostname(%q) unexpected error: %v", tt.hostname, err)
			}
		})
	}
}

// TestValidateComment tests comment validation
func TestValidateComment(t *testing.T) {
	tests := []struct {
		name      string
		comment   string
		expectErr bool
	}{
		// Valid comments
		{name: "empty comment", comment: "", expectErr: false},
		{name: "simple comment", comment: "This is a test server", expectErr: false},
		{name: "comment with numbers", comment: "Server #1", expectErr: false},
		{name: "comment with symbols", comment: "API server (production)", expectErr: false},
		{name: "comment with newlines", comment: "Line 1\nLine 2", expectErr: false},
		{name: "comment with tabs", comment: "Tabbed\tcomment", expectErr: false},
		{name: "max length", comment: strings.Repeat("a", 500), expectErr: false},

		// Invalid comments
		{name: "too long", comment: strings.Repeat("a", 501), expectErr: true},
		{name: "script tag", comment: "<script>alert(1)</script>", expectErr: true},
		{name: "javascript protocol", comment: "javascript:alert(1)", expectErr: true},
		{name: "data protocol", comment: "data:text/html,<script>alert(1)</script>", expectErr: true},
		{name: "vbscript", comment: "vbscript:msgbox(1)", expectErr: true},
		{name: "onload event", comment: "onload=alert(1)", expectErr: true},
		{name: "onerror event", comment: "onerror=alert(1)", expectErr: true},
		{name: "eval function", comment: "eval('alert(1)')", expectErr: true},
		{name: "setTimeout function", comment: "setTimeout(alert, 1000)", expectErr: true},
		{name: "setInterval function", comment: "setInterval(alert, 1000)", expectErr: true},
		{name: "control chars", comment: "test\x01comment", expectErr: true},
		{name: "null byte", comment: "test\x00comment", expectErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateComment(tt.comment)

			if tt.expectErr && err == nil {
				t.Errorf("ValidateComment(%q) expected error but got none", tt.comment)
			}
			if !tt.expectErr && err != nil {
				t.Errorf("ValidateComment(%q) unexpected error: %v", tt.comment, err)
			}
		})
	}
}

// TestValidateEntry tests complete entry validation
func TestValidateEntry(t *testing.T) {
	tests := []struct {
		name      string
		entry     Entry
		expectErr bool
	}{
		{
			name: "valid entry",
			entry: Entry{
				IP:        "127.0.0.1",
				Hostnames: []string{"localhost"},
				Comment:   "Local loopback",
				Category:  "default",
				Enabled:   true,
			},
			expectErr: false,
		},
		{
			name: "valid entry with multiple hostnames",
			entry: Entry{
				IP:        "192.168.1.100",
				Hostnames: []string{"api.dev", "web.dev"},
				Comment:   "Development servers",
				Category:  "development",
				Enabled:   true,
			},
			expectErr: false,
		},
		{
			name: "invalid IP",
			entry: Entry{
				IP:        "999.999.999.999",
				Hostnames: []string{"test.local"},
				Comment:   "",
				Category:  "default",
				Enabled:   true,
			},
			expectErr: true,
		},
		{
			name: "no hostnames",
			entry: Entry{
				IP:        "127.0.0.1",
				Hostnames: []string{},
				Comment:   "",
				Category:  "default",
				Enabled:   true,
			},
			expectErr: true,
		},
		{
			name: "invalid hostname",
			entry: Entry{
				IP:        "127.0.0.1",
				Hostnames: []string{"invalid..hostname"},
				Comment:   "",
				Category:  "default",
				Enabled:   true,
			},
			expectErr: true,
		},
		{
			name: "invalid comment",
			entry: Entry{
				IP:        "127.0.0.1",
				Hostnames: []string{"localhost"},
				Comment:   "<script>alert(1)</script>",
				Category:  "default",
				Enabled:   true,
			},
			expectErr: true,
		},
		{
			name: "invalid category",
			entry: Entry{
				IP:        "127.0.0.1",
				Hostnames: []string{"localhost"},
				Comment:   "",
				Category:  "invalid category name!",
				Enabled:   true,
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEntry(tt.entry)

			if tt.expectErr && err == nil {
				t.Errorf("ValidateEntry(%+v) expected error but got none", tt.entry)
			}
			if !tt.expectErr && err != nil {
				t.Errorf("ValidateEntry(%+v) unexpected error: %v", tt.entry, err)
			}
		})
	}
}

// TestValidateCategoryName tests category name validation
func TestValidateCategoryName(t *testing.T) {
	tests := []struct {
		name      string
		category  string
		expectErr bool
	}{
		// Valid category names
		{name: "simple", category: "development", expectErr: false},
		{name: "with numbers", category: "env1", expectErr: false},
		{name: "with hyphens", category: "my-env", expectErr: false},
		{name: "with underscores", category: "my_env", expectErr: false},
		{name: "mixed case", category: "MyEnv", expectErr: false},
		{name: "max length", category: strings.Repeat("a", 50), expectErr: false},

		// Invalid category names
		{name: "empty", category: "", expectErr: true},
		{name: "too long", category: strings.Repeat("a", 51), expectErr: true},
		{name: "with spaces", category: "my env", expectErr: true},
		{name: "with special chars", category: "env@prod", expectErr: true},
		{name: "with dots", category: "env.prod", expectErr: true},
		{name: "with slashes", category: "env/prod", expectErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCategoryName(tt.category)

			if tt.expectErr && err == nil {
				t.Errorf("validateCategoryName(%q) expected error but got none", tt.category)
			}
			if !tt.expectErr && err != nil {
				t.Errorf("validateCategoryName(%q) unexpected error: %v", tt.category, err)
			}
		})
	}
}

// TestValidateIPSecurity tests IP security validation
func TestValidateIPSecurity(t *testing.T) {
	tests := []struct {
		name      string
		ip        string
		expectErr bool
	}{
		// Should be allowed
		{name: "localhost", ip: "127.0.0.1", expectErr: false},
		{name: "private 192.168.x.x", ip: "192.168.1.1", expectErr: false},
		{name: "private 10.x.x.x", ip: "10.0.0.1", expectErr: false},
		{name: "private 172.16.x.x", ip: "172.16.0.1", expectErr: false},
		{name: "public IP", ip: "8.8.8.8", expectErr: false},
		{name: "IPv6 localhost", ip: "::1", expectErr: false},
		{name: "IPv6 private", ip: "fc00::1", expectErr: false},

		// Should be rejected
		{name: "multicast IPv4", ip: "224.0.0.1", expectErr: true},
		{name: "unspecified IPv4", ip: "0.0.0.0", expectErr: true},
		{name: "IPv6 multicast", ip: "ff02::1", expectErr: true},
		{name: "IPv6 unspecified", ip: "::", expectErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := parseIP(tt.ip)
			if ip == nil {
				t.Fatalf("Failed to parse IP: %s", tt.ip)
			}

			err := validateIPSecurity(ip)

			if tt.expectErr && err == nil {
				t.Errorf("validateIPSecurity(%q) expected error but got none", tt.ip)
			}
			if !tt.expectErr && err != nil {
				t.Errorf("validateIPSecurity(%q) unexpected error: %v", tt.ip, err)
			}
		})
	}
}

// TestIsPrivateIP tests private IP detection
func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		name    string
		ip      string
		private bool
	}{
		// Private IPs
		{name: "localhost", ip: "127.0.0.1", private: true},
		{name: "private 192.168.x.x", ip: "192.168.1.1", private: true},
		{name: "private 10.x.x.x", ip: "10.0.0.1", private: true},
		{name: "private 172.16.x.x", ip: "172.16.0.1", private: true},
		{name: "link-local", ip: "169.254.1.1", private: true},
		{name: "IPv6 localhost", ip: "::1", private: true},
		{name: "IPv6 private", ip: "fc00::1", private: true},
		{name: "IPv6 link-local", ip: "fe80::1", private: true},

		// Public IPs
		{name: "Google DNS", ip: "8.8.8.8", private: false},
		{name: "Cloudflare DNS", ip: "1.1.1.1", private: false},
		{name: "public IPv6", ip: "2001:4860:4860::8888", private: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := parseIP(tt.ip)
			if ip == nil {
				t.Fatalf("Failed to parse IP: %s", tt.ip)
			}

			result := isPrivateIP(ip)
			if result != tt.private {
				t.Errorf("isPrivateIP(%q) = %v, want %v", tt.ip, result, tt.private)
			}
		})
	}
}

// TestContainsHomographs tests homograph detection
func TestContainsHomographs(t *testing.T) {
	tests := []struct {
		name             string
		hostname         string
		expectHomographs bool
	}{
		{name: "normal ASCII", hostname: "example.com", expectHomographs: false},
		{name: "with numbers", hostname: "test123.com", expectHomographs: false},
		{name: "cyrillic a", hostname: "еxample.com", expectHomographs: true},  // Cyrillic 'e'
		{name: "cyrillic o", hostname: "examplе.com", expectHomographs: true},  // Cyrillic 'e'
		{name: "mixed script", hostname: "gооgle.com", expectHomographs: true}, // Cyrillic 'o'
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsHomographs(tt.hostname)
			if result != tt.expectHomographs {
				t.Errorf("containsHomographs(%q) = %v, want %v", tt.hostname, result, tt.expectHomographs)
			}
		})
	}
}

// TestValidateHostnameLabel tests individual hostname label validation
func TestValidateHostnameLabel(t *testing.T) {
	tests := []struct {
		name      string
		label     string
		expectErr bool
	}{
		// Valid labels
		{name: "simple label", label: "test", expectErr: false},
		{name: "with numbers", label: "test123", expectErr: false},
		{name: "with hyphens", label: "test-label", expectErr: false},
		{name: "single char", label: "a", expectErr: false},
		{name: "max length", label: strings.Repeat("a", 63), expectErr: false},

		// Invalid labels
		{name: "empty label", label: "", expectErr: true},
		{name: "too long", label: strings.Repeat("a", 64), expectErr: true},
		{name: "starts with hyphen", label: "-test", expectErr: true},
		{name: "ends with hyphen", label: "test-", expectErr: true},
		{name: "with underscore", label: "test_label", expectErr: true},
		{name: "with space", label: "test label", expectErr: true},
		{name: "with special char", label: "test@label", expectErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateHostnameLabel(tt.label)

			if tt.expectErr && err == nil {
				t.Errorf("validateHostnameLabel(%q) expected error but got none", tt.label)
			}
			if !tt.expectErr && err != nil {
				t.Errorf("validateHostnameLabel(%q) unexpected error: %v", tt.label, err)
			}
		})
	}
}

// TestValidateHostnameSecurity tests hostname security validation
func TestValidateHostnameSecurity(t *testing.T) {
	tests := []struct {
		name      string
		hostname  string
		expectErr bool
	}{
		// Safe hostnames
		{name: "normal hostname", hostname: "api.example.com", expectErr: false},
		{name: "with numbers", hostname: "host123.example.com", expectErr: false},
		{name: "with hyphens", hostname: "my-host.example.com", expectErr: false},

		// Dangerous patterns
		{name: "path traversal", hostname: "../etc/passwd", expectErr: true},
		{name: "script injection", hostname: "<script>alert(1)</script>", expectErr: true},
		{name: "html chars", hostname: "host<test>.com", expectErr: true},
		{name: "control chars", hostname: "host\x00.com", expectErr: true},
		{name: "non-ASCII", hostname: "тест.com", expectErr: true},
		{name: "homographs", hostname: "gооgle.com", expectErr: true}, // Cyrillic 'o'
		{name: "trailing dot", hostname: "example.com.", expectErr: true},
		{name: "only dots", hostname: "...", expectErr: true},
		{name: "whitespace", hostname: "host test.com", expectErr: true},
		{name: "long label", hostname: strings.Repeat("a", 64) + ".com", expectErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateHostnameSecurity(tt.hostname)

			if tt.expectErr && err == nil {
				t.Errorf("validateHostnameSecurity(%q) expected error but got none", tt.hostname)
			}
			if !tt.expectErr && err != nil {
				t.Errorf("validateHostnameSecurity(%q) unexpected error: %v", tt.hostname, err)
			}
		})
	}
}

// Helper function to parse IP addresses for testing
func parseIP(ipStr string) []byte {
	// Simple wrapper around net.ParseIP for testing
	if ipStr == "" {
		return nil
	}

	// Check if it's a valid IPv4
	parts := strings.Split(ipStr, ".")
	if len(parts) == 4 {
		ip := make([]byte, 4)
		for i, part := range parts {
			if len(part) == 0 || len(part) > 3 {
				return nil
			}
			val := 0
			for _, r := range part {
				if r < '0' || r > '9' {
					return nil
				}
				val = val*10 + int(r-'0')
				if val > 255 {
					return nil
				}
			}
			ip[i] = byte(val)
		}
		return ip
	}

	// For IPv6 and other complex cases, we'd need a more sophisticated parser
	// For now, return a simple mock based on the string
	switch ipStr {
	case "::1":
		return []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}
	case "::":
		return []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	case "ff02::1":
		ip := make([]byte, 16)
		ip[0] = 0xff
		ip[1] = 0x02
		ip[15] = 1
		return ip
	case "fc00::1":
		ip := make([]byte, 16)
		ip[0] = 0xfc
		ip[15] = 1
		return ip
	case "fe80::1":
		ip := make([]byte, 16)
		ip[0] = 0xfe
		ip[1] = 0x80
		ip[15] = 1
		return ip
	case "2001:4860:4860::8888":
		ip := make([]byte, 16)
		ip[0] = 0x20
		ip[1] = 0x01
		ip[2] = 0x48
		ip[3] = 0x60
		ip[4] = 0x48
		ip[5] = 0x60
		ip[14] = 0x88
		ip[15] = 0x88
		return ip
	default:
		return nil
	}
}

// Benchmark tests
func BenchmarkValidateIP(b *testing.B) {
	testIPs := []string{
		"127.0.0.1",
		"192.168.1.1",
		"2001:db8::1",
		"invalid.ip",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateIP(testIPs[i%len(testIPs)])
	}
}

func BenchmarkValidateHostname(b *testing.B) {
	testHostnames := []string{
		"localhost",
		"api.example.com",
		"very-long-hostname-with-many-parts.subdomain.example.com",
		"invalid..hostname",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateHostname(testHostnames[i%len(testHostnames)])
	}
}

func BenchmarkValidateEntry(b *testing.B) {
	testEntry := Entry{
		IP:        "192.168.1.100",
		Hostnames: []string{"api.dev", "web.dev"},
		Comment:   "Development servers",
		Category:  "development",
		Enabled:   true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateEntry(testEntry)
	}
}
