package frontmatter

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSlugDetectScopes(t *testing.T) {
	dir := t.TempDir()
	d1 := filepath.Join(dir, "a")
	d2 := filepath.Join(dir, "b")
	if err := os.MkdirAll(d1, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(d2, 0o755); err != nil {
		t.Fatal(err)
	}
	c := "---\ntitle: \"Note\"\nuid: \"123e4567-e89b-12d3-a456-426614174000\"\nslug: \"same\"\ncreated: \"2026-01-01T00:00:00Z\"\nupdated: \"2026-01-01T00:00:00Z\"\ntype: \"reference\"\nstatus: \"draft\"\n---\n"
	if err := os.WriteFile(filepath.Join(d1, "one.md"), []byte(c), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(d2, "two.md"), []byte(c), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := Command()
	output, err := execute(t, cmd, "slug", "detect", "--root", dir, "--scope", "project", "--output", "json")
	if err != nil {
		t.Fatalf("slug detect failed: %v", err)
	}
	var payload struct {
		Collisions []interface{} `json:"collisions"`
	}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Collisions) == 0 {
		t.Fatal("expected collisions")
	}
}
