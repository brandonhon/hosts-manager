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
		// The standard blocking idiom, and what the shipped "blocked" category is for.
		{name: "unspecified IPv4 0.0.0.0", ip: "0.0.0.0", expectErr: false},
		{name: "unspecified IPv6 ::", ip: "::", expectErr: false},
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
		{name: "comment with tabs", comment: "Tabbed\tcomment", expectErr: false},
		{name: "max length", comment: strings.Repeat("a", 500), expectErr: false},
		// A comment is inert text in a hosts file. The old blocklist of
		// script patterns ("data:", "eval(", ...) protected nothing and
		// rejected ordinary comments like this one.
		{name: "looks like a url scheme", comment: "metadata: see wiki", expectErr: false},
		{name: "html-ish text", comment: "<script> host for tests", expectErr: false},

		// Invalid comments
		{name: "too long", comment: strings.Repeat("a", 501), expectErr: true},
		// A line break would split the entry it is appended to, and the text
		// after the break parses back as a real host mapping.
		{name: "line feed", comment: "Line 1\nLine 2", expectErr: true},
		{name: "carriage return", comment: "Line 1\rLine 2", expectErr: true},
		{name: "crlf", comment: "Line 1\r\nLine 2", expectErr: true},
		{name: "injected host mapping", comment: "note\n10.0.0.1 evil.example.com", expectErr: true},
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
				Comment:   "note\n10.0.0.1 evil.example.com",
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

// TestCommentCannotInjectHostMapping is the regression test for the injection
// itself rather than for the validator in isolation. formatEntry appends a
// comment to the entry line verbatim, so a line break used to split the entry
// and the remainder was parsed back as a genuine host mapping.
func TestCommentCannotInjectHostMapping(t *testing.T) {
	entry := Entry{
		IP:        "127.0.0.1",
		Hostnames: []string{"safe.local"},
		Comment:   "note\n10.0.0.1 evil.example.com",
		Category:  "custom",
		Enabled:   true,
	}

	if err := ValidateEntry(entry); err == nil {
		t.Fatal("ValidateEntry accepted a comment containing a line break")
	}

	// Belt and braces: if the entry ever reached the formatter anyway, show
	// exactly what the damage would be, so this test documents the mechanism.
	line := formatEntry(entry)
	if strings.Count(line, "\n") > 0 {
		t.Logf("formatEntry would emit %d lines from one entry:\n%s", strings.Count(line, "\n")+1, line)
	}
}
