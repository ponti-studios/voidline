# Phase 0 Research: Frontmatter CLI Command

## Decision 1: Implement as Cobra subcommands under `voidline frontmatter`
- **Decision**: Add a dedicated command group in `cmd/cli/commands/frontmatter` with subcommands `walk`, `validate`, `migrate`, `slug detect`, and `slug resolve`.
- **Rationale**: The repository already standardizes on Cobra command modules under `cmd/cli/commands/*`; this keeps discoverability and command ergonomics consistent.
- **Alternatives considered**:
  - Add all flags to one monolithic `frontmatter` command (rejected: harder UX, mixed concerns).
  - Build a separate binary (rejected: unnecessary operational overhead and duplicate wiring).

## Decision 2: Reuse `internal/frontmatter` as the only business logic layer
- **Decision**: Command handlers call `WalkMarkdownFiles`, `ValidateFiles`, `MigrateFiles`, `DetectSlugCollisions`, and `ResolveSlugCollision` directly.
- **Rationale**: Avoids reimplementing parsing/validation/migration behavior and keeps semantics aligned with existing tested code.
- **Alternatives considered**:
  - Re-implement lightweight parser in command package (rejected: drift risk and duplicated maintenance).
  - Move package to external module first (rejected for this feature scope; can be done later).

## Decision 3: Provide deterministic text + JSON output contracts
- **Decision**: Every subcommand supports `--output text|json` (default `text`) and writes stable JSON for machine consumption.
- **Rationale**: AI agents and CI pipelines require parseable output with deterministic shape; humans need concise terminal summaries.
- **Alternatives considered**:
  - Text-only output (rejected: brittle for automation parsing).
  - JSON-only output (rejected: poor default UX for humans).

## Decision 4: Use safety-first mutation model
- **Decision**: `migrate` defaults to dry-run semantics unless `--write` is explicitly provided; optional `--backup` creates pre-write backups.
- **Rationale**: Frontmatter mutation is potentially destructive. Explicit intent and backup capability reduce accidental data loss risk.
- **Alternatives considered**:
  - Write by default with optional dry-run (rejected: too risky).
  - Always backup (rejected: unnecessary filesystem overhead for every run).

## Decision 5: Exit code strategy optimized for automation
- **Decision**:
  - `0`: success (no validation errors)
  - `1`: domain failures (validation errors, collision found in strict mode)
  - `2`: runtime/usage failures (bad args, config load errors, IO failure)
- **Rationale**: Agents can branch behavior quickly using status code alone before parsing full output payloads.
- **Alternatives considered**:
  - Single non-zero code for all errors (rejected: less actionable automation behavior).

## Decision 6: Config and schema loading source
- **Decision**: Accept explicit `--config` path with fallback to package defaults (`frontmatter.DefaultConfig()`) when absent.
- **Rationale**: Supports portable agent runs and local ad-hoc use without mandatory setup.
- **Alternatives considered**:
  - Require config file always (rejected: unnecessary friction).
  - Infer from many fallback locations (rejected for MVP complexity).

## Decision 7: Performance and scale handling
- **Decision**: Process files in a single pass and operate file-by-file, using existing traversal options including `MaxFiles`.
- **Rationale**: Fits expected 1k-file workloads while keeping memory bounded and implementation straightforward.
- **Alternatives considered**:
  - Add concurrency worker pool (rejected for initial release; complexity not required yet).
  - Read all file contents before processing (rejected: avoid memory spikes).

## Resolved Technical Clarifications
- Language/version: **Go 1.25**.
- Dependency strategy: **Cobra + existing internal frontmatter package**.
- Persistence model: **filesystem only, no DB**.
- Testing strategy: **go test table-driven command tests + existing frontmatter tests**.
- Platform target: **cross-platform CLI with macOS/Linux as primary development targets**.
- Output contract for agents: **stable JSON via `--output json` plus deterministic exit codes**.
