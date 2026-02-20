package frontmatter

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestMigrateDryRunShowsChangesWithoutWrite(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "draft.md")
	content := "---\ntitle: \"Draft\"\nslug: \"draft\"\n---\nbody\n"
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := Command()
	output, err := execute(t, cmd, "migrate", "--root", dir, "--schema", "personal", "--output", "json")
	if err != nil {
		t.Fatalf("migrate dry-run failed: %v", err)
	}

	after, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != content {
		t.Fatal("expected file to remain unchanged in dry-run mode")
	}

	var payload struct {
		Summary struct {
			Processed int `json:"processedFiles"`
		} `json:"summary"`
	}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Summary.Processed == 0 {
		t.Fatal("expected processed files")
	}
}
