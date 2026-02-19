package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"

	"gogogo/cmd/cli/commands/browser"
	"gogogo/cmd/cli/commands/finance"
	"gogogo/cmd/cli/commands/flatten"
	"gogogo/cmd/cli/commands/importcmd/music"
	"gogogo/cmd/cli/commands/notes"
	"gogogo/cmd/cli/commands/random"
	"gogogo/cmd/cli/commands/server"
	"gogogo/cmd/cli/commands/typingmind"
)

func main() {
	app := &cli.App{
		Name:  "gogogo",
		Usage: "CLI utilities and tools",
		Commands: []*cli.Command{
			{
				Name:   "browser",
				Usage:  "Browser automation tools",
				Action: func(c *cli.Context) error { return browser.Run() },
			},
			{
				Name:  "flatten",
				Usage: "Flatten directory structure",
				Action: func(c *cli.Context) error {
					return flatten.Run()
				},
			},
			{
				Name:  "typingmind",
				Usage: "Convert TypingMind chat data",
				Action: func(c *cli.Context) error {
					return typingmind.Run()
				},
			},
			{
				Name:   "finance",
				Usage:  "Finance and budget calculator",
				Action: func(c *cli.Context) error { return finance.RunCLI() },
			},
			{
				Name:   "notes",
				Usage:  "Parse markdown files for notes",
				Action: func(c *cli.Context) error { return notes.Run() },
			},
			{
				Name:   "random",
				Usage:  "Go example code snippets",
				Action: func(c *cli.Context) error { return random.Run() },
			},
			{
				Name:   "server",
				Usage:  "Start GraphQL server",
				Action: func(c *cli.Context) error { return server.Run() },
			},
			{
				Name:        "import",
				Usage:       "Import data from various sources",
				Subcommands: importSubcommands(),
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func importSubcommands() []*cli.Command {
	return []*cli.Command{
		{
			Name:  "amazon",
			Usage: "Import Amazon orders",
			Action: func(c *cli.Context) error {
				fmt.Println("Usage: gogogo import amazon <args>")
				return nil
			},
		},
		{
			Name:  "apple",
			Usage: "Import Apple receipts",
			Action: func(c *cli.Context) error {
				fmt.Println("Usage: gogogo import apple <args>")
				return nil
			},
		},
		{
			Name:  "health",
			Usage: "Import health data",
			Action: func(c *cli.Context) error {
				fmt.Println("Usage: gogogo import health <args>")
				return nil
			},
		},
		{
			Name:  "music",
			Usage: "Import music data (Spotify, Apple Music)",
			Subcommands: []*cli.Command{
				{
					Name:  "spotify",
					Usage: "Import Spotify data",
					Flags: []cli.Flag{
						&cli.StringFlag{
							Name:  "db",
							Usage: "Path to SQLite database",
						},
						&cli.StringFlag{
							Name:  "source",
							Usage: "Source directory containing Spotify export (e.g., .data/spotify)",
						},
						&cli.BoolFlag{
							Name:  "dry-run",
							Usage: "Validate without importing",
						},
						&cli.BoolFlag{
							Name:  "force",
							Usage: "Skip duplicate checking",
						},
					},
					Action: func(c *cli.Context) error {
						return music.HandleSpotifyImport(
							c.Context,
							c.String("db"),
							c.String("source"),
							c.Bool("dry-run"),
							c.Bool("force"),
						)
					},
				},
				{
					Name:  "apple",
					Usage: "Import Apple Music data",
					Flags: []cli.Flag{
						&cli.StringFlag{
							Name:  "db",
							Usage: "Path to SQLite database",
						},
						&cli.StringFlag{
							Name:  "source",
							Usage: "Source directory containing Apple Music export",
						},
						&cli.BoolFlag{
							Name:  "dry-run",
							Usage: "Validate without importing",
						},
						&cli.BoolFlag{
							Name:  "force",
							Usage: "Skip duplicate checking",
						},
					},
					Action: func(c *cli.Context) error {
						return music.HandleAppleMusicImport(
							c.Context,
							c.String("db"),
							c.String("source"),
							c.Bool("dry-run"),
							c.Bool("force"),
						)
					},
				},
			},
		},
		{
			Name:  "tracking",
			Usage: "Import tracking data",
			Action: func(c *cli.Context) error {
				fmt.Println("Usage: gogogo import tracking <args>")
				return nil
			},
		},
		{
			Name:  "social",
			Usage: "Import social media data",
			Action: func(c *cli.Context) error {
				fmt.Println("Usage: gogogo import social <args>")
				return nil
			},
		},
	}
}
