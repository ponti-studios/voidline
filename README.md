# voidline

CLI utilities and tools for finance, frontmatter, imports, flattening, and server operations.

## Build

```bash
make build
```

## Run

```bash
./bin/voidline --help
```

## Command Overview

Current top-level commands:

- `finance`
- `frontmatter`
- `import`
- `flatten`
- `server`

### finance

Finance and budget utilities.

```bash
./voidline finance --help
```

Budget subcommands:

- init: Create a new budget interactively
- show: Display current budget status
- calendar: Show cash flow calendar
- scenario: Test what-if scenarios
- export: Export budget data

Examples:

```bash
./voidline finance budget init
./voidline finance budget show --view categories
./voidline finance budget calendar --month 2026-02
./voidline finance budget scenario --reduce-expense 'Dining:50'
./voidline finance budget export --format yaml
```

Report and dashboard:

```bash
./voidline finance report --db /path/to/db.sqlite --type transactions --format table
./voidline finance dashboard --db /path/to/db.sqlite --format json
```

### frontmatter

Validate, migrate, and manage markdown frontmatter.

```bash
./bin/voidline frontmatter --help
```

Core subcommands:

- walk: List markdown files under a root
- validate: Validate frontmatter against a schema
- migrate: Apply schema-guided migration (dry-run by default)
- slug detect: Detect slug collisions
- slug resolve: Resolve slug collisions by policy

Examples:

```bash
./bin/voidline frontmatter walk --root ./testdata/frontmatter-cli/notebook --output json
./bin/voidline frontmatter validate --root ./testdata/frontmatter-cli/notebook/personal --schema personal --output json
./bin/voidline frontmatter migrate --root ./testdata/frontmatter-cli/notebook/personal --schema personal --strategy fill --output text
./bin/voidline frontmatter migrate --root ./testdata/frontmatter-cli/notebook/personal --schema personal --write --backup --output text
./bin/voidline frontmatter slug detect --root ./testdata/frontmatter-cli/notebook/personal/collision --scope directory --output json
./bin/voidline frontmatter slug resolve --slug same-slug --policy increment --existing-slugs same-slug --existing-slugs same-slug-2 --output json
```

Exit code semantics:

- `0`: success
- `1`: domain validation/collision failure
- `2`: runtime or usage failure

### import

Import data from multiple sources.

```bash
./voidline import --help
```

Available imports:

- amazon
- apple
- health
- music (spotify, apple)
- social
- typingmind
- openai

Examples:

```bash
./voidline import amazon --source /path/to/amazon --db /path/to/db.sqlite
./voidline import health --source /path/to/health --source-type all
./voidline import music spotify --source /path/to/spotify --db /path/to/db.sqlite
./voidline import typingmind --source /path/to/typingmind.json
./voidline import openai --source /path/to/openai-export
```

### flatten

Flatten a directory structure into a single folder.

```bash
./voidline flatten --dir /path/to/dir --d --p
```

Flags:

- --dir: Directory to flatten (required)
- -d: Dry run
- -p: Include parent directory name in filename

### server

Start the REST API server.

```bash
./voidline server
```

## Development

```bash
make tools
make dev
```

Recommended workflow:

1. Install toolchain (lint, format, swagger, live reload):

```bash
make tools
```

2. Run live-reload server with auto Swagger regeneration:

```bash
make dev
```

3. Lint and format before commits:

```bash
make lint
make fmt
```

4. Regenerate Swagger docs manually (if needed):

```bash
make swagger
```
