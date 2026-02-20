package main

import (
	"context"
	"errors"
	"os"

	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"

	"voidline/cmd/cli/commands/finance"
	"voidline/cmd/cli/commands/flatten"
	"voidline/cmd/cli/commands/frontmatter"
	"voidline/cmd/cli/commands/importcmd/amazon"
	"voidline/cmd/cli/commands/importcmd/apple"
	"voidline/cmd/cli/commands/importcmd/health"
	"voidline/cmd/cli/commands/importcmd/music"
	"voidline/cmd/cli/commands/importcmd/openai"
	"voidline/cmd/cli/commands/importcmd/social"
	"voidline/cmd/cli/commands/importcmd/typingmind"
	"voidline/cmd/cli/commands/server"
)

func main() {
	rootCmd := rootCommand()

	if err := fang.Execute(context.Background(), rootCmd); err != nil {
		var exitCoder interface{ ExitCode() int }
		if errors.As(err, &exitCoder) {
			os.Exit(exitCoder.ExitCode())
		}
		os.Exit(1)
	}
}

func rootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "voidline",
		Short: "CLI utilities and tools",
	}

	rootCmd.AddCommand(flatten.Command())
	rootCmd.AddCommand(finance.Command())
	rootCmd.AddCommand(frontmatter.Command())
	rootCmd.AddCommand(serverCmd())
	rootCmd.AddCommand(importCmd())
	return rootCmd
}

func serverCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "server",
		Short: "Start REST API server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return server.Run()
		},
	}
}

func importCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import data from various sources",
	}

	cmd.AddCommand(amazon.Command())
	cmd.AddCommand(apple.Command())
	cmd.AddCommand(health.Command())
	cmd.AddCommand(importMusicCmd())
	cmd.AddCommand(social.Command())
	cmd.AddCommand(typingmind.Command())
	cmd.AddCommand(openai.Command())

	return cmd
}

func importMusicCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "music",
		Short: "Import music data (Spotify, Apple Music)",
	}

	cmd.AddCommand(importSpotifyCmd())
	cmd.AddCommand(importAppleMusicCmd())

	return cmd
}

func importSpotifyCmd() *cobra.Command {
	var db, source string
	var dryRun, force bool

	cmd := &cobra.Command{
		Use:   "spotify",
		Short: "Import Spotify data",
		RunE: func(cmd *cobra.Command, args []string) error {
			return music.HandleSpotifyImport(
				cmd.Context(),
				db,
				source,
				dryRun,
				force,
			)
		},
	}

	cmd.Flags().StringVar(&db, "db", "", "Path to SQLite database")
	cmd.Flags().StringVar(&source, "source", "", "Source directory containing Spotify export")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Validate without importing")
	cmd.Flags().BoolVar(&force, "force", false, "Skip duplicate checking")

	return cmd
}

func importAppleMusicCmd() *cobra.Command {
	var db, source string
	var dryRun, force bool

	cmd := &cobra.Command{
		Use:   "apple",
		Short: "Import Apple Music data",
		RunE: func(cmd *cobra.Command, args []string) error {
			return music.HandleAppleMusicImport(
				cmd.Context(),
				db,
				source,
				dryRun,
				force,
			)
		},
	}

	cmd.Flags().StringVar(&db, "db", "", "Path to SQLite database")
	cmd.Flags().StringVar(&source, "source", "", "Source directory containing Apple Music export")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Validate without importing")
	cmd.Flags().BoolVar(&force, "force", false, "Skip duplicate checking")

	return cmd
}
