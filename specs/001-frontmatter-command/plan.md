# Implementation Plan: Frontmatter CLI Command

**Branch**: `001-frontmatter-command` | **Date**: 2026-02-20 | **Spec**: `/specs/001-frontmatter-command/spec.md`
**Input**: Feature specification from `/specs/001-frontmatter-command/spec.md`

## Summary

Add a new `voidline frontmatter` command group that exposes existing `internal/frontmatter` capabilities through a CLI-first interface for both humans and automation/AI agents. The implementation will add Cobra subcommands (`walk`, `validate`, `migrate`, `slug detect`, `slug resolve`) with deterministic exit codes, stable JSON output, and explicit write controls (`--dry-run`, `--write`, `--backup`).

## Technical Context

**Language/Version**: Go 1.25  
**Primary Dependencies**: `github.com/spf13/cobra`, `gopkg.in/yaml.v3`, existing `internal/frontmatter` package  
**Storage**: File system (markdown files and optional backups)  
**Testing**: `go test` with table-driven unit tests in command package and existing `internal/frontmatter/*_test.go`  
**Target Platform**: Cross-platform CLI (macOS/Linux primary)  
**Project Type**: Single Go CLI project  
**Performance Goals**: Validate 1,000 markdown files in one run without crash; bounded memory via file-by-file processing  
**Constraints**: No file mutation unless `--write` is set; deterministic JSON output for automation; non-zero exits on failures  
**Scale/Scope**: One new command group + subcommands, integration with existing frontmatter library only

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

The constitution is now ratified and enforceable at `.specify/memory/constitution.md` (Version 1.0.0).

**Pre-Phase 0 Gate Result**: PASS  
**Rationale**: Planned approach aligns with constitutional requirements for reuse-first architecture, deterministic CLI contracts, safety-before-mutation defaults, and verified behavior changes.

Constitution gates applied for this feature:
1. Reuse existing domain/library code instead of duplicating frontmatter logic.
2. Keep CLI behavior deterministic with machine-readable output mode and explicit exit semantics.
3. Preserve safety defaults (`--dry-run` behavior; explicit write intent and backup option).
4. Add or update command-layer tests for behavior and failure-mode verification.
5. Keep delivery incremental by user-story phases with MVP-first ordering.

**Post-Phase 1 Re-check**: PASS  
Design artifacts and tasks remain consistent with constitutional principles and include explicit scale/performance validation coverage.

## Project Structure

### Documentation (this feature)

```text
specs/001-frontmatter-command/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── frontmatter-cli.openapi.yaml
└── tasks.md
```

### Source Code (repository root)

```text
cmd/
└── cli/
    ├── main.go
    └── commands/
        └── frontmatter/
            ├── cmd.go
            ├── walk.go
            ├── validate.go
            ├── migrate.go
            ├── slug.go
            └── output.go

internal/
└── frontmatter/
    ├── workflow.go
    ├── migrator.go
    ├── validator.go
    ├── slug.go
    └── ...
```

**Structure Decision**: Use the existing single-project CLI structure and add a focused `cmd/cli/commands/frontmatter` adapter layer that orchestrates existing `internal/frontmatter` business logic.

## Complexity Tracking

No constitution violations currently require exception tracking.
