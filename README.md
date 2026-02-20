# gogogo

CLI utilities and tools for finance, imports, and server operations.

## Build

```bash
make build
```

## Run

```bash
./bin/gogogo --help
```

## Command Overview

### finance

Finance and budget utilities.

```bash
./gogogo finance --help
```

Budget subcommands:

- init: Create a new budget interactively
- show: Display current budget status
- calendar: Show cash flow calendar
- scenario: Test what-if scenarios
- export: Export budget data

Examples:

```bash
./gogogo finance budget init
./gogogo finance budget show --view categories
./gogogo finance budget calendar --month 2026-02
./gogogo finance budget scenario --reduce-expense 'Dining:50'
./gogogo finance budget export --format yaml
```

Report and dashboard:

```bash
./gogogo finance report --db /path/to/db.sqlite --type transactions --format table
./gogogo finance dashboard --db /path/to/db.sqlite --format json
```

### import

Import data from multiple sources.

```bash
./gogogo import --help
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
./gogogo import amazon --source /path/to/amazon --db /path/to/db.sqlite
./gogogo import health --source /path/to/health --source-type all
./gogogo import music spotify --source /path/to/spotify --db /path/to/db.sqlite
./gogogo import typingmind --source /path/to/typingmind.json
./gogogo import openai --source /path/to/openai-export
```

### flatten

Flatten a directory structure into a single folder.

```bash
./gogogo flatten --dir /path/to/dir --d --p
```

Flags:

- --dir: Directory to flatten (required)
- -d: Dry run
- -p: Include parent directory name in filename

### server

Start the REST API server.

```bash
./gogogo server
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
