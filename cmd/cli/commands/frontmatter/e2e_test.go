package frontmatter

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEndToEndFlow(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "draft.md")
	content := "---\ntitle: \"Draft\"\nslug: \"draft\"\n---\nbody\n"
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := Command()
	if _, err := execute(t, cmd, "walk", "--root", dir, "--output", "json"); err != nil {
		t.Fatalf("walk failed: %v", err)
	}
	if _, err := execute(t, cmd, "validate", "--root", dir, "--schema", "personal", "--output", "json"); err == nil {
		t.Fatal("expected validate to fail for incomplete frontmatter")
	}
	if _, err := execute(t, cmd, "migrate", "--root", dir, "--schema", "personal", "--write", "--output", "json"); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}
	if _, err := execute(t, cmd, "slug", "detect", "--root", dir, "--scope", "project", "--output", "json"); err != nil {
		t.Fatalf("slug detect failed: %v", err)
	}
}
