package frontmatter

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

func execute(t *testing.T, cmd *cobra.Command, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func testDataPath(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join("testdata", name)
}

func TestCommandHasSubcommands(t *testing.T) {
	cmd := Command()
	if cmd == nil {
		t.Fatal("expected command")
	}
	if len(cmd.Commands()) < 4 {
		t.Fatalf("expected at least 4 subcommands, got %d", len(cmd.Commands()))
	}
}

func TestValidateOutputInvalid(t *testing.T) {
	err := validateOutput("xml")
	if err == nil {
		t.Fatal("expected error")
	}
	ce, ok := err.(*commandError)
	if !ok {
		t.Fatalf("expected commandError, got %T", err)
	}
	if ce.ExitCode() != exitRuntime {
		t.Fatalf("expected runtime exit code, got %d", ce.ExitCode())
	}
}

func TestParseStrategyInvalid(t *testing.T) {
	_, err := parseStrategy("invalid")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestWalkJSONShape(t *testing.T) {
	cmd := Command()
	output, err := execute(t, cmd, "walk", "--root", testDataPath(t, "."), "--output", "json")
	if err != nil {
		t.Fatalf("walk failed: %v", err)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("invalid json output: %v", err)
	}
	if _, ok := payload["files"]; !ok {
		t.Fatal("expected files field")
	}
	if _, ok := payload["summary"]; !ok {
		t.Fatal("expected summary field")
	}
}
