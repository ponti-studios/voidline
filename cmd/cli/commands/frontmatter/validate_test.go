package frontmatter

import (
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestValidateCommandDomainFailure(t *testing.T) {
	cmd := Command()
	root := filepath.Join("testdata")
	output, err := execute(t, cmd, "validate", "--root", root, "--schema", "personal", "--output", "json")
	if err == nil {
		t.Fatal("expected domain error")
	}
	ce, ok := err.(*commandError)
	if !ok {
		t.Fatalf("expected commandError, got %T", err)
	}
	if ce.ExitCode() != exitDomain {
		t.Fatalf("expected exitDomain, got %d", ce.ExitCode())
	}

	var payload struct {
		Summary struct {
			ErrorFiles int `json:"errorFiles"`
			ExitCode   int `json:"exitCode"`
		} `json:"summary"`
	}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("invalid json payload: %v", err)
	}
	if payload.Summary.ErrorFiles == 0 {
		t.Fatal("expected error files")
	}
	if payload.Summary.ExitCode != exitDomain {
		t.Fatalf("expected summary exit code %d, got %d", exitDomain, payload.Summary.ExitCode)
	}
}

func TestValidateJSONDeterministic(t *testing.T) {
	cmd := Command()
	root := filepath.Join("testdata")
	output1, _ := execute(t, cmd, "validate", "--root", root, "--schema", "personal", "--output", "json")
	output2, _ := execute(t, cmd, "validate", "--root", root, "--schema", "personal", "--output", "json")
	if output1 != output2 {
		t.Fatal("expected deterministic JSON output")
	}
}
