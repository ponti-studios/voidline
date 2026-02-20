package health

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"voidline/internal/application/health"
	"voidline/internal/infrastructure/config"
	"voidline/internal/infrastructure/persistence/sqlite"
)

type ImportCommand struct {
	DBPath    string
	SourceDir string
	DryRun    bool
	Force     bool
	Source    string
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

	repo := sqlite.NewHealthRepository(conn.DB())
	service := health.NewService(repo)

	options := health.ImportOptions{
		DryRun: c.DryRun,
		Force:  c.Force,
	}

	switch c.Source {
	case "withings":
		_, err := service.ImportWithingsActivities(ctx, c.SourceDir, options)
		if err != nil {
			return fmt.Errorf("import failed: %w", err)
		}
	case "spo2":
		_, err := service.ImportSpO2(ctx, c.SourceDir, options)
		if err != nil {
			return fmt.Errorf("import failed: %w", err)
		}
	case "mfp":
		_, err := service.ImportMFPWeight(ctx, c.SourceDir, options)
		if err != nil {
			return fmt.Errorf("import failed: %w", err)
		}
	case "weight":
		_, err := service.ImportWeight(ctx, c.SourceDir, options)
		if err != nil {
			return fmt.Errorf("import failed: %w", err)
		}
	case "sleep":
		_, err := service.ImportSleep(ctx, c.SourceDir, options)
		if err != nil {
			return fmt.Errorf("import failed: %w", err)
		}
	case "bp", "blood-pressure":
		_, err := service.ImportBloodPressure(ctx, c.SourceDir, options)
		if err != nil {
			return fmt.Errorf("import failed: %w", err)
		}
	case "hr", "heart-rate":
		_, err := service.ImportHeartRate(ctx, c.SourceDir, options)
		if err != nil {
			return fmt.Errorf("import failed: %w", err)
		}
	case "all":
		_, err := service.ImportWithingsActivities(ctx, c.SourceDir, options)
		if err != nil {
			return fmt.Errorf("import withings failed: %w", err)
		}
		_, err = service.ImportSpO2(ctx, c.SourceDir, options)
		if err != nil {
			return fmt.Errorf("import spo2 failed: %w", err)
		}
		_, err = service.ImportMFPWeight(ctx, c.SourceDir, options)
		if err != nil {
			return fmt.Errorf("import mfp failed: %w", err)
		}
		_, err = service.ImportWeight(ctx, c.SourceDir, options)
		if err != nil {
			return fmt.Errorf("import weight failed: %w", err)
		}
		_, err = service.ImportSleep(ctx, c.SourceDir, options)
		if err != nil {
			return fmt.Errorf("import sleep failed: %w", err)
		}
		_, err = service.ImportBloodPressure(ctx, c.SourceDir, options)
		if err != nil {
			return fmt.Errorf("import blood pressure failed: %w", err)
		}
		_, err = service.ImportHeartRate(ctx, c.SourceDir, options)
		if err != nil {
			return fmt.Errorf("import heart rate failed: %w", err)
		}
		fmt.Println("Health data migration complete.")
	default:
		return fmt.Errorf("unknown source: %s (use: withings, spo2, mfp, weight, sleep, bp, hr, or all)", c.Source)
	}

	return nil
}

func Command() *cobra.Command {
	var dbPath, sourceDir, sourceType string
	var dryRun, force bool

	cmd := &cobra.Command{
		Use:   "health",
		Short: "Import health data",
		RunE: func(cmd *cobra.Command, args []string) error {
			return (&ImportCommand{
				DBPath:    dbPath,
				SourceDir: sourceDir,
				DryRun:    dryRun,
				Force:     force,
				Source:    sourceType,
			}).Execute(cmd.Context())
		},
	}

	cmd.Flags().StringVar(&dbPath, "db", "", "Path to SQLite database")
	cmd.Flags().StringVar(&sourceDir, "source", "", "Source directory containing health data files")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Validate without importing")
	cmd.Flags().BoolVar(&force, "force", false, "Skip duplicate checking")
	cmd.Flags().StringVar(&sourceType, "source-type", "all", "Source type: withings, spo2, mfp, weight, sleep, bp, hr, or all")

	return cmd
}
