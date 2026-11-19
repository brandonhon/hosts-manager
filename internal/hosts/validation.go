package hosts

import (
	"fmt"
	"net"
	"regexp"
	"strings"
	"unicode"
)

var (
	// hostnameRegex is RFC 1123: dot-separated labels of 1-63 ASCII letters,
	// digits and inner hyphens. That alone rules out empty labels, leading or
	// trailing hyphens and dots, whitespace, control and non-ASCII characters.
	hostnameRegex = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?)*$`)

	// categoryNameRegex matches a category name, which is written to the hosts
	// file as an @category marker and so must be a single token.
	categoryNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,50}$`)
)

// ValidateIP accepts any IPv4 or IPv6 address except multicast. Loopback,
// private and 0.0.0.0 / :: (the conventional way to block a host) are allowed.
func ValidateIP(ip string) error {
	if ip == "" {
		return fmt.Errorf("IP address cannot be empty")
	}
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return fmt.Errorf("invalid IP address format: %s", ip)
	}
	if parsed.IsMulticast() {
		return fmt.Errorf("multicast IP addresses not allowed: %s", ip)
	}
	return nil
}

// ValidateHostname checks hostname against RFC 1123.
func ValidateHostname(hostname string) error {
	if hostname == "" {
		return fmt.Errorf("hostname cannot be empty")
	}
	if len(hostname) > 253 {
		return fmt.Errorf("hostname too long (max 253 characters): %s", hostname)
	}
	if !hostnameRegex.MatchString(hostname) {
		return fmt.Errorf("invalid hostname format: %s", hostname)
	}
	return nil
}

// ValidateComment checks a comment that is appended to an entry's line.
func ValidateComment(comment string) error {
	if len(comment) > 500 {
		return fmt.Errorf("comment too long (max 500 characters)")
	}

	// A hosts file entry occupies exactly one line, and formatEntry appends the
	// comment to it verbatim. A line break in a comment therefore splits the
	// entry, and whatever follows the break is parsed back as a real host
	// mapping — so a comment can inject an arbitrary hostname. Tabs are fine;
	// they do not terminate the line.
	for _, r := range comment {
		if r == '\n' || r == '\r' {
			return fmt.Errorf("comment cannot contain line breaks")
		}
		if unicode.IsControl(r) && r != '\t' {
			return fmt.Errorf("comment contains control characters")
		}
	}

	return nil
}

// ValidateEntry validates every field of entry that is written to the file.
func ValidateEntry(entry Entry) error {
	if err := ValidateIP(entry.IP); err != nil {
		return fmt.Errorf("invalid IP address: %w", err)
	}

	if len(entry.Hostnames) == 0 {
		return fmt.Errorf("entry must have at least one hostname")
	}
	for _, hostname := range entry.Hostnames {
		if err := ValidateHostname(hostname); err != nil {
			return fmt.Errorf("invalid hostname: %w", err)
		}
	}

	if err := ValidateComment(entry.Comment); err != nil {
		return fmt.Errorf("invalid comment: %w", err)
	}

	// Notes are written back verbatim, one per line, above the entry. A note
	// carrying a line break or missing its '#' would emit a line the parser
	// reads as a host mapping — the same injection the comment check closes,
	// reachable through an import file that sets the field directly.
	for _, note := range entry.Notes {
		if strings.ContainsAny(note, "\n\r") {
			return fmt.Errorf("invalid note: cannot contain line breaks")
		}
		if !strings.HasPrefix(strings.TrimSpace(note), "#") {
			return fmt.Errorf("invalid note: must be a comment line beginning with '#'")
		}
	}

	if entry.Category != "" {
		if err := validateCategoryName(entry.Category); err != nil {
			return fmt.Errorf("invalid category: %w", err)
		}
	}

	return nil
}

func validateCategoryName(category string) error {
	if !categoryNameRegex.MatchString(category) {
		return fmt.Errorf("category name must be 1-50 characters of a-z, A-Z, 0-9, _ or -: %q", category)
	}
	return nil
}
