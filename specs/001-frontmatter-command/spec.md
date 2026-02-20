# Feature Specification: Frontmatter CLI Command

**Feature Branch**: `001-frontmatter-command`  
**Created**: 2026-02-20  
**Status**: Draft  
**Input**: User description: "create plan for the new frontmatter command"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Validate frontmatter at scale (Priority: P1)

As a user or automation workflow owner, I can run a CLI command to validate frontmatter across a markdown tree and get deterministic results in text or JSON.

**Why this priority**: Validation is the minimum usable capability and provides immediate value for both humans and CI/agent workflows.

**Independent Test**: Can be fully tested by running `voidline frontmatter validate` against a sample tree and confirming pass/fail, per-file errors, and expected exit code.

**Acceptance Scenarios**:

1. **Given** a directory of markdown files with invalid metadata, **When** I run `voidline frontmatter validate --root <dir> --schema personal`, **Then** the command reports file-level errors and exits non-zero.
2. **Given** valid frontmatter files, **When** I run the same command, **Then** the command reports success and exits zero.
3. **Given** `--output json`, **When** validation completes, **Then** output is machine-readable and stable for automation parsing.

---

### User Story 2 - Migrate/fix frontmatter safely (Priority: P2)

As a user, I can migrate frontmatter by applying defaults/generators/normalization with dry-run and backup options.

**Why this priority**: Safe migration is the next highest value after visibility because it turns diagnostics into actionable fixes.

**Independent Test**: Can be tested by running `migrate` in dry-run and write modes and verifying changed files and backups.

**Acceptance Scenarios**:

1. **Given** files missing required fields with generator/default rules, **When** I run `voidline frontmatter migrate --write`, **Then** files are updated according to schema strategy.
2. **Given** `--dry-run`, **When** migrate is executed, **Then** no files are modified and planned changes are reported.
3. **Given** `--backup`, **When** migrate writes files, **Then** backups are created before replacement.

---

### User Story 3 - Manage slug collisions for note creation (Priority: P3)

As an AI agent creating markdown notes, I can detect and resolve slug collisions using explicit policy rules before writing new files.

**Why this priority**: This is high leverage for autonomous agents but depends on the first two command capabilities.

**Independent Test**: Can be tested by preparing duplicate slugs and verifying detection and deterministic resolution policy behavior.

**Acceptance Scenarios**:

1. **Given** duplicate slugs in the selected scope, **When** I run `voidline frontmatter slug detect`, **Then** collisions are listed by file.
2. **Given** an existing slug and increment policy, **When** I run `voidline frontmatter slug resolve`, **Then** a non-colliding slug is returned or a bounded error when max attempts are exhausted.

---

### Edge Cases

- Empty directory or no markdown files found.
- Files with malformed YAML frontmatter.
- Files without frontmatter delimiters.
- Hidden files and excluded glob patterns.
- Unknown schema name or invalid config path.
- Permission errors while reading or writing files.
- Very large directory trees with `--max-files` cap reached.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST provide a `voidline frontmatter` command group with subcommands for `walk`, `validate`, `migrate`, and `slug` operations.
- **FR-002**: System MUST support markdown tree traversal with include/exclude globs, extension filtering, hidden-file behavior, and max-file limits.
- **FR-003**: System MUST support frontmatter validation against named schemas from config and report file-scoped errors.
- **FR-004**: System MUST support migration with `--dry-run`, optional `--write`, and optional `--backup` semantics.
- **FR-005**: System MUST support text and JSON output formats suitable for humans and machine consumers.
- **FR-006**: System MUST return deterministic non-zero exit codes when validation fails, command input is invalid, or runtime errors occur.
- **FR-007**: System MUST support slug collision detection with scopes (`directory`, `project`, `global`) and resolution policies (`fail`, `increment`, `append-uid`).
- **FR-008**: System MUST preserve existing frontmatter parsing semantics from `internal/frontmatter` library functions.
- **FR-009**: System MUST avoid modifying files unless explicitly requested via write mode.

### Key Entities *(include if feature involves data)*

- **FrontmatterCommandRequest**: CLI invocation context including root, schema, strategy, output format, and traversal options.
- **FileActionResult**: Per-file action status from validation/migration including path, changes, and errors.
- **SlugCollisionResult**: Collision report containing slug, path, and colliding paths.
- **CommandSummary**: Aggregate outcome counts and exit status for agent-friendly orchestration.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Running `voidline frontmatter validate` against a fixture set of at least 1,000 markdown files completes without panic/crash and returns a deterministic success/failure outcome.
- **SC-002**: For a fixed input directory and identical flags, JSON output remains stable across repeated runs, including deterministic ordering of result items and consistent required fields.
- **SC-003**: `--dry-run` mode performs zero file writes while still reporting proposed changes.
- **SC-004**: In sample collision datasets, slug resolution returns a valid non-colliding slug within configured attempt bounds.
