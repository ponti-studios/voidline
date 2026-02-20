package music

import (
	"context"
	"fmt"

	"voidline/internal/application/music"
	"voidline/internal/infrastructure/config"
	"voidline/internal/infrastructure/persistence/sqlite"
)

type AppleCommand struct {
	DBPath    string
	SourceDir string
	DryRun    bool
	Force     bool
}

func (c *AppleCommand) Execute(ctx context.Context) error {
	if c.DBPath == "" {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		c.DBPath = cfg.Database.Path
	}

	if c.SourceDir == "" {
		return fmt.Errorf("source directory is required")
	}

	conn, err := sqlite.NewConnection(c.DBPath)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer conn.Close()

	repo := sqlite.NewMusicRepository(conn.DB())
	service := music.NewService(repo)

	options := music.ImportOptions{
		DryRun: c.DryRun,
		Force:  c.Force,
	}

	_, err = service.ImportAppleMusic(ctx, c.SourceDir, options)
	if err != nil {
		return fmt.Errorf("import failed: %w", err)
	}

	return nil
}

func HandleAppleMusicImport(ctx context.Context, dbPath, sourceDir string, dryRun, force bool) error {
	cmd := AppleCommand{
		DBPath:    dbPath,
		SourceDir: sourceDir,
		DryRun:    dryRun,
		Force:     force,
	}

	return cmd.Execute(ctx)
}
