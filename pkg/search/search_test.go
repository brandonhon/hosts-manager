package search

import (
	"testing"

	"github.com/brandonhon/hosts-manager/internal/hosts"
)

func createTestHostsFile() *hosts.HostsFile {
	return &hosts.HostsFile{
		Categories: []hosts.Category{
			{
				Name:    "development",
				Enabled: true,
				Entries: []hosts.Entry{
					{
						IP:        "127.0.0.1",
						Hostnames: []string{"localhost", "local.dev"},
						Comment:   "Local development server",
						Category:  "development",
						Enabled:   true,
					},
					{
						IP:        "192.168.1.100",
						Hostnames: []string{"api.dev", "backend.local"},
						Comment:   "Development API server",
						Category:  "development",
						Enabled:   true,
					},
				},
			},
			{
				Name:    "production",
				Enabled: true,
				Entries: []hosts.Entry{
					{
						IP:        "203.0.113.1",
						Hostnames: []string{"api.example.com", "www.example.com"},
						Comment:   "Production web server",
						Category:  "production",
						Enabled:   true,
					},
					{
						IP:        "198.51.100.50",
						Hostnames: []string{"db.example.com"},
						Comment:   "Production database",
						Category:  "production",
						Enabled:   false,
					},
				},
			},
		},
	}
}

func TestNewSearcher(t *testing.T) {
	tests := []struct {
		name          string
		caseSensitive bool
		fuzzy         bool
	}{
		{
			name:          "case sensitive exact search",
			caseSensitive: true,
			fuzzy:         false,
		},
		{
			name:          "case insensitive exact search",
			caseSensitive: false,
			fuzzy:         false,
		},
		{
			name:          "case sensitive fuzzy search",
			caseSensitive: true,
			fuzzy:         true,
		},
		{
			name:          "case insensitive fuzzy search",
			caseSensitive: false,
			fuzzy:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			searcher := NewSearcher(tt.caseSensitive, tt.fuzzy)

			if searcher == nil {
				t.Fatal("NewSearcher returned nil")
			}

			if searcher.caseSensitive != tt.caseSensitive {
				t.Errorf("NewSearcher().caseSensitive = %v, want %v", searcher.caseSensitive, tt.caseSensitive)
			}

			if searcher.fuzzy != tt.fuzzy {
				t.Errorf("NewSearcher().fuzzy = %v, want %v", searcher.fuzzy, tt.fuzzy)
			}
		})
	}
}

func TestSearch(t *testing.T) {
	hostsFile := createTestHostsFile()

	tests := []struct {
		name          string
		caseSensitive bool
		fuzzy         bool
		query         string
		expectedCount int
		validate      func([]Result) bool
	}{
		{
			name:          "empty query returns no results",
			caseSensitive: false,
			fuzzy:         false,
			query:         "",
			expectedCount: 0,
			validate: func(results []Result) bool {
				return len(results) == 0
			},
		},
		{
			name:          "exact hostname match",
			caseSensitive: false,
			fuzzy:         false,
			query:         "localhost",
			expectedCount: 1,
			validate: func(results []Result) bool {
				return len(results) == 1 && results[0].Entry.Hostnames[0] == "localhost"
			},
		},
		{
			name:          "IP address search",
			caseSensitive: false,
			fuzzy:         false,
			query:         "127.0.0.1",
			expectedCount: 1,
			validate: func(results []Result) bool {
				return len(results) == 1 && results[0].Entry.IP == "127.0.0.1"
			},
		},
		{
			name:          "partial hostname match",
			caseSensitive: false,
			fuzzy:         false,
			query:         "dev",
			expectedCount: 2,
			validate: func(results []Result) bool {
				if len(results) < 2 {
					return false
				}
				// Should find api.dev and local.dev
				foundApiDev := false
				foundLocalDev := false
				for _, result := range results {
					for _, hostname := range result.Entry.Hostnames {
						if hostname == "api.dev" {
							foundApiDev = true
						}
						if hostname == "local.dev" {
							foundLocalDev = true
						}
					}
				}
				return foundApiDev && foundLocalDev
			},
		},
		{
			name:          "comment search",
			caseSensitive: false,
			fuzzy:         false,
			query:         "API",
			expectedCount: 1,
			validate: func(results []Result) bool {
				return len(results) >= 1 && results[0].Entry.Comment == "Development API server"
			},
		},
		{
			name:          "case sensitive search",
			caseSensitive: true,
			fuzzy:         false,
			query:         "API",
			expectedCount: 1,
			validate: func(results []Result) bool {
				return len(results) >= 1
			},
		},
		{
			name:          "fuzzy search with typo",
			caseSensitive: false,
			fuzzy:         true,
			query:         "localhst",
			expectedCount: 1,
			validate: func(results []Result) bool {
				return len(results) >= 1 && results[0].Score > 0.5
			},
		},
		{
			name:          "results sorted by score",
			caseSensitive: false,
			fuzzy:         true,
			query:         "example",
			expectedCount: 2,
			validate: func(results []Result) bool {
				if len(results) < 2 {
					return false
				}
				// Results should be sorted by score descending
				for i := 0; i < len(results)-1; i++ {
					if results[i].Score < results[i+1].Score {
						return false
					}
				}
				return true
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			searcher := NewSearcher(tt.caseSensitive, tt.fuzzy)
			results := searcher.Search(hostsFile, tt.query)

			if len(results) < tt.expectedCount {
				t.Errorf("Search() returned %d results, want at least %d", len(results), tt.expectedCount)
			}

			if !tt.validate(results) {
				t.Errorf("Search() validation failed for results: %+v", results)
			}
		})
	}
}

func TestSearchByCategory(t *testing.T) {
	hostsFile := createTestHostsFile()
	searcher := NewSearcher(false, false)

	tests := []struct {
		name     string
		query    string
		category string
		expected int
	}{
		{
			name:     "search in development category",
			query:    "dev",
			category: "development",
			expected: 2,
		},
		{
			name:     "search in production category",
			query:    "example",
			category: "production",
			expected: 2,
		},
		{
			name:     "search in non-existent category",
			query:    "localhost",
			category: "staging",
			expected: 0,
		},
		{
			name:     "empty query in category",
			query:    "",
			category: "development",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := searcher.SearchByCategory(hostsFile, tt.query, tt.category)

			if len(results) != tt.expected {
				t.Errorf("SearchByCategory() = %d results, want %d", len(results), tt.expected)
			}

			// Verify all results are from the specified category
			for _, result := range results {
				if result.Entry.Category != tt.category {
					t.Errorf("SearchByCategory() returned entry from wrong category: got %s, want %s", result.Entry.Category, tt.category)
				}
			}
		})
	}
}

func TestSearchByIP(t *testing.T) {
	hostsFile := createTestHostsFile()

	tests := []struct {
		name          string
		caseSensitive bool
		ip            string
		expected      int
		expectedIP    string
	}{
		{
			name:          "exact IP match",
			caseSensitive: false,
			ip:            "127.0.0.1",
			expected:      1,
			expectedIP:    "127.0.0.1",
		},
		{
			name:          "case insensitive IP match",
			caseSensitive: false,
			ip:            "127.0.0.1",
			expected:      1,
			expectedIP:    "127.0.0.1",
		},
		{
			name:          "no IP match",
			caseSensitive: false,
			ip:            "10.0.0.1",
			expected:      0,
			expectedIP:    "",
		},
		{
			name:          "production IP match",
			caseSensitive: false,
			ip:            "203.0.113.1",
			expected:      1,
			expectedIP:    "203.0.113.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			searcher := NewSearcher(tt.caseSensitive, false)
			results := searcher.SearchByIP(hostsFile, tt.ip)

			if len(results) != tt.expected {
				t.Errorf("SearchByIP() = %d results, want %d", len(results), tt.expected)
			}

			if tt.expected > 0 {
				if results[0].Entry.IP != tt.expectedIP {
					t.Errorf("SearchByIP() returned IP %s, want %s", results[0].Entry.IP, tt.expectedIP)
				}
				if results[0].Score != 1.0 {
					t.Errorf("SearchByIP() returned score %f, want 1.0", results[0].Score)
				}
				if results[0].Match != tt.expectedIP {
					t.Errorf("SearchByIP() returned match %s, want %s", results[0].Match, tt.expectedIP)
				}
			}
		})
	}
}

func TestSearchByHostname(t *testing.T) {
	hostsFile := createTestHostsFile()

	tests := []struct {
		name             string
		caseSensitive    bool
		fuzzy            bool
		hostname         string
		expectedCount    int
		expectedHostname string
	}{
		{
			name:             "exact hostname match",
			caseSensitive:    false,
			fuzzy:            false,
			hostname:         "localhost",
			expectedCount:    1,
			expectedHostname: "localhost",
		},
		{
			name:             "partial hostname match",
			caseSensitive:    false,
			fuzzy:            false,
			hostname:         "api",
			expectedCount:    2,         // Should match both "api.dev" and "api.example.com"
			expectedHostname: "api.dev", // First (highest score) result
		},
		{
			name:             "fuzzy hostname match",
			caseSensitive:    false,
			fuzzy:            true,
			hostname:         "localhst",
			expectedCount:    6, // Fuzzy matching is permissive, but localhost should be first
			expectedHostname: "localhost",
		},
		{
			name:             "case sensitive match",
			caseSensitive:    true,
			fuzzy:            false,
			hostname:         "LOCALHOST",
			expectedCount:    0,
			expectedHostname: "",
		},
		{
			name:             "no hostname match",
			caseSensitive:    false,
			fuzzy:            false,
			hostname:         "nonexistent",
			expectedCount:    0,
			expectedHostname: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			searcher := NewSearcher(tt.caseSensitive, tt.fuzzy)
			results := searcher.SearchByHostname(hostsFile, tt.hostname)

			if len(results) != tt.expectedCount {
				t.Errorf("SearchByHostname() = %d results, want %d", len(results), tt.expectedCount)
			}

			if tt.expectedCount > 0 {
				if results[0].Match != tt.expectedHostname {
					t.Errorf("SearchByHostname() returned match %s, want %s", results[0].Match, tt.expectedHostname)
				}
				if results[0].Score <= 0 {
					t.Errorf("SearchByHostname() returned score %f, want > 0", results[0].Score)
				}

				// Verify results are sorted by score
				for i := 0; i < len(results)-1; i++ {
					if results[i].Score < results[i+1].Score {
						t.Errorf("SearchByHostname() results not sorted by score")
					}
				}
			}
		})
	}
}

func TestExactMatch(t *testing.T) {
	searcher := NewSearcher(false, false)

	tests := []struct {
		name     string
		text     string
		query    string
		expected float64
	}{
		{
			name:     "exact match",
			text:     "localhost",
			query:    "localhost",
			expected: 1.0,
		},
		{
			name:     "prefix match",
			text:     "localhost.domain",
			query:    "localhost",
			expected: 0.9,
		},
		{
			name:     "contains match",
			text:     "my-localhost-server",
			query:    "localhost",
			expected: 0.7,
		},
		{
			name:     "word exact match",
			text:     "my localhost server",
			query:    "localhost",
			expected: 0.7, // Contains match takes precedence over word match
		},
		{
			name:     "word prefix match",
			text:     "localhost server",
			query:    "local",
			expected: 0.9, // Prefix match takes precedence over word prefix
		},
		{
			name:     "no match",
			text:     "example.com",
			query:    "localhost",
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := searcher.exactMatch(tt.text, tt.query)

			if score != tt.expected {
				t.Errorf("exactMatch() = %f, want %f", score, tt.expected)
			}
		})
	}
}

func TestFuzzyMatch(t *testing.T) {
	searcher := NewSearcher(false, true)

	tests := []struct {
		name     string
		text     string
		query    string
		validate func(float64) bool
	}{
		{
			name:  "exact match",
			text:  "localhost",
			query: "localhost",
			validate: func(score float64) bool {
				return score == 1.0
			},
		},
		{
			name:  "close match with typo",
			text:  "localhost",
			query: "localhst",
			validate: func(score float64) bool {
				return score > 0.7 && score < 1.0
			},
		},
		{
			name:  "contains match",
			text:  "my-localhost-server",
			query: "localhost",
			validate: func(score float64) bool {
				return score > 0.5
			},
		},
		{
			name:  "prefix match",
			text:  "localhost.domain",
			query: "localhost",
			validate: func(score float64) bool {
				return score > 0.7 && score < 1.0 // Fuzzy match averages similarity with prefix bonus
			},
		},
		{
			name:  "very different strings",
			text:  "example.com",
			query: "localhost",
			validate: func(score float64) bool {
				return score < 0.5
			},
		},
		{
			name:  "empty strings",
			text:  "",
			query: "",
			validate: func(score float64) bool {
				return score == 1.0
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := searcher.fuzzyMatch(tt.text, tt.query)

			if !tt.validate(score) {
				t.Errorf("fuzzyMatch() = %f, failed validation", score)
			}
		})
	}
}

func TestLevenshteinDistance(t *testing.T) {
	searcher := NewSearcher(false, true)

	tests := []struct {
		name     string
		a        string
		b        string
		expected int
	}{
		{
			name:     "identical strings",
			a:        "localhost",
			b:        "localhost",
			expected: 0,
		},
		{
			name:     "one character difference",
			a:        "localhost",
			b:        "localhst",
			expected: 1,
		},
		{
			name:     "completely different",
			a:        "abc",
			b:        "def",
			expected: 3,
		},
		{
			name:     "empty strings",
			a:        "",
			b:        "",
			expected: 0,
		},
		{
			name:     "one empty string",
			a:        "abc",
			b:        "",
			expected: 3,
		},
		{
			name:     "other empty string",
			a:        "",
			b:        "abc",
			expected: 3,
		},
		{
			name:     "insertion",
			a:        "abc",
			b:        "abcd",
			expected: 1,
		},
		{
			name:     "deletion",
			a:        "abcd",
			b:        "abc",
			expected: 1,
		},
		{
			name:     "substitution",
			a:        "abc",
			b:        "axc",
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			distance := searcher.levenshteinDistance(tt.a, tt.b)

			if distance != tt.expected {
				t.Errorf("levenshteinDistance() = %d, want %d", distance, tt.expected)
			}
		})
	}
}

func TestMin(t *testing.T) {
	tests := []struct {
		name     string
		a, b, c  int
		expected int
	}{
		{
			name:     "a is minimum",
			a:        1,
			b:        2,
			c:        3,
			expected: 1,
		},
		{
			name:     "b is minimum",
			a:        3,
			b:        1,
			c:        2,
			expected: 1,
		},
		{
			name:     "c is minimum",
			a:        3,
			b:        2,
			c:        1,
			expected: 1,
		},
		{
			name:     "all equal",
			a:        5,
			b:        5,
			c:        5,
			expected: 5,
		},
		{
			name:     "a and b equal, smaller than c",
			a:        1,
			b:        1,
			c:        2,
			expected: 1,
		},
		{
			name:     "b and c equal, smaller than a",
			a:        3,
			b:        1,
			c:        1,
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := min(tt.a, tt.b, tt.c)

			if result != tt.expected {
				t.Errorf("min(%d, %d, %d) = %d, want %d", tt.a, tt.b, tt.c, result, tt.expected)
			}
		})
	}
}

// Benchmark tests
func BenchmarkExactMatch(b *testing.B) {
	searcher := NewSearcher(false, false)
	text := "localhost.development.example.com"
	query := "localhost"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		searcher.exactMatch(text, query)
	}
}

func BenchmarkFuzzyMatch(b *testing.B) {
	searcher := NewSearcher(false, true)
	text := "localhost.development.example.com"
	query := "localhst"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		searcher.fuzzyMatch(text, query)
	}
}

func BenchmarkLevenshteinDistance(b *testing.B) {
	searcher := NewSearcher(false, true)
	a := "localhost.development.example.com"
	b_str := "localhst.development.example.com"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		searcher.levenshteinDistance(a, b_str)
	}
}

func BenchmarkSearch(b *testing.B) {
	hostsFile := createTestHostsFile()
	searcher := NewSearcher(false, true)
	query := "dev"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		searcher.Search(hostsFile, query)
	}
}

// Edge case tests
func TestSearchEdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		hostsFile *hosts.HostsFile
		query     string
		expected  int
	}{
		{
			name:      "nil hosts file",
			hostsFile: &hosts.HostsFile{},
			query:     "localhost",
			expected:  0,
		},
		{
			name: "empty categories",
			hostsFile: &hosts.HostsFile{
				Categories: []hosts.Category{},
			},
			query:    "localhost",
			expected: 0,
		},
		{
			name: "category with no entries",
			hostsFile: &hosts.HostsFile{
				Categories: []hosts.Category{
					{
						Name:    "empty",
						Entries: []hosts.Entry{},
					},
				},
			},
			query:    "localhost",
			expected: 0,
		},
		{
			name: "entry with no hostnames",
			hostsFile: &hosts.HostsFile{
				Categories: []hosts.Category{
					{
						Name: "test",
						Entries: []hosts.Entry{
							{
								IP:        "127.0.0.1",
								Hostnames: []string{},
								Comment:   "No hostnames",
							},
						},
					},
				},
			},
			query:    "127.0.0.1",
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			searcher := NewSearcher(false, false)
			results := searcher.Search(tt.hostsFile, tt.query)

			if len(results) != tt.expected {
				t.Errorf("Search() edge case = %d results, want %d", len(results), tt.expected)
			}
		})
	}
}
