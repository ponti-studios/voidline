package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootCommandIncludesFrontmatter(t *testing.T) {
	cmd := rootCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("help failed: %v", err)
	}
	if !strings.Contains(out.String(), "frontmatter") {
		t.Fatal("expected frontmatter command in help output")
	}
}
