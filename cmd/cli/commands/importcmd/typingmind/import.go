package typingmind

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"voidline/internal/application/conversation"
	"voidline/internal/infrastructure/config"
	"voidline/internal/infrastructure/persistence/sqlite"
)

func Command() *cobra.Command {
	var dbPath string
	var source string
	var skipDuplicates bool

	cmd := &cobra.Command{
		Use:   "typingmind",
		Short: "Import TypingMind conversations",
		RunE: func(cmd *cobra.Command, args []string) error {
			return handleImport(cmd.Context(), dbPath, source, skipDuplicates)
		},
	}

	cmd.Flags().StringVar(&dbPath, "db", "", "Path to SQLite database")
	cmd.Flags().StringVar(&source, "source", "", "Path to TypingMind export JSON file")
	cmd.Flags().BoolVar(&skipDuplicates, "skip-duplicates", true, "Skip existing records")

	if err := cmd.MarkFlagRequired("source"); err != nil {
		panic(err)
	}

	return cmd
}

func handleImport(ctx context.Context, dbPath, source string, skipDuplicates bool) error {
	if dbPath == "" {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		dbPath = cfg.Database.Path
	}

	if source == "" {
		return fmt.Errorf("source is required")
	}

	if _, err := os.Stat(source); os.IsNotExist(err) {
		return fmt.Errorf("source file does not exist: %s", source)
	}

	conn, err := sqlite.NewConnection(dbPath)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer conn.Close()

	repo := sqlite.NewConversationRepository(conn.DB())
	service := conversation.NewService(repo)

	filesDir := getFilesDir()
	options := conversation.ImportOptions{
		SkipDuplicates: skipDuplicates,
		FilesDir:       filesDir,
	}

	_, err = service.ImportTypingMind(ctx, source, options)
	if err != nil {
		return fmt.Errorf("import failed: %w", err)
	}

	return nil
}

func getFilesDir() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".config", "hominem", "files")
}
