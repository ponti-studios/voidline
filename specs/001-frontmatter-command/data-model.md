# Data Model: Frontmatter CLI Command

## 1) FrontmatterCommandRequest
Represents normalized command input after Cobra flag parsing.

### Fields
- `action` (string, required): `walk | validate | migrate | slug-detect | slug-resolve`
- `root` (string, required for walk/validate/migrate/slug-detect): filesystem root directory
- `schema` (string, optional): schema name for validation/migration
- `configPath` (string, optional): explicit config file path
- `output` (string, required): `text | json`
- `includeHidden` (bool, optional, default false)
- `extensions` ([]string, optional, default `.md,.markdown`)
- `includeGlobs` ([]string, optional)
- `excludeGlobs` ([]string, optional)
- `maxFiles` (int, optional, default 0 meaning unlimited)
- `write` (bool, optional, default false)
- `backup` (bool, optional, default false)
- `strategy` (string, optional): migration strategy name
- `slug` (string, required for slug-resolve)
- `scope` (string, optional, default `directory`): `directory | project | global`
- `policy` (string, optional, default `increment`): `fail | increment | append-uid`
- `maxAttempts` (int, optional, default from config)

### Validation Rules
- `root` must exist and be a directory where required.
- `output` must be one of `text|json`.
- `scope` and `policy` must be from allowed enumerations.
- `maxFiles >= 0`, `maxAttempts >= 1` when used.
- `write=false` forbids file mutation.

## 2) FileActionResult
Per-file action status; maps directly to `internal/frontmatter.FileAction` with CLI-safe serialization.

### Fields
- `path` (string): file path
- `hasChanges` (bool): whether migration changed content
- `errors` ([]string): validation or processing errors for this file
- `result` (object): migration/validation result details
  - `hasFrontmatter` (bool)
  - `validationBefore` (object)
  - `validationAfter` (object)
  - `hasChanges` (bool)

### State Transitions
- `pending -> processed-success`
- `pending -> processed-error`
- `processed-success` may include `hasChanges=true|false`

## 3) SlugCollisionResult
Collision detail for one file/slug context.

### Fields
- `slug` (string)
- `path` (string)
- `collisions` ([]string): other paths with same slug in scope

### Validation Rules
- `slug` non-empty
- `path` non-empty

## 4) CommandSummary
Aggregated run outcome for humans and agents.

### Fields
- `command` (string)
- `root` (string)
- `totalFiles` (int)
- `processedFiles` (int)
- `changedFiles` (int)
- `errorFiles` (int)
- `warnings` ([]string)
- `durationMs` (int64)
- `exitCode` (int)

### Derived Rules
- `exitCode=0` when `errorFiles=0` and no hard failures.
- `exitCode=1` when domain validation/collision failures present.
- `exitCode=2` on usage/runtime/IO/config failures.
