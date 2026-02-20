package frontmatter

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	fm "voidline/internal/frontmatter"
)

func newSlugCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "slug", Short: "Detect and resolve slug collisions"}
	cmd.AddCommand(newSlugDetectCommand())
	cmd.AddCommand(newSlugResolveCommand())
	return cmd
}

func newSlugDetectCommand() *cobra.Command {
	options := commonOptions{}
	var scope string

	cmd := &cobra.Command{
		Use:   "detect",
		Short: "Detect slug collisions in markdown files",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateOutput(options.output); err != nil {
				return err
			}
			if scope == "" {
				scope = "directory"
			}

			collisions, err := fm.DetectSlugCollisions(options.root, scope, walkOptionsFromCommon(options))
			if err != nil {
				return &commandError{code: exitRuntime, err: err}
			}
			sort.Slice(collisions, func(i, j int) bool {
				if collisions[i].Slug == collisions[j].Slug {
					return collisions[i].Path < collisions[j].Path
				}
				return collisions[i].Slug < collisions[j].Slug
			})

			summary := commandSummary{Command: "slug detect", Root: options.root, Processed: len(collisions), TotalFiles: len(collisions), ExitCode: exitSuccess}
			if options.output == outputJSON {
				return renderJSON(cmd, slugDetectResponse{Collisions: collisions, Summary: summary})
			}
			return renderSlugDetectText(cmd, collisions, summary)
		},
	}

	addCommonWalkFlags(cmd, &options)
	cmd.Flags().StringVar(&scope, "scope", "directory", "Collision scope: directory|project|global")
	return cmd
}

func newSlugResolveCommand() *cobra.Command {
	var output string
	var slug string
	var policy string
	var maxAttempts int
	var existing []string

	cmd := &cobra.Command{
		Use:   "resolve",
		Short: "Resolve a slug collision based on policy",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateOutput(output); err != nil {
				return err
			}
			if strings.TrimSpace(slug) == "" {
				return &commandError{code: exitRuntime, err: fmt.Errorf("--slug is required")}
			}
			if maxAttempts < 1 {
				return &commandError{code: exitRuntime, err: fmt.Errorf("--max-attempts must be >= 1")}
			}

			existingMap := map[string]bool{}
			for _, value := range existing {
				if strings.TrimSpace(value) != "" {
					existingMap[value] = true
				}
			}

			resolved, err := fm.ResolveSlugCollision(slug, existingMap, policy, maxAttempts)
			if err != nil {
				if strings.Contains(err.Error(), "unknown slug collision policy") {
					return &commandError{code: exitRuntime, err: err}
				}
				var collisionErr *fm.SlugCollisionError
				if errors.As(err, &collisionErr) {
					return &commandError{code: exitDomain, err: err}
				}
				return &commandError{code: exitDomain, err: err}
			}

			summary := commandSummary{Command: "slug resolve", Processed: 1, TotalFiles: 1, ExitCode: exitSuccess}
			if output == outputJSON {
				return renderJSON(cmd, slugResolveResponse{Slug: slug, Resolved: resolved, Summary: summary})
			}
			return renderSlugResolveText(cmd, slug, resolved, summary)
		},
	}

	cmd.Flags().StringVar(&output, "output", outputText, "Output format: text|json")
	cmd.Flags().StringVar(&slug, "slug", "", "Slug to resolve")
	cmd.Flags().StringVar(&policy, "policy", "increment", "Collision policy: fail|increment|append-uid")
	cmd.Flags().IntVar(&maxAttempts, "max-attempts", 10, "Maximum attempts for increment policy")
	cmd.Flags().StringSliceVar(&existing, "existing-slugs", nil, "Existing slugs to check against")

	return cmd
}
