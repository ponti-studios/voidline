package frontmatter

import (
	"sort"

	"github.com/spf13/cobra"

	fm "voidline/internal/frontmatter"
)

func newWalkCommand() *cobra.Command {
	options := commonOptions{}
	cmd := &cobra.Command{
		Use:   "walk",
		Short: "List markdown files under a root",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateOutput(options.output); err != nil {
				return err
			}

			files, err := fm.WalkMarkdownFiles(options.root, walkOptionsFromCommon(options))
			if err != nil {
				return &commandError{code: exitRuntime, err: err}
			}
			sort.Strings(files)

			summary := commandSummary{
				Command:    "walk",
				Root:       options.root,
				TotalFiles: len(files),
				Processed:  len(files),
				ExitCode:   exitSuccess,
			}

			if options.output == outputJSON {
				return renderJSON(cmd, walkResponse{Files: files, Summary: summary})
			}
			return renderWalkText(cmd, files, summary)
		},
	}

	addCommonWalkFlags(cmd, &options)
	return cmd
}
