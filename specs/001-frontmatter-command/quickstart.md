# Quickstart: Frontmatter CLI Command

## Prerequisites
- Go 1.25+
- Project checked out on branch `001-frontmatter-command`
- Markdown files available for testing

## 1) Build and inspect command help
```bash
go run ./cmd/cli --help
go run ./cmd/cli frontmatter --help
```

## 2) Walk markdown files
```bash
go run ./cmd/cli frontmatter walk \
  --root ./notes \
  --include-globs "notebook/**" \
  --exclude-globs "archive/**" \
  --output json
```

Expected: list of matched markdown file paths in stable JSON order.

## 3) Validate frontmatter against schema
```bash
go run ./cmd/cli frontmatter validate \
  --root ./notes \
  --schema personal \
  --output json
```

Expected: per-file validation results and summary. Exit code `1` if validation errors exist.

## 4) Migrate frontmatter (safe dry run)
```bash
go run ./cmd/cli frontmatter migrate \
  --root ./notes \
  --schema personal \
  --strategy conservative \
  --output json
```

Expected: no writes, but shows proposed changes.

## 5) Apply migration with backups
```bash
go run ./cmd/cli frontmatter migrate \
  --root ./notes \
  --schema personal \
  --write \
  --backup \
  --output text
```

Expected: changed files written and backup files created.

## 6) Detect and resolve slug collisions
```bash
go run ./cmd/cli frontmatter slug detect \
  --root ./notes \
  --scope project \
  --output json

go run ./cmd/cli frontmatter slug resolve \
  --slug my-note \
  --policy increment \
  --max-attempts 10 \
  --output json
```

Expected: collision list for detect; single resolved slug for resolve.

## 7) Run targeted tests
```bash
go test ./internal/frontmatter/...
go test ./cmd/cli/commands/frontmatter/...
go test ./cmd/cli -run TestRootCommandIncludesFrontmatter
```

Expected: command package tests pass including e2e and scale (`1000` markdown file) validation coverage.

## Agent Usage Pattern
1. Call `validate --output json`.
2. If exit code `1`, inspect errors and call `migrate` (dry-run then write).
3. Before creating files, call `slug detect` or `slug resolve`.
4. Treat exit codes `0/1/2` as success/domain-failure/runtime-failure branches.

## Exit Code Contract
- `0`: Success, no domain validation failures.
- `1`: Domain failure (for example validation errors or unresolved collision policy outcomes).
- `2`: Runtime/usage failure (invalid flags, IO/config errors, unexpected command errors).
