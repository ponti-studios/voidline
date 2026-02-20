package apple

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"voidline/internal/application/apple"
	"voidline/internal/infrastructure/config"
	"voidline/internal/infrastructure/persistence/sqlite"
)

type ImportCommand struct {
	DBPath    string
	SourceDir string
	DryRun    bool
	Force     bool
}

func (c *ImportCommand) Execute(ctx context.Context) error {
	if c.DBPath == "" {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		c.DBPath = cfg.Database.Path
	}

	if c.SourceDir == "" {
		fmt.Fprintln(os.Stderr, "Error: Source directory is required")
		return fmt.Errorf("source directory is required")
	}

	conn, err := sqlite.NewConnection(c.DBPath)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer conn.Close()

	repo := sqlite.NewAppleRepository(conn.DB())
	service := apple.NewService(repo)

	options := apple.ImportOptions{
		DryRun: c.DryRun,
		Force:  c.Force,
	}

	_, err = service.ImportAll(ctx, c.SourceDir, options)
	if err != nil {
		return fmt.Errorf("import failed: %w", err)
	}

	return nil
}

func Command() *cobra.Command {
	var dbPath, sourceDir string
	var dryRun, force bool

	cmd := &cobra.Command{
		Use:   "apple",
		Short: "Import Apple receipts",
		RunE: func(cmd *cobra.Command, args []string) error {
			return (&ImportCommand{
				DBPath:    dbPath,
				SourceDir: sourceDir,
				DryRun:    dryRun,
				Force:     force,
			}).Execute(cmd.Context())
		},
	}

	cmd.Flags().StringVar(&dbPath, "db", "", "Path to SQLite database")
	cmd.Flags().StringVar(&sourceDir, "source", "", "Source directory containing Apple data")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Validate without importing")
	cmd.Flags().BoolVar(&force, "force", false, "Skip duplicate checking")

	return cmd
}
