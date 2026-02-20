package frontmatter

import "testing"

func TestSlugResolveIncrement(t *testing.T) {
	cmd := Command()
	output, err := execute(t, cmd, "slug", "resolve", "--slug", "note", "--policy", "increment", "--existing-slugs", "note", "--existing-slugs", "note-2", "--output", "json")
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	if output == "" {
		t.Fatal("expected output")
	}
}

func TestSlugResolveFailPolicyReturnsDomainError(t *testing.T) {
	cmd := Command()
	_, err := execute(t, cmd, "slug", "resolve", "--slug", "note", "--policy", "fail", "--existing-slugs", "note", "--output", "text")
	if err == nil {
		t.Fatal("expected error")
	}
	ce, ok := err.(*commandError)
	if !ok {
		t.Fatalf("expected commandError, got %T", err)
	}
	if ce.ExitCode() != exitDomain {
		t.Fatalf("expected domain exit code, got %d", ce.ExitCode())
	}
}
