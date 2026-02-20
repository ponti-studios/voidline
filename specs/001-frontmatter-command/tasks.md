# Tasks: Frontmatter CLI Command

**Input**: Design documents from `/specs/001-frontmatter-command/`
**Prerequisites**: `plan.md` (required), `spec.md` (required for user stories), `research.md`, `data-model.md`, `contracts/`

**Tests**: Include command-layer tests because the plan explicitly requires adding/updating tests for command behavior and deterministic exit/output semantics.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Create command package structure and wire it into the CLI.

- [X] T001 Create frontmatter command package files in `cmd/cli/commands/frontmatter/cmd.go`, `cmd/cli/commands/frontmatter/walk.go`, `cmd/cli/commands/frontmatter/validate.go`, `cmd/cli/commands/frontmatter/migrate.go`, `cmd/cli/commands/frontmatter/slug.go`, and `cmd/cli/commands/frontmatter/output.go`
- [X] T002 Register the new `frontmatter` command group in `cmd/cli/main.go`
- [X] T003 [P] Create command test fixtures directory and seed files in `cmd/cli/commands/frontmatter/testdata/`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Build shared plumbing required by all stories.

**⚠️ CRITICAL**: No user story work should begin until this phase is complete.

- [X] T004 Implement shared request/flag structs and root command factory in `cmd/cli/commands/frontmatter/cmd.go`
- [X] T005 [P] Implement config/schema resolution adapter to `internal/frontmatter` defaults and optional config path in `cmd/cli/commands/frontmatter/cmd.go`
- [X] T006 [P] Implement shared output envelope and text/JSON rendering helpers in `cmd/cli/commands/frontmatter/output.go`
- [X] T007 Implement deterministic exit-code mapping (`0/1/2`) and error normalization in `cmd/cli/commands/frontmatter/output.go`
- [X] T008 Add foundational command behavior tests for flag validation and exit-code mapping in `cmd/cli/commands/frontmatter/cmd_test.go`

**Checkpoint**: Foundation complete; story-specific subcommands can now be implemented.

---

## Phase 3: User Story 1 - Validate frontmatter at scale (Priority: P1) 🎯 MVP

**Goal**: Users and automation can traverse markdown trees and validate frontmatter with deterministic text/JSON output.

**Independent Test**: Run `voidline frontmatter validate --root <dir> --schema personal --output json` on valid/invalid fixtures and verify deterministic payload and expected exit code.

### Tests for User Story 1

- [X] T009 [P] [US1] Add traversal behavior tests (globs, hidden files, max-files) in `cmd/cli/commands/frontmatter/walk_test.go`
- [X] T010 [P] [US1] Add validate command success/failure and JSON shape tests in `cmd/cli/commands/frontmatter/validate_test.go`

### Implementation for User Story 1

- [X] T011 [P] [US1] Implement `walk` subcommand and traversal flags in `cmd/cli/commands/frontmatter/walk.go`
- [X] T012 [US1] Implement `validate` subcommand and request mapping in `cmd/cli/commands/frontmatter/validate.go`
- [X] T013 [US1] Connect `validate` execution to `internal/frontmatter.ValidateFiles` in `cmd/cli/commands/frontmatter/validate.go`
- [X] T014 [US1] Implement validation summary aggregation and response rendering in `cmd/cli/commands/frontmatter/output.go`
- [X] T015 [US1] Add walk/validate examples and help text in `cmd/cli/commands/frontmatter/cmd.go`

**Checkpoint**: US1 is fully functional and independently testable.

---

## Phase 4: User Story 2 - Migrate/fix frontmatter safely (Priority: P2)

**Goal**: Users can run safe dry-run migrations and opt into writes/backups explicitly.

**Independent Test**: Run `migrate` in dry-run and write modes and verify changed file counts, backup behavior, and write safety constraints.

### Tests for User Story 2

- [X] T016 [P] [US2] Add migrate dry-run and summary behavior tests in `cmd/cli/commands/frontmatter/migrate_test.go`
- [X] T017 [P] [US2] Add migrate write/backup mutation tests with temp directories in `cmd/cli/commands/frontmatter/migrate_write_test.go`

### Implementation for User Story 2

- [X] T018 [US2] Implement `migrate` subcommand flags (`--strategy`, `--write`, `--backup`) in `cmd/cli/commands/frontmatter/migrate.go`
- [X] T019 [US2] Connect `migrate` execution to `internal/frontmatter.MigrateFiles` in `cmd/cli/commands/frontmatter/migrate.go`
- [X] T020 [US2] Enforce no-write-by-default semantics and mutation safeguards in `cmd/cli/commands/frontmatter/migrate.go`
- [X] T021 [US2] Add migration change/backup reporting in text and JSON outputs in `cmd/cli/commands/frontmatter/output.go`

**Checkpoint**: US2 is independently testable and preserves safety-first defaults.

---

## Phase 5: User Story 3 - Manage slug collisions for note creation (Priority: P3)

**Goal**: AI agents and users can detect collisions and resolve slugs under policy constraints.

**Independent Test**: Use duplicate-slug fixtures to validate `slug detect` scope behavior and `slug resolve` policy behavior with deterministic outputs.

### Tests for User Story 3

- [X] T022 [P] [US3] Add slug detect scope tests (`directory`, `project`, `global`) in `cmd/cli/commands/frontmatter/slug_detect_test.go`
- [X] T023 [P] [US3] Add slug resolve policy and max-attempt tests in `cmd/cli/commands/frontmatter/slug_resolve_test.go`

### Implementation for User Story 3

- [X] T024 [US3] Implement `slug` parent command and child routing in `cmd/cli/commands/frontmatter/slug.go`
- [X] T025 [US3] Implement `slug detect` request mapping to `internal/frontmatter.DetectSlugCollisions` in `cmd/cli/commands/frontmatter/slug.go`
- [X] T026 [US3] Implement `slug resolve` request mapping to `internal/frontmatter.ResolveSlugCollision` in `cmd/cli/commands/frontmatter/slug.go`
- [X] T027 [US3] Implement slug detect/resolve output payloads and exit behavior in `cmd/cli/commands/frontmatter/output.go`

**Checkpoint**: US3 is independently testable and agent-ready.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Final integration hardening and documentation consistency.

- [X] T028 [P] Add end-to-end command coverage test for `walk/validate/migrate/slug` flows in `cmd/cli/commands/frontmatter/e2e_test.go`
- [X] T032 [P] Add scale validation test for at least 1,000 markdown files and no-crash guarantee in `cmd/cli/commands/frontmatter/scale_test.go`
- [X] T029 [P] Update command usage and automation workflow examples in `specs/001-frontmatter-command/quickstart.md`
- [X] T030 [P] Align output and exit-code examples with implementation in `specs/001-frontmatter-command/contracts/frontmatter-cli.openapi.yaml`
- [X] T031 Update top-level CLI integration assertions for frontmatter command discovery/help output in `cmd/cli/main_test.go`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: Starts immediately.
- **Phase 2 (Foundational)**: Depends on Phase 1 and blocks all story work.
- **Phase 3 (US1)**: Depends on Phase 2; recommended MVP first.
- **Phase 4 (US2)**: Depends on Phase 2; may proceed after US1 for incremental delivery.
- **Phase 5 (US3)**: Depends on Phase 2; may proceed after US1/US2 depending on priority.
- **Phase 6 (Polish)**: Depends on completion of selected user stories.

### User Story Dependencies

- **US1 (P1)**: No dependency on other stories after foundational work.
- **US2 (P2)**: Depends on shared foundational output/exit infrastructure; can reuse US1 command scaffolding.
- **US3 (P3)**: Depends on shared foundational infrastructure; independent of US2 logic.

### Within-Story Ordering

- Test tasks precede implementation tasks in each story.
- Subcommand wiring precedes output polishing for that story.
- Tasks touching the same file (`output.go`, `cmd.go`, `slug.go`, `migrate.go`) should run sequentially.

---

## Parallel Execution Examples

### US1 Parallel Example

- Run `T009` and `T010` together (different test files).
- Run `T011` in parallel with test authoring, then complete `T012-T015` sequentially.

### US2 Parallel Example

- Run `T016` and `T017` together (different test files).
- Execute `T018-T021` sequentially due to shared file dependencies.

### US3 Parallel Example

- Run `T022` and `T023` together (different test files).
- Execute `T024-T027` sequentially due to shared `slug.go` and `output.go` dependencies.

---

## Implementation Strategy

### MVP First (Recommended)

1. Complete Phase 1 and Phase 2.
2. Deliver Phase 3 (US1) as the first shippable increment.
3. Validate with quickstart command scenarios before continuing.

### Incremental Delivery

1. Add US2 migration safety workflow after US1 validation is stable.
2. Add US3 slug-agent workflows as the final functional increment.
3. Complete polish tasks to lock docs/contracts to implemented behavior.
