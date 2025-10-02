package search

import (
	"sort"
	"strings"

	"github.com/brandonhon/hosts-manager/internal/hosts"
)

type Result struct {
	Entry hosts.Entry `json:"entry"`
	Score float64     `json:"score"`
	Match string      `json:"match"`
}

type Searcher struct {
	caseSensitive bool
	fuzzy         bool
}

func NewSearcher(caseSensitive, fuzzy bool) *Searcher {
	return &Searcher{
		caseSensitive: caseSensitive,
		fuzzy:         fuzzy,
	}
}

func (s *Searcher) Search(hostsFile *hosts.HostsFile, query string) []Result {
	if query == "" {
		return []Result{}
	}

	var results []Result

	for _, category := range hostsFile.Categories {
		for _, entry := range category.Entries {
			if score, match := s.scoreEntry(entry, query); score > 0 {
				results = append(results, Result{
					Entry: entry,
					Score: score,
					Match: match,
				})
			}
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results
}

func (s *Searcher) scoreEntry(entry hosts.Entry, query string) (float64, string) {
	if !s.caseSensitive {
		query = strings.ToLower(query)
	}

	maxScore := 0.0
	bestMatch := ""

	for _, hostname := range entry.Hostnames {
		searchText := hostname
		if !s.caseSensitive {
			searchText = strings.ToLower(hostname)
		}

		var score float64
		if s.fuzzy {
			score = s.fuzzyMatch(searchText, query)
		} else {
			score = s.exactMatch(searchText, query)
		}

		if score > maxScore {
			maxScore = score
			bestMatch = hostname
		}
	}

	ipSearchText := entry.IP
	if !s.caseSensitive {
		ipSearchText = strings.ToLower(entry.IP)
	}

	var ipScore float64
	if s.fuzzy {
		ipScore = s.fuzzyMatch(ipSearchText, query)
	} else {
		ipScore = s.exactMatch(ipSearchText, query)
	}

	if ipScore > maxScore {
		maxScore = ipScore
		bestMatch = entry.IP
	}

	if entry.Comment != "" {
		commentSearchText := entry.Comment
		if !s.caseSensitive {
			commentSearchText = strings.ToLower(entry.Comment)
		}

		var commentScore float64
		if s.fuzzy {
			commentScore = s.fuzzyMatch(commentSearchText, query) * 0.5
		} else {
			commentScore = s.exactMatch(commentSearchText, query) * 0.5
		}

		if commentScore > maxScore {
			maxScore = commentScore
			bestMatch = entry.Comment
		}
	}

	return maxScore, bestMatch
}

func (s *Searcher) exactMatch(text, query string) float64 {
	if text == query {
		return 1.0
	}

	if strings.HasPrefix(text, query) {
		return 0.9
	}

	if strings.Contains(text, query) {
		return 0.7
	}

	words := strings.Fields(text)
	for _, word := range words {
		if word == query {
			return 0.8
		}
		if strings.HasPrefix(word, query) {
			return 0.6
		}
	}

	return 0.0
}

func (s *Searcher) fuzzyMatch(text, query string) float64 {
	if text == query {
		return 1.0
	}

	distance := s.levenshteinDistance(text, query)
	maxLen := len(text)
	if len(query) > maxLen {
		maxLen = len(query)
	}

	if maxLen == 0 {
		return 1.0
	}

	similarity := 1.0 - float64(distance)/float64(maxLen)

	if strings.Contains(text, query) {
		similarity = (similarity + 0.8) / 2
	}

	if strings.HasPrefix(text, query) {
		similarity = (similarity + 0.9) / 2
	}

	return similarity
}

func (s *Searcher) levenshteinDistance(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	matrix := make([][]int, len(a)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(b)+1)
		matrix[i][0] = i
	}

	for j := 1; j <= len(b); j++ {
		matrix[0][j] = j
	}

	for i := 1; i <= len(a); i++ {
		for j := 1; j <= len(b); j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}

			matrix[i][j] = min(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len(a)][len(b)]
}

func min(a, b, c int) int {
	if a < b && a < c {
		return a
	}
	if b < c {
		return b
	}
	return c
}

func (s *Searcher) SearchByCategory(hostsFile *hosts.HostsFile, query, category string) []Result {
	results := s.Search(hostsFile, query)

	var filtered []Result
	for _, result := range results {
		if result.Entry.Category == category {
			filtered = append(filtered, result)
		}
	}

	return filtered
}

func (s *Searcher) SearchByIP(hostsFile *hosts.HostsFile, ip string) []Result {
	var results []Result

	for _, category := range hostsFile.Categories {
		for _, entry := range category.Entries {
			entryIP := entry.IP
			if !s.caseSensitive {
				entryIP = strings.ToLower(entry.IP)
				ip = strings.ToLower(ip)
			}

			if entryIP == ip {
				results = append(results, Result{
					Entry: entry,
					Score: 1.0,
					Match: entry.IP,
				})
			}
		}
	}

	return results
}

func (s *Searcher) SearchByHostname(hostsFile *hosts.HostsFile, hostname string) []Result {
	var results []Result

	for _, category := range hostsFile.Categories {
		for _, entry := range category.Entries {
			for _, h := range entry.Hostnames {
				searchText := h
				queryText := hostname
				if !s.caseSensitive {
					searchText = strings.ToLower(h)
					queryText = strings.ToLower(hostname)
				}

				var score float64
				if s.fuzzy {
					score = s.fuzzyMatch(searchText, queryText)
				} else {
					score = s.exactMatch(searchText, queryText)
				}

				if score > 0 {
					results = append(results, Result{
						Entry: entry,
						Score: score,
						Match: h,
					})
				}
			}
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results
}
