package frontmatter

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	fm "voidline/internal/frontmatter"
)

type commandSummary struct {
	Command    string   `json:"command"`
	Root       string   `json:"root,omitempty"`
	TotalFiles int      `json:"totalFiles,omitempty"`
	Processed  int      `json:"processedFiles"`
	Changed    int      `json:"changedFiles,omitempty"`
	ErrorFiles int      `json:"errorFiles"`
	Warnings   []string `json:"warnings,omitempty"`
	DurationMs int64    `json:"durationMs,omitempty"`
	ExitCode   int      `json:"exitCode"`
}

type fileActionResult struct {
	Path       string      `json:"path"`
	HasChanges bool        `json:"hasChanges"`
	Errors     []string    `json:"errors"`
	Result     interface{} `json:"result,omitempty"`
}

type walkResponse struct {
	Files   []string       `json:"files"`
	Summary commandSummary `json:"summary"`
}

type actionResponse struct {
	Actions []fileActionResult `json:"actions"`
	Summary commandSummary     `json:"summary"`
}

type slugDetectResponse struct {
	Collisions []fm.SlugCollisionResult `json:"collisions"`
	Summary    commandSummary           `json:"summary"`
}

type slugResolveResponse struct {
	Slug     string         `json:"slug"`
	Resolved string         `json:"resolvedSlug"`
	Summary  commandSummary `json:"summary"`
}

func convertActions(actions []fm.FileAction) ([]fileActionResult, commandSummary) {
	sorted := make([]fm.FileAction, len(actions))
	copy(sorted, actions)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Path < sorted[j].Path
	})

	results := make([]fileActionResult, 0, len(sorted))
	summary := commandSummary{Processed: len(sorted), TotalFiles: len(sorted), ExitCode: exitSuccess}

	for _, action := range sorted {
		entry := fileActionResult{Path: action.Path, HasChanges: action.HasChanges}
		if action.Result.HasFrontmatter || action.Result.HasChanges || len(action.Result.Changes) > 0 {
			entry.Result = action.Result
		}
		if len(action.Errors) > 0 {
			summary.ErrorFiles++
			errs := make([]string, 0, len(action.Errors))
			for _, err := range action.Errors {
				errs = append(errs, err.Error())
			}
			sort.Strings(errs)
			entry.Errors = errs
		}
		if action.HasChanges {
			summary.Changed++
		}
		results = append(results, entry)
	}

	if summary.ErrorFiles > 0 {
		summary.ExitCode = exitDomain
	}

	return results, summary
}

func renderJSON(cmd *cobra.Command, data interface{}) error {
	payload, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return &commandError{code: exitRuntime, err: fmt.Errorf("encode json: %w", err)}
	}
	if _, err := fmt.Fprintln(cmd.OutOrStdout(), string(payload)); err != nil {
		return &commandError{code: exitRuntime, err: fmt.Errorf("write output: %w", err)}
	}
	return nil
}

func renderActionText(cmd *cobra.Command, title string, actions []fileActionResult, summary commandSummary) error {
	if _, err := fmt.Fprintln(cmd.OutOrStdout(), title); err != nil {
		return &commandError{code: exitRuntime, err: fmt.Errorf("write output: %w", err)}
	}

	for _, action := range actions {
		status := "ok"
		if len(action.Errors) > 0 {
			status = "error"
		} else if action.HasChanges {
			status = "changed"
		}
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "- [%s] %s\n", status, action.Path); err != nil {
			return &commandError{code: exitRuntime, err: fmt.Errorf("write output: %w", err)}
		}
		for _, message := range action.Errors {
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", message); err != nil {
				return &commandError{code: exitRuntime, err: fmt.Errorf("write output: %w", err)}
			}
		}
	}

	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Summary: processed=%d changed=%d errorFiles=%d exitCode=%d\n", summary.Processed, summary.Changed, summary.ErrorFiles, summary.ExitCode); err != nil {
		return &commandError{code: exitRuntime, err: fmt.Errorf("write output: %w", err)}
	}
	return nil
}

func renderWalkText(cmd *cobra.Command, files []string, summary commandSummary) error {
	sorted := append([]string(nil), files...)
	sort.Strings(sorted)
	for _, path := range sorted {
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), path); err != nil {
			return &commandError{code: exitRuntime, err: fmt.Errorf("write output: %w", err)}
		}
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Summary: files=%d exitCode=%d\n", summary.TotalFiles, summary.ExitCode); err != nil {
		return &commandError{code: exitRuntime, err: fmt.Errorf("write output: %w", err)}
	}
	return nil
}

func renderSlugDetectText(cmd *cobra.Command, collisions []fm.SlugCollisionResult, summary commandSummary) error {
	for _, collision := range collisions {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s [%s] -> %s\n", collision.Path, collision.Slug, strings.Join(collision.Collisions, ", ")); err != nil {
			return &commandError{code: exitRuntime, err: fmt.Errorf("write output: %w", err)}
		}
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Summary: collisions=%d exitCode=%d\n", len(collisions), summary.ExitCode); err != nil {
		return &commandError{code: exitRuntime, err: fmt.Errorf("write output: %w", err)}
	}
	return nil
}

func renderSlugResolveText(cmd *cobra.Command, slug, resolved string, summary commandSummary) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s -> %s\nSummary: exitCode=%d\n", slug, resolved, summary.ExitCode); err != nil {
		return &commandError{code: exitRuntime, err: fmt.Errorf("write output: %w", err)}
	}
	return nil
}
