package frontmatter

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	fm "voidline/internal/frontmatter"
)

const (
	outputText = "text"
	outputJSON = "json"
)

const (
	exitSuccess = 0
	exitDomain  = 1
	exitRuntime = 2
)

type commandError struct {
	code int
	err  error
}

func (e *commandError) Error() string {
	if e.err == nil {
		return "command failed"
	}
	return e.err.Error()
}

func (e *commandError) Unwrap() error {
	return e.err
}

func (e *commandError) ExitCode() int {
	return e.code
}

type commonOptions struct {
	root          string
	schema        string
	configPath    string
	output        string
	includeHidden bool
	extensions    []string
	includeGlobs  []string
	excludeGlobs  []string
	maxFiles      int
}

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "frontmatter",
		Short:         "Validate and migrate markdown frontmatter",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.AddCommand(newWalkCommand())
	cmd.AddCommand(newValidateCommand())
	cmd.AddCommand(newMigrateCommand())
	cmd.AddCommand(newSlugCommand())

	return cmd
}

func addCommonWalkFlags(cmd *cobra.Command, options *commonOptions) {
	cmd.Flags().StringVar(&options.root, "root", ".", "Root directory to scan")
	cmd.Flags().StringVar(&options.output, "output", outputText, "Output format: text|json")
	cmd.Flags().BoolVar(&options.includeHidden, "include-hidden", false, "Include hidden files and directories")
	cmd.Flags().StringSliceVar(&options.extensions, "extensions", []string{".md", ".markdown"}, "Allowed file extensions")
	cmd.Flags().StringSliceVar(&options.includeGlobs, "include-globs", nil, "Include glob patterns (relative to root)")
	cmd.Flags().StringSliceVar(&options.excludeGlobs, "exclude-globs", nil, "Exclude glob patterns (relative to root)")
	cmd.Flags().IntVar(&options.maxFiles, "max-files", 0, "Maximum files to process (0 = unlimited)")
}

func addSchemaFlags(cmd *cobra.Command, options *commonOptions) {
	cmd.Flags().StringVar(&options.schema, "schema", "", "Schema name to use")
	cmd.Flags().StringVar(&options.configPath, "config", "", "Explicit frontmatter config path")
}

func validateOutput(output string) error {
	switch output {
	case outputText, outputJSON:
		return nil
	default:
		return &commandError{code: exitRuntime, err: fmt.Errorf("invalid output format %q (expected text|json)", output)}
	}
}

func walkOptionsFromCommon(options commonOptions) fm.WalkOptions {
	return fm.WalkOptions{
		IncludeHidden: options.includeHidden,
		Extensions:    options.extensions,
		IncludeGlobs:  options.includeGlobs,
		ExcludeGlobs:  options.excludeGlobs,
		MaxFiles:      options.maxFiles,
	}
}

func resolveConfig(root, configPath string) (fm.Config, error) {
	cfg, err := fm.LoadConfigWithOptions(root, configPath, false)
	if err != nil {
		return fm.Config{}, &commandError{code: exitRuntime, err: fmt.Errorf("load config: %w", err)}
	}
	return cfg, nil
}

func resolveSchema(cfg fm.Config, schemaName string) (fm.SchemaDefinition, error) {
	if len(cfg.Frontmatter.Schemas) == 0 {
		return fm.SchemaDefinition{}, &commandError{code: exitRuntime, err: fmt.Errorf("no schemas configured")}
	}

	if schemaName != "" {
		schema, ok := cfg.Frontmatter.Schemas[schemaName]
		if !ok {
			return fm.SchemaDefinition{}, &commandError{code: exitRuntime, err: fmt.Errorf("unknown schema %q", schemaName)}
		}
		return schema, nil
	}

	if schema, ok := cfg.Frontmatter.Schemas["personal"]; ok {
		return schema, nil
	}

	names := make([]string, 0, len(cfg.Frontmatter.Schemas))
	for name := range cfg.Frontmatter.Schemas {
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) == 0 {
		return fm.SchemaDefinition{}, &commandError{code: exitRuntime, err: fmt.Errorf("no schemas configured")}
	}
	return cfg.Frontmatter.Schemas[names[0]], nil
}

func parseStrategy(raw string) (fm.MigrationStrategy, error) {
	value := strings.TrimSpace(strings.ToLower(raw))
	if value == "" {
		return fm.StrategyFill, nil
	}

	switch fm.MigrationStrategy(value) {
	case fm.StrategyFill, fm.StrategyRepair, fm.StrategyOverwrite, fm.StrategyTimestamps:
		return fm.MigrationStrategy(value), nil
	default:
		return "", &commandError{code: exitRuntime, err: fmt.Errorf("invalid strategy %q", raw)}
	}
}
