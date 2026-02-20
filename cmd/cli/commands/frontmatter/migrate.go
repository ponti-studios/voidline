package frontmatter

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	fm "voidline/internal/frontmatter"
)

func newMigrateCommand() *cobra.Command {
	options := commonOptions{}
	var strategyRaw string
	var write bool
	var backup bool

	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate markdown frontmatter according to schema strategy",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateOutput(options.output); err != nil {
				return err
			}

			cfg, err := resolveConfig(options.root, options.configPath)
			if err != nil {
				return err
			}
			schema, err := resolveSchema(cfg, options.schema)
			if err != nil {
				return err
			}

			strategy, err := parseStrategy(strategyRaw)
			if err != nil {
				return err
			}

			actions, err := fm.MigrateFiles(
				options.root,
				schema,
				strategy,
				walkOptionsFromCommon(options),
				write,
				backup,
				fm.NewGeneratorRegistry(),
				time.Now().UTC(),
			)
			if err != nil {
				return &commandError{code: exitRuntime, err: err}
			}

			resultActions, summary := convertActions(actions)
			summary.Command = "migrate"
			summary.Root = options.root

			if options.output == outputJSON {
				if err := renderJSON(cmd, actionResponse{Actions: resultActions, Summary: summary}); err != nil {
					return err
				}
			} else {
				title := fmt.Sprintf("Migration Results (strategy=%s, write=%t, backup=%t)", strategy, write, backup)
				if err := renderActionText(cmd, title, resultActions, summary); err != nil {
					return err
				}
			}

			if summary.ExitCode == exitDomain {
				return &commandError{code: exitDomain, err: fmt.Errorf("migration completed with validation errors")}
			}
			return nil
		},
	}

	addCommonWalkFlags(cmd, &options)
	addSchemaFlags(cmd, &options)
	cmd.Flags().StringVar(&strategyRaw, "strategy", "fill", "Migration strategy: fill|repair|overwrite|timestamps")
	cmd.Flags().BoolVar(&write, "write", false, "Write changes to files (default dry-run)")
	cmd.Flags().BoolVar(&backup, "backup", false, "Create backup files when writing changes")

	return cmd
}
