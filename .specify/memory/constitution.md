# Voidline Constitution

## Core Principles

### I. Reuse-First Architecture
All feature work MUST reuse existing domain and application packages when suitable behavior already exists. New abstractions are allowed only when required to avoid coupling or preserve clear boundaries, and that decision MUST be documented in the plan.

### II. Deterministic CLI Contracts
Every CLI command added to this repository MUST provide deterministic behavior for both humans and automation: stable flag semantics, explicit exit codes, and machine-readable output when JSON mode is offered.

### III. Verified Behavior Changes
Code that changes command behavior, parsing, output shape, migration safety, or policy decisions MUST include tests that verify expected behavior and failure modes. Tests SHOULD be targeted and table-driven when practical.

### IV. Safety Before Mutation
Any feature that mutates user files or data MUST default to non-destructive operation and require explicit opt-in for writes. Risk-reducing options (for example backup generation) MUST be provided when destructive operations are supported.

### V. Incremental Delivery
Work MUST be organized into independently verifiable slices that provide value in priority order. Plans and task lists SHOULD identify MVP scope, story checkpoints, and dependencies clearly.

## Additional Constraints

- Existing repository conventions and tooling MUST be followed unless a plan explicitly records and justifies a deviation.
- Public-facing command output formats and exit-code semantics MUST be documented in feature artifacts when introduced or changed.
- Performance-sensitive requirements in specifications MUST be translated into concrete verification tasks.

## Development Workflow

- `/speckit.specify`, `/speckit.plan`, and `/speckit.tasks` outputs MUST remain internally consistent before implementation starts.
- Constitution checks in plans are blocking gates; unresolved placeholder governance is not acceptable for implementation readiness.
- Any CRITICAL analysis findings MUST be resolved or explicitly deferred with rationale before `/speckit.implement`.

## Governance

- This constitution supersedes conflicting workflow guidance for planning and implementation quality gates.
- Amendments require a documented rationale, explicit version update, and synchronization of affected templates or prompts.
- Reviews and planning analysis MUST verify compliance with these principles.

**Version**: 1.0.0 | **Ratified**: 2026-02-20 | **Last Amended**: 2026-02-20
