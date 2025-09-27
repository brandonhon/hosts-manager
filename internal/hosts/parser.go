package hosts

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"
)

var (
	commentLineRegex = regexp.MustCompile(`^\s*#(.*)$`)
	entryLineRegex   = regexp.MustCompile(`^\s*([0-9a-fA-F:.]+)\s+([^\s#]+(?:\s+[^\s#]+)*)\s*(?:#(.*))?$`)
	categoryRegex    = regexp.MustCompile(`^\s*#\s*@category\s+(\w+)(?:\s+(.*))?$`)
	sectionRegex     = regexp.MustCompile(`^\s*#\s*===+\s*(.*?)\s*===+\s*$`)
)

type Parser struct {
	filePath string
}

func NewParser(filePath string) *Parser {
	return &Parser{filePath: filePath}
}

func (p *Parser) Parse() (*HostsFile, error) {
	file, err := os.Open(p.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open hosts file: %w", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file stats: %w", err)
	}

	hostsFile := &HostsFile{
		Categories: []Category{},
		Header:     []string{},
		Footer:     []string{},
		Modified:   stat.ModTime(),
		FilePath:   p.filePath,
	}

	scanner := bufio.NewScanner(file)
	lineNum := 0
	currentCategory := CategoryDefault
	var categories = make(map[string]*Category)
	var headerDone bool

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		originalLine := line

		if matches := categoryRegex.FindStringSubmatch(line); matches != nil {
			currentCategory = matches[1]
			if _, exists := categories[currentCategory]; !exists {
				categories[currentCategory] = &Category{
					Name:    currentCategory,
					Enabled: true,
					Entries: []Entry{},
				}
				if len(matches) > 2 && matches[2] != "" {
					categories[currentCategory].Description = matches[2]
				}
			}
			headerDone = true
			continue
		}

		if sectionRegex.MatchString(line) {
			headerDone = true
			continue
		}

		if entry, isEntry := p.parseEntry(line, lineNum); isEntry {
			headerDone = true
			entry.Category = currentCategory

			if _, exists := categories[currentCategory]; !exists {
				categories[currentCategory] = &Category{
					Name:    currentCategory,
					Enabled: true,
					Entries: []Entry{},
				}
			}
			categories[currentCategory].Entries = append(categories[currentCategory].Entries, entry)
		} else if commentLineRegex.MatchString(line) || strings.TrimSpace(line) == "" {
			if !headerDone {
				hostsFile.Header = append(hostsFile.Header, originalLine)
			}
		} else if strings.TrimSpace(line) != "" {
			if !headerDone {
				hostsFile.Header = append(hostsFile.Header, originalLine)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	for _, category := range categories {
		hostsFile.Categories = append(hostsFile.Categories, *category)
	}

	if len(hostsFile.Categories) == 0 {
		hostsFile.Categories = append(hostsFile.Categories, Category{
			Name:    CategoryDefault,
			Enabled: true,
			Entries: []Entry{},
		})
	}

	return hostsFile, nil
}

func (p *Parser) parseEntry(line string, lineNum int) (Entry, bool) {
	line = strings.TrimSpace(line)

	if line == "" || strings.HasPrefix(line, "#") {
		if strings.HasPrefix(line, "#") {
			uncommented := strings.TrimSpace(strings.TrimPrefix(line, "#"))
			if matches := entryLineRegex.FindStringSubmatch(uncommented); matches != nil {
				ip := matches[1]
				hostnames := strings.Fields(matches[2])
				comment := ""
				if len(matches) > 3 {
					comment = strings.TrimSpace(matches[3])
				}

				if p.isValidIP(ip) && len(hostnames) > 0 {
					return Entry{
						IP:        ip,
						Hostnames: hostnames,
						Comment:   comment,
						Enabled:   false,
						LineNum:   lineNum,
					}, true
				}
			}
		}
		return Entry{}, false
	}

	matches := entryLineRegex.FindStringSubmatch(line)
	if matches == nil {
		return Entry{}, false
	}

	ip := matches[1]
	hostnames := strings.Fields(matches[2])
	comment := ""
	if len(matches) > 3 {
		comment = strings.TrimSpace(matches[3])
	}

	if !p.isValidIP(ip) || len(hostnames) == 0 {
		return Entry{}, false
	}

	return Entry{
		IP:        ip,
		Hostnames: hostnames,
		Comment:   comment,
		Enabled:   true,
		LineNum:   lineNum,
	}, true
}

func (p *Parser) isValidIP(ip string) bool {
	return ValidateIP(ip) == nil
}

func (hf *HostsFile) Write(filePath string) error {
	return AtomicWrite(filePath, func(file io.Writer) error {
		writer := bufio.NewWriter(file)
		defer writer.Flush()

		// Write managed file header
		managedHeader := []string{
			"# This file is currently managed by hosts-manager",
			"# See https://github.com/brandonhon/hosts-manager for usage",
			"",
		}

		for _, line := range managedHeader {
			if _, err := writer.WriteString(line + "\n"); err != nil {
				return fmt.Errorf("failed to write managed header: %w", err)
			}
		}

		// Write original header (if any) but skip managed headers and compress blank lines
		var headerLines []string
		var lastLineWasBlank bool

		for _, headerLine := range hf.Header {
			// Skip our managed headers
			if strings.Contains(headerLine, "managed by hosts-manager") ||
				strings.Contains(headerLine, "github.com/brandonhon/hosts-manager") {
				continue
			}

			// Compress multiple blank lines into single blank line
			if strings.TrimSpace(headerLine) == "" {
				if !lastLineWasBlank {
					headerLines = append(headerLines, headerLine)
					lastLineWasBlank = true
				}
			} else {
				headerLines = append(headerLines, headerLine)
				lastLineWasBlank = false
			}
		}

		// Remove trailing blank lines from header
		for len(headerLines) > 0 && strings.TrimSpace(headerLines[len(headerLines)-1]) == "" {
			headerLines = headerLines[:len(headerLines)-1]
		}

		// Write the cleaned header lines
		for _, headerLine := range headerLines {
			if _, err := writer.WriteString(headerLine + "\n"); err != nil {
				return fmt.Errorf("failed to write header: %w", err)
			}
		}

		// Add single separator line if we have original header content
		if len(headerLines) > 0 {
			if _, err := writer.WriteString("\n"); err != nil {
				return err
			}
		}

		// Write categories with cleaner spacing
		for i, category := range hf.Categories {
			if len(category.Entries) == 0 {
				continue
			}

			// Add separator between categories (but not before first)
			if i > 0 {
				if _, err := writer.WriteString("\n"); err != nil {
					return fmt.Errorf("failed to write category separator: %w", err)
				}
			}

			categoryHeader := fmt.Sprintf("# @category %s", category.Name)
			if category.Description != "" {
				categoryHeader += " " + category.Description
			}
			if _, err := writer.WriteString(categoryHeader + "\n"); err != nil {
				return fmt.Errorf("failed to write category header: %w", err)
			}

			sectionHeader := fmt.Sprintf("# =============== %s ===============", strings.ToUpper(category.Name))
			if _, err := writer.WriteString(sectionHeader + "\n"); err != nil {
				return fmt.Errorf("failed to write section header: %w", err)
			}

			for _, entry := range category.Entries {
				line := formatEntry(entry)
				if _, err := writer.WriteString(line + "\n"); err != nil {
					return fmt.Errorf("failed to write entry: %w", err)
				}
			}
		}

		// Write footer with spacing if needed
		if len(hf.Footer) > 0 {
			if _, err := writer.WriteString("\n"); err != nil {
				return err
			}
			for _, footerLine := range hf.Footer {
				if _, err := writer.WriteString(footerLine + "\n"); err != nil {
					return fmt.Errorf("failed to write footer: %w", err)
				}
			}
		}

		hf.Modified = time.Now()
		return nil
	})
}

func formatEntry(entry Entry) string {
	line := fmt.Sprintf("%s %s", entry.IP, strings.Join(entry.Hostnames, " "))

	if entry.Comment != "" {
		line += " # " + entry.Comment
	}

	if !entry.Enabled {
		line = "# " + line
	}

	return line
}

func (hf *HostsFile) AddEntry(entry Entry) error {
	// Validate the entry before adding
	if err := ValidateEntry(entry); err != nil {
		return fmt.Errorf("entry validation failed: %w", err)
	}

	categoryName := entry.Category
	if categoryName == "" {
		categoryName = CategoryDefault
		entry.Category = categoryName
	}

	for i := range hf.Categories {
		if hf.Categories[i].Name == categoryName {
			hf.Categories[i].Entries = append(hf.Categories[i].Entries, entry)
			return nil
		}
	}

	hf.Categories = append(hf.Categories, Category{
		Name:    categoryName,
		Enabled: true,
		Entries: []Entry{entry},
	})

	return nil
}

func (hf *HostsFile) RemoveEntry(hostname string) bool {
	for i := range hf.Categories {
		for j := len(hf.Categories[i].Entries) - 1; j >= 0; j-- {
			entry := &hf.Categories[i].Entries[j]
			for k, h := range entry.Hostnames {
				if h == hostname {
					if len(entry.Hostnames) == 1 {
						hf.Categories[i].Entries = append(hf.Categories[i].Entries[:j], hf.Categories[i].Entries[j+1:]...)
					} else {
						entry.Hostnames = append(entry.Hostnames[:k], entry.Hostnames[k+1:]...)
					}
					return true
				}
			}
		}
	}
	return false
}

func (hf *HostsFile) EnableEntry(hostname string) bool {
	for i := range hf.Categories {
		for j := range hf.Categories[i].Entries {
			entry := &hf.Categories[i].Entries[j]
			for _, h := range entry.Hostnames {
				if h == hostname {
					entry.Enabled = true
					return true
				}
			}
		}
	}
	return false
}

func (hf *HostsFile) DisableEntry(hostname string) bool {
	for i := range hf.Categories {
		for j := range hf.Categories[i].Entries {
			entry := &hf.Categories[i].Entries[j]
			for _, h := range entry.Hostnames {
				if h == hostname {
					entry.Enabled = false
					return true
				}
			}
		}
	}
	return false
}

func (hf *HostsFile) FindEntries(query string) []Entry {
	var results []Entry
	query = strings.ToLower(query)

	for _, category := range hf.Categories {
		for _, entry := range category.Entries {
			for _, hostname := range entry.Hostnames {
				if strings.Contains(strings.ToLower(hostname), query) {
					results = append(results, entry)
					break
				}
			}
			if strings.Contains(strings.ToLower(entry.IP), query) {
				results = append(results, entry)
			}
		}
	}

	return results
}

func (hf *HostsFile) GetCategory(name string) *Category {
	for i := range hf.Categories {
		if hf.Categories[i].Name == name {
			return &hf.Categories[i]
		}
	}
	return nil
}

func (hf *HostsFile) EnableCategory(name string) {
	if category := hf.GetCategory(name); category != nil {
		category.Enabled = true
		for i := range category.Entries {
			category.Entries[i].Enabled = true
		}
	}
}

func (hf *HostsFile) DisableCategory(name string) {
	if category := hf.GetCategory(name); category != nil {
		category.Enabled = false
		for i := range category.Entries {
			category.Entries[i].Enabled = false
		}
	}
}
