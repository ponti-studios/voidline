package frontmatter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMigrateWriteWithBackup(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "draft.md")
	content := "---\ntitle: \"Draft\"\nslug: \"draft\"\n---\nbody\n"
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := Command()
	_, err := execute(t, cmd, "migrate", "--root", dir, "--schema", "personal", "--write", "--backup", "--output", "text")
	if err != nil {
		t.Fatalf("migrate write failed: %v", err)
	}

	after, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) == content {
		t.Fatal("expected file content to change after write")
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	foundBackup := false
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".bak") {
			foundBackup = true
			break
		}
	}
	if !foundBackup {
		t.Fatal("expected backup file")
	}
}
