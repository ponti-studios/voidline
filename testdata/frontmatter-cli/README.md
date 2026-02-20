# Frontmatter CLI Fixture Pack

This directory is a comprehensive markdown fixture set for directly testing:
- `voidline frontmatter walk`
- `voidline frontmatter validate`
- `voidline frontmatter migrate`
- `voidline frontmatter slug detect`
- `voidline frontmatter slug resolve`

## Coverage Matrix

- Valid personal schema file
- Missing required fields
- Invalid enum values (`type`, `status`)
- Invalid pattern values (`uid`, `slug`)
- No frontmatter block
- Malformed YAML frontmatter
- Unclosed frontmatter delimiter
- JSON-object style frontmatter inside delimiters
- Directory-level slug collisions
- Hidden file and hidden directory handling
- Include/exclude glob behavior
- `.markdown` extension inclusion and non-markdown exclusion
- Project schema valid/invalid examples
- PRD schema valid example
- Optional custom config via `--config`

## Quick Commands

Run from repo root:

```bash
go run ./cmd/cli frontmatter walk   --root testdata/frontmatter-cli/notebook   --output json

go run ./cmd/cli frontmatter validate   --root testdata/frontmatter-cli/notebook/personal   --schema personal   --output json

go run ./cmd/cli frontmatter migrate   --root testdata/frontmatter-cli/notebook/personal   --schema personal   --strategy fill   --output text

go run ./cmd/cli frontmatter slug detect   --root testdata/frontmatter-cli/notebook/personal/collision   --scope directory   --output json

go run ./cmd/cli frontmatter slug resolve   --slug same-slug   --policy increment   --existing-slugs same-slug   --existing-slugs same-slug-2   --output json
```

## Optional strict config test

```bash
go run ./cmd/cli frontmatter validate   --root testdata/frontmatter-cli/notebook/personal   --schema personal   --config testdata/frontmatter-cli/configs/strict-settings.yaml   --output json
```

The strict config adds `summary` as required to force additional validation failures.
