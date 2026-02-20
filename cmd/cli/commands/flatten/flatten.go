package flatten

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

type Config struct {
	dryRun           bool
	directory        string
	includeParentDir bool
}

func log(message string) {
	fmt.Printf("[%s] %s\n", time.Now().Format("2006-01-02 15:04:05"), message)
}

func moveFile(path string, config Config) error {
	base := filepath.Base(path)
	if strings.HasPrefix(base, ".") {
		return nil
	}

	dir := filepath.Dir(path)
	if dir == config.directory {
		return nil
	}

	newBase := base
	if config.includeParentDir {
		parentDir := filepath.Base(filepath.Dir(path))
		if parentDir != config.directory {
			ext := filepath.Ext(base)
			name := strings.TrimSuffix(base, ext)
			newBase = fmt.Sprintf("%s_%s%s", name, parentDir, ext)
		}
	}

	newPath := filepath.Join(config.directory, newBase)
	counter := 1
	for {
		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			break
		}
		ext := filepath.Ext(newBase)
		name := strings.TrimSuffix(newBase, ext)
		newPath = filepath.Join(config.directory, fmt.Sprintf("%s_%d%s", name, counter, ext))
		counter++
	}

	if config.dryRun {
		log(fmt.Sprintf("Would move: %s -> %s", path, newPath))
		return nil
	}

	if err := os.Rename(path, newPath); err != nil {
		return fmt.Errorf("error moving file: %w", err)
	}
	log(fmt.Sprintf("Moved: %s -> %s", path, newPath))
	return nil
}

func Command() *cobra.Command {
	config := Config{}

	cmd := &cobra.Command{
		Use:   "flatten",
		Short: "Flatten directory structure",
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(config)
		},
	}

	cmd.Flags().BoolVar(&config.dryRun, "d", false, "Dry run mode")
	cmd.Flags().BoolVar(&config.includeParentDir, "p", false, "Include parent directory name in filename")
	cmd.Flags().StringVar(&config.directory, "dir", "", "Directory to flatten")

	if err := cmd.MarkFlagRequired("dir"); err != nil {
		panic(err)
	}

	return cmd
}

func Run() error {
	cmd := Command()
	cmd.SetArgs(os.Args[1:])
	return cmd.Execute()
}

func run(config Config) error {
	if config.directory == "" {
		return fmt.Errorf("missing required directory")
	}

	if info, err := os.Stat(config.directory); err != nil || !info.IsDir() {
		return fmt.Errorf("invalid directory: %s", config.directory)
	}

	log(fmt.Sprintf("Starting file reorganization in directory: %s", config.directory))

	err := filepath.WalkDir(config.directory, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && d.Name() != ".DS_Store" {
			if err := moveFile(path, config); err != nil {
				log(fmt.Sprintf("Failed to process: %s - %v", path, err))
			}
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("error walking directory: %w", err)
	}

	log("Finished file reorganization")
	return nil
}
