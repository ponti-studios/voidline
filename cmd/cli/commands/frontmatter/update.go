package frontmatter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	fm "voidline/internal/frontmatter"
)

const (
	openRouterBaseURL      = "https://openrouter.ai/api/v1"
	defaultOpenRouterModel = "minimax/minimax-m2"
)

type updateOptions struct {
	root       string
	apiKey     string
	model      string
	dryRun     bool
	maxFiles   int
	extensions []string
}

type aiFrontmatterRequest struct {
	Model       string  `json:"model"`
	Prompt      string  `json:"prompt"`
	Temperature float64 `json:"temperature,omitempty"`
}

type aiFrontmatterResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func newUpdateCommand() *cobra.Command {
	options := updateOptions{
		extensions: []string{".md", ".markdown"},
	}

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update essay frontmatter using AI (OpenRouter)",
		Long: `Analyzes markdown essays and updates their frontmatter metadata using AI.
The AI suggests appropriate values for type, tags, status, and description fields.

Requires OPENROUTER_API_KEY environment variable or --api-key flag.`,
		Example: `  voidline frontmatter update -d ./essays
  voidline frontmatter update -d ./essays --api-key sk-or-v1-...
  voidline frontmatter update -d ./essays --dry-run
  voidline frontmatter update -d ./essays --model anthropic/claude-3-haiku`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdate(cmd, options)
		},
	}

	cmd.Flags().StringVarP(&options.root, "dir", "d", ".", "Root directory containing essays")
	cmd.Flags().StringVar(&options.apiKey, "api-key", "", "OpenRouter API key (or set OPENROUTER_API_KEY)")
	cmd.Flags().StringVar(&options.model, "model", defaultOpenRouterModel, "OpenRouter model to use")
	cmd.Flags().BoolVar(&options.dryRun, "dry-run", false, "Show what would be updated without making changes")
	cmd.Flags().IntVar(&options.maxFiles, "max-files", 0, "Maximum files to process (0 = unlimited)")
	cmd.Flags().StringSliceVar(&options.extensions, "extensions", []string{".md", ".markdown"}, "File extensions to process")

	return cmd
}

func runUpdate(cmd *cobra.Command, options updateOptions) error {
	apiKey := options.apiKey
	if apiKey == "" {
		apiKey = os.Getenv("OPENROUTER_API_KEY")
	}
	if apiKey == "" {
		return &commandError{code: exitRuntime, err: fmt.Errorf("OpenRouter API key required: set --api-key flag or OPENROUTER_API_KEY environment variable")}
	}

	root := options.root
	if root == "" {
		root = "."
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return &commandError{code: exitRuntime, err: fmt.Errorf("invalid root path: %w", err)}
	}

	files, err := gatherMarkdownFiles(absRoot, options.extensions, options.maxFiles)
	if err != nil {
		return &commandError{code: exitRuntime, err: fmt.Errorf("failed to gather files: %w", err)}
	}

	if len(files) == 0 {
		fmt.Println("No markdown files found.")
		return nil
	}

	fmt.Printf("📁 Found %d markdown files in %s\n", len(files), absRoot)
	if options.dryRun {
		fmt.Println("🔍 Dry run mode - no changes will be made")
	}
	fmt.Println()

	httpClient := &http.Client{}
	updated := 0
	skipped := 0
	errors := 0

	for i, filePath := range files {
		relPath, _ := filepath.Rel(absRoot, filePath)
		fmt.Printf("[%d/%d] Processing: %s... ", i+1, len(files), relPath)

		result, err := processFileWithAI(httpClient, filePath, apiKey, options.model, options.dryRun)
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			errors++
			continue
		}

		if result.skipped {
			fmt.Println("⏭️  Skipped (empty or too short)")
			skipped++
		} else if result.noChange {
			fmt.Println("✅ Already up to date")
		} else if options.dryRun {
			fmt.Printf("🔄 Would update: type=%q, status=%q, tags=%v\n",
				result.frontmatter["type"], result.frontmatter["status"], result.frontmatter["tags"])
			updated++
		} else {
			fmt.Println("✅ Updated")
			updated++
		}
	}

	fmt.Println()
	fmt.Println("📊 Summary:")
	fmt.Printf("   Updated: %d\n", updated)
	fmt.Printf("   Skipped: %d\n", skipped)
	fmt.Printf("   Errors:  %d\n", errors)

	return nil
}

type processResult struct {
	frontmatter map[string]interface{}
	skipped     bool
	noChange    bool
}

func processFileWithAI(client *http.Client, filePath, apiKey, model string, dryRun bool) (processResult, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return processResult{}, fmt.Errorf("failed to read file: %w", err)
	}

	contentStr := string(content)
	parsed, err := fm.ParseYAMLFrontmatter(contentStr)
	if err != nil {
		return processResult{}, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	body := parsed.Body
	if len(body) < 100 {
		return processResult{skipped: true}, nil
	}

	existingFM := parsed.Frontmatter
	suggested, err := queryAIForFrontmatter(client, body, apiKey, model)
	if err != nil {
		return processResult{}, fmt.Errorf("AI query failed: %w", err)
	}

	newFM := mergeFrontmatter(existingFM, suggested)

	if !hasChanges(existingFM, newFM) {
		return processResult{noChange: true}, nil
	}

	if dryRun {
		return processResult{frontmatter: newFM}, nil
	}

	newContent, err := fm.BuildYAMLFrontmatter(newFM, body)
	if err != nil {
		return processResult{}, fmt.Errorf("failed to build frontmatter: %w", err)
	}

	if err := os.WriteFile(filePath, []byte(newContent), 0644); err != nil {
		return processResult{}, fmt.Errorf("failed to write file: %w", err)
	}

	return processResult{frontmatter: newFM}, nil
}

func queryAIForFrontmatter(client *http.Client, body, apiKey, model string) (map[string]interface{}, error) {
	prompt := fmt.Sprintf(`Analyze this essay and suggest appropriate frontmatter metadata.
Consider: type (essay, note, draft), tags (technology, philosophy, business, etc.),
status (published, draft, idea), and description.

Respond ONLY with valid JSON frontmatter object with these fields:
- type: "essay" or "note" or "draft"
- tags: array of relevant topic tags (kebab-case, e.g., ["technology", "ai", "software-engineering"])
- status: "published" or "draft" or "idea"
- description: brief 1-2 sentence description

Example response:
{"type": "essay", "tags": ["technology", "ai"], "status": "published", "description": "..."}

Essay content:
%s`, truncateString(body, 12000))

	reqBody := aiFrontmatterRequest{
		Model:       model,
		Prompt:      prompt,
		Temperature: 0.3,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, openRouterBaseURL+"/chat/completions", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("HTTP-Referer", "https://voidline.dev")
	req.Header.Set("X-Title", "Voidline Frontmatter Updater")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp aiFrontmatterResponse
		if decodeErr := json.NewDecoder(resp.Body).Decode(&errResp); decodeErr == nil && errResp.Error != nil {
			return nil, fmt.Errorf("OpenRouter error: %s", errResp.Error.Message)
		}
		return nil, fmt.Errorf("OpenRouter returned status %d", resp.StatusCode)
	}

	var parsed aiFrontmatterResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if parsed.Error != nil && parsed.Error.Message != "" {
		return nil, fmt.Errorf("OpenRouter error: %s", parsed.Error.Message)
	}

	if len(parsed.Choices) == 0 || parsed.Choices[0].Message.Content == "" {
		return nil, fmt.Errorf("empty response from OpenRouter")
	}

	return parseAIResponse(parsed.Choices[0].Message.Content)
}

func parseAIResponse(content string) (map[string]interface{}, error) {
	content = strings.TrimSpace(content)

	var jsonStart int
	for i := 0; i < len(content); i++ {
		if content[i] == '{' {
			jsonStart = i
			break
		}
	}
	if jsonStart > 0 {
		content = content[jsonStart:]
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(content), &result); err == nil {
		return result, nil
	}

	var end int
	depth := 0
	for i, ch := range content {
		if ch == '{' {
			depth++
		} else if ch == '}' {
			depth--
			if depth == 0 {
				end = i + 1
				break
			}
		}
	}
	if end > 0 {
		if err := json.Unmarshal([]byte(content[:end]), &result); err == nil {
			return result, nil
		}
	}

	return parseFrontmatterFromText(content)
}

func parseFrontmatterFromText(content string) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		colonIdx := strings.Index(line, ":")
		if colonIdx == -1 {
			continue
		}

		key := strings.TrimSpace(line[:colonIdx])
		valueStr := strings.TrimSpace(line[colonIdx+1:])
		if key == "" || valueStr == "" {
			continue
		}

		valueStr = strings.Trim(valueStr, "\"' ")

		if strings.HasPrefix(valueStr, "[") && strings.HasSuffix(valueStr, "]") {
			inner := strings.Trim(valueStr, "[]")
			if inner != "" {
				parts := strings.Split(inner, ",")
				var tags []string
				for _, p := range parts {
					tag := strings.Trim(strings.TrimSpace(p), "\"' ")
					if tag != "" {
						tags = append(tags, tag)
					}
				}
				result[key] = tags
			} else {
				result[key] = []string{}
			}
		} else {
			result[key] = valueStr
		}
	}

	return result, nil
}

func mergeFrontmatter(existing, suggested map[string]interface{}) map[string]interface{} {
	merged := make(map[string]interface{})

	for k, v := range existing {
		merged[k] = v
	}

	if v, ok := suggested["type"].(string); ok && v != "" {
		merged["type"] = v
	} else if existing["type"] == nil {
		merged["type"] = "essay"
	}

	if v, ok := suggested["tags"]; ok {
		if tags, ok := v.([]interface{}); ok {
			var stringTags []string
			for _, t := range tags {
				if s, ok := t.(string); ok {
					stringTags = append(stringTags, s)
				}
			}
			if len(stringTags) > 0 {
				merged["tags"] = stringTags
			}
		} else if v != nil {
			merged["tags"] = v
		}
	}
	if _, ok := merged["tags"]; !ok {
		merged["tags"] = []string{}
	}

	if v, ok := suggested["status"].(string); ok && v != "" {
		merged["status"] = v
	} else if existing["status"] == nil {
		merged["status"] = "draft"
	}

	if v, ok := suggested["description"].(string); ok && v != "" {
		merged["description"] = v
	}

	return merged
}

func hasChanges(existing, new map[string]interface{}) bool {
	fields := []string{"type", "status", "description", "tags"}

	for _, field := range fields {
		oldVal := existing[field]
		newVal := new[field]

		oldStr := fmt.Sprintf("%v", oldVal)
		newStr := fmt.Sprintf("%v", newVal)

		if oldStr != newStr {
			return true
		}
	}

	return false
}

func gatherMarkdownFiles(root string, extensions []string, maxFiles int) ([]string, error) {
	var files []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			return nil
		}

		ext := filepath.Ext(path)
		validExt := false
		for _, e := range extensions {
			if ext == e {
				validExt = true
				break
			}
		}
		if !validExt {
			return nil
		}

		files = append(files, path)

		if maxFiles > 0 && len(files) >= maxFiles {
			return filepath.SkipAll
		}

		return nil
	})

	if err != nil && err != filepath.SkipAll {
		return nil, err
	}

	return files, nil
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
