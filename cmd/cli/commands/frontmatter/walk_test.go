package frontmatter

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestWalkWithGlobsAndHidden(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte("# a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, ".hidden"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".hidden", "b.md"), []byte("# b"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := Command()
	output, err := execute(t, cmd, "walk", "--root", dir, "--output", "json", "--include-globs", "*.md")
	if err != nil {
		t.Fatalf("walk failed: %v", err)
	}
	var payload struct {
		Files []string `json:"files"`
	}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(payload.Files))
	}

	cmd = Command()
	output, err = execute(t, cmd, "walk", "--root", dir, "--output", "json", "--include-hidden")
	if err != nil {
		t.Fatalf("walk include-hidden failed: %v", err)
	}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(payload.Files))
	}
}

func TestWalkMaxFiles(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 5; i++ {
		path := filepath.Join(dir, fmt.Sprintf("file-%d.md", i))
		if err := os.WriteFile(path, []byte("# x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cmd := Command()
	output, err := execute(t, cmd, "walk", "--root", dir, "--output", "json", "--max-files", "2")
	if err != nil {
		t.Fatalf("walk failed: %v", err)
	}
	var payload struct {
		Files []string `json:"files"`
	}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(payload.Files))
	}
}
