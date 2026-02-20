package frontmatter

import (
	"fmt"

	"github.com/spf13/cobra"

	fm "voidline/internal/frontmatter"
)

func newValidateCommand() *cobra.Command {
	options := commonOptions{}
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate markdown frontmatter against a schema",
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

			actions, err := fm.ValidateFiles(options.root, schema, walkOptionsFromCommon(options))
			if err != nil {
				return &commandError{code: exitRuntime, err: err}
			}

			resultActions, summary := convertActions(actions)
			summary.Command = "validate"
			summary.Root = options.root

			if options.output == outputJSON {
				if err := renderJSON(cmd, actionResponse{Actions: resultActions, Summary: summary}); err != nil {
					return err
				}
			} else {
				if err := renderActionText(cmd, "Validation Results", resultActions, summary); err != nil {
					return err
				}
			}

			if summary.ExitCode == exitDomain {
				return &commandError{code: exitDomain, err: fmt.Errorf("validation errors found")}
			}
			return nil
		},
	}

	addCommonWalkFlags(cmd, &options)
	addSchemaFlags(cmd, &options)
	return cmd
}
