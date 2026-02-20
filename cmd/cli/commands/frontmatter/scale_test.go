package frontmatter

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestScaleValidate1000FilesNoCrash(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 1000; i++ {
		uid := fmt.Sprintf("123e4567-e89b-12d3-a456-%012d", i)
		slug := fmt.Sprintf("note-%d", i)
		content := fmt.Sprintf("---\ntitle: \"Note %d\"\nuid: \"%s\"\nslug: \"%s\"\ncreated: \"2026-01-01T00:00:00Z\"\nupdated: \"2026-01-01T00:00:00Z\"\ntype: \"reference\"\nstatus: \"draft\"\n---\n", i, uid, slug)
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("%04d.md", i)), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cmd := Command()
	if _, err := execute(t, cmd, "validate", "--root", dir, "--schema", "personal", "--output", "json"); err != nil {
		t.Fatalf("validate failed unexpectedly: %v", err)
	}
}
