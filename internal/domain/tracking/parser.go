package tracking

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"voidline/internal/domain/shared"
	"gopkg.in/yaml.v2"
)

// MarkdownParser handles parsing of tracking markdown files
type MarkdownParser struct {
	DefaultType string
}

// NewMarkdownParser creates a new parser with default settings
func NewMarkdownParser() *MarkdownParser {
	return &MarkdownParser{
		DefaultType: "general",
	}
}

// ParsedFile represents a parsed markdown file
type ParsedFile struct {
	Frontmatter map[string]interface{}
	Content     string
	FilePath    string
}

// ParseFile parses a markdown file into a TrackingEntry
func (mp *MarkdownParser) ParseFile(filePath, content string) (*TrackingEntry, error) {
	parsed, err := mp.parseMarkdown(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse markdown: %w", err)
	}

	parsed.FilePath = filePath

	// Create tracking entry from parsed data
	entry, err := mp.convertToTrackingEntry(parsed)
	if err != nil {
		return nil, fmt.Errorf("failed to convert to tracking entry: %w", err)
	}

	return entry, nil
}

// parseMarkdown separates YAML frontmatter from markdown content
func (mp *MarkdownParser) parseMarkdown(content string) (*ParsedFile, error) {
	lines := strings.Split(content, "\n")

	var frontmatterLines []string
	var contentLines []string
	var inFrontmatter bool
	var frontmatterEnded bool

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check for frontmatter start/end
		if trimmed == "---" {
			if i == 0 {
				inFrontmatter = true
				continue
			} else if inFrontmatter && !frontmatterEnded {
				frontmatterEnded = true
				inFrontmatter = false
				continue
			}
		}

		if inFrontmatter {
			frontmatterLines = append(frontmatterLines, line)
		} else if frontmatterEnded || i > 0 {
			contentLines = append(contentLines, line)
		}
	}

	// Parse YAML frontmatter
	var frontmatter map[string]interface{}
	if len(frontmatterLines) > 0 {
		yamlContent := strings.Join(frontmatterLines, "\n")
		if err := yaml.Unmarshal([]byte(yamlContent), &frontmatter); err != nil {
			return nil, fmt.Errorf("failed to parse YAML frontmatter: %w", err)
		}
	}

	if frontmatter == nil {
		frontmatter = make(map[string]interface{})
	}

	return &ParsedFile{
		Frontmatter: frontmatter,
		Content:     strings.Join(contentLines, "\n"),
	}, nil
}

// convertToTrackingEntry converts parsed file data into a TrackingEntry
func (mp *MarkdownParser) convertToTrackingEntry(parsed *ParsedFile) (*TrackingEntry, error) {
	// Extract basic information
	title := mp.getStringFromFrontmatter(parsed.Frontmatter, "title")
	if title == "" {
		title = mp.extractTitleFromContent(parsed.Content)
	}
	if title == "" {
		title = "Tracking Entry"
	}

	entryType := mp.getStringFromFrontmatter(parsed.Frontmatter, "category")
	if entryType == "" {
		entryType = mp.getStringFromFrontmatter(parsed.Frontmatter, "type")
	}
	if entryType == "" {
		entryType = mp.DefaultType
	}

	// Extract date
	var entryDate shared.Timestamp
	dateStr := mp.getStringFromFrontmatter(parsed.Frontmatter, "date")
	if dateStr != "" {
		if parsedTime, err := mp.parseFlexibleDate(dateStr); err == nil {
			entryDate = shared.Timestamp(parsedTime)
		} else {
			entryDate = shared.Now()
		}
	} else {
		entryDate = shared.Now()
	}

	// Create the entry
	entry := NewTrackingEntry(entryType, title, parsed.Content, entryDate)
	entry.Metadata = parsed.Frontmatter
	entry.SourceFile = parsed.FilePath

	// Extract data points from content
	dataPoints, err := mp.extractDataPoints(parsed.Content, entry.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to extract data points: %w", err)
	}

	entry.DataPoints = dataPoints

	return &entry, nil
}

// extractDataPoints parses the markdown content for structured data points
func (mp *MarkdownParser) extractDataPoints(content string, entryID shared.ID) ([]TrackingDataPoint, error) {
	var dataPoints []TrackingDataPoint

	lines := strings.Split(content, "\n")

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Try different patterns to extract data points
		points := []TrackingDataPoint{}

		// Pattern 1: Date with value (e.g., "2025-11-03, 8 units (0.2mg)")
		if dateValuePoint := mp.parseDateValueLine(trimmed, entryID); dateValuePoint != nil {
			points = append(points, *dateValuePoint)
		}

		// Pattern 2: Date with description (e.g., "2022-05-06, Griffith Observatory to Hollywood Sign")
		if dateDescPoint := mp.parseDateDescriptionLine(trimmed, entryID); dateDescPoint != nil {
			points = append(points, *dateDescPoint)
		}

		// Pattern 3: Checkbox items with dates (e.g., "- [x] 2024-10-20, Temescal Canyon Trail")
		if checkboxPoint := mp.parseCheckboxLine(trimmed, entryID); checkboxPoint != nil {
			points = append(points, *checkboxPoint)
		}

		// Pattern 4: Percentage tracking (e.g., "2026-01-14: 74%")
		if percentPoint := mp.parsePercentageLine(trimmed, entryID); percentPoint != nil {
			points = append(points, *percentPoint)
		}

		// Pattern 5: Simple bullet points with dates (e.g., "- 2022-05-06, Griffith Observatory")
		if bulletPoint := mp.parseBulletDateLine(trimmed, entryID); bulletPoint != nil {
			points = append(points, *bulletPoint)
		}

		dataPoints = append(dataPoints, points...)
	}

	// Sort data points by date
	sort.Slice(dataPoints, func(i, j int) bool {
		return dataPoints[i].Date.Time().Before(dataPoints[j].Date.Time())
	})

	return dataPoints, nil
}

// parseDateValueLine parses lines like "2025-11-03, 8 units (0.2mg)"
func (mp *MarkdownParser) parseDateValueLine(line string, entryID shared.ID) *TrackingDataPoint {
	// Pattern: date, value (unit)
	re := regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}),\s*(.+)$`)
	matches := re.FindStringSubmatch(line)

	if len(matches) != 3 {
		return nil
	}

	date, err := time.Parse("2006-01-02", matches[1])
	if err != nil {
		return nil
	}

	valueStr := strings.TrimSpace(matches[2])

	// Extract value and unit if in format "X units (Y)"
	unitRe := regexp.MustCompile(`^(.+?)\s*\((.+?)\)$`)
	unitMatches := unitRe.FindStringSubmatch(valueStr)

	var value, unit string
	if len(unitMatches) == 3 {
		value = strings.TrimSpace(unitMatches[1])
		unit = strings.TrimSpace(unitMatches[2])
	} else {
		value = valueStr
	}

	point := NewTrackingDataPoint(entryID, "measurement", value, unit, shared.Timestamp(date))
	return &point
}

// parseDateDescriptionLine parses lines like "2022-05-06, Griffith Observatory to Hollywood Sign"
func (mp *MarkdownParser) parseDateDescriptionLine(line string, entryID shared.ID) *TrackingDataPoint {
	// Pattern: date, description
	re := regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}),\s*(.+)$`)
	matches := re.FindStringSubmatch(line)

	if len(matches) != 3 {
		return nil
	}

	date, err := time.Parse("2006-01-02", matches[1])
	if err != nil {
		return nil
	}

	description := strings.TrimSpace(matches[2])

	point := NewTrackingDataPoint(entryID, "event", description, "", shared.Timestamp(date))
	return &point
}

// parseCheckboxLine parses checkbox lines like "- [x] 2024-10-20, Temescal Canyon Trail"
func (mp *MarkdownParser) parseCheckboxLine(line string, entryID shared.ID) *TrackingDataPoint {
	// Pattern: - [x] or - [ ] followed by date and description
	re := regexp.MustCompile(`^-\s*\[([x ])\]\s*(\d{4}-\d{2}-\d{2}),\s*(.+)$`)
	matches := re.FindStringSubmatch(line)

	if len(matches) != 4 {
		return nil
	}

	completed := strings.TrimSpace(matches[1]) == "x"

	date, err := time.Parse("2006-01-02", matches[2])
	if err != nil {
		return nil
	}

	description := strings.TrimSpace(matches[3])
	status := "planned"
	if completed {
		status = "completed"
	}

	point := NewTrackingDataPoint(entryID, "goal", description, status, shared.Timestamp(date))
	return &point
}

// parsePercentageLine parses lines like "2026-01-14: 74%"
func (mp *MarkdownParser) parsePercentageLine(line string, entryID shared.ID) *TrackingDataPoint {
	// Pattern: date: percentage%
	re := regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}):\s*(\d+(?:\.\d+)?)%`)
	matches := re.FindStringSubmatch(line)

	if len(matches) != 3 {
		return nil
	}

	date, err := time.Parse("2006-01-02", matches[1])
	if err != nil {
		return nil
	}

	percentage := matches[2]

	point := NewTrackingDataPoint(entryID, "percentage", percentage, "%", shared.Timestamp(date))
	return &point
}

// parseBulletDateLine parses simple bullet points with dates (e.g., "- 2022-05-06, Griffith Observatory")
func (mp *MarkdownParser) parseBulletDateLine(line string, entryID shared.ID) *TrackingDataPoint {
	// Pattern: - date, description
	re := regexp.MustCompile(`^-\s*(\d{4}-\d{2}-\d{2}),\s*(.+)$`)
	matches := re.FindStringSubmatch(line)

	if len(matches) != 3 {
		return nil
	}

	date, err := time.Parse("2006-01-02", matches[1])
	if err != nil {
		return nil
	}

	description := strings.TrimSpace(matches[2])

	point := NewTrackingDataPoint(entryID, "activity", description, "", shared.Timestamp(date))
	return &point
}

// Helper functions

func (mp *MarkdownParser) getStringFromFrontmatter(frontmatter map[string]interface{}, key string) string {
	if val, exists := frontmatter[key]; exists {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func (mp *MarkdownParser) extractTitleFromContent(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, "#"))
		}
	}
	return ""
}

func (mp *MarkdownParser) parseFlexibleDate(dateStr string) (time.Time, error) {
	// Try common date formats
	formats := []string{
		"2006-01-02",
		"2006-01",
		"2006-02-01",
		"January 2, 2006",
		"Jan 2, 2006",
		"2006/01/02",
		"01/02/2006",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
}

// ParseMultipleFiles parses multiple markdown files and returns tracking entries
func (mp *MarkdownParser) ParseMultipleFiles(files map[string]string) ([]TrackingEntry, error) {
	var entries []TrackingEntry
	var errors []string

	for filePath, content := range files {
		entry, err := mp.ParseFile(filePath, content)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", filePath, err))
			continue
		}
		entries = append(entries, *entry)
	}

	if len(errors) > 0 {
		return entries, fmt.Errorf("parsing errors: %s", strings.Join(errors, "; "))
	}

	return entries, nil
}

// ExtractSummaryStats provides summary statistics from parsed entries
func (mp *MarkdownParser) ExtractSummaryStats(entries []TrackingEntry) map[string]interface{} {
	stats := make(map[string]interface{})

	typeCount := make(map[string]int)
	totalDataPoints := 0
	dateRange := struct {
		earliest *time.Time
		latest   *time.Time
	}{}

	for _, entry := range entries {
		typeCount[entry.Type]++
		totalDataPoints += len(entry.DataPoints)

		entryTime := entry.Date.Time()
		if dateRange.earliest == nil || entryTime.Before(*dateRange.earliest) {
			dateRange.earliest = &entryTime
		}
		if dateRange.latest == nil || entryTime.After(*dateRange.latest) {
			dateRange.latest = &entryTime
		}

		for _, point := range entry.DataPoints {
			pointTime := point.Date.Time()
			if pointTime.Before(*dateRange.earliest) {
				dateRange.earliest = &pointTime
			}
			if pointTime.After(*dateRange.latest) {
				dateRange.latest = &pointTime
			}
		}
	}

	stats["total_entries"] = len(entries)
	stats["total_data_points"] = totalDataPoints
	stats["types"] = typeCount
	if dateRange.earliest != nil {
		stats["date_range"] = map[string]string{
			"earliest": dateRange.earliest.Format("2006-01-02"),
			"latest":   dateRange.latest.Format("2006-01-02"),
		}
	}

	return stats
}
