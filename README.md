# Warehouse MCP

Warehouse is a local, read-only Model Context Protocol server for the Warehouse
SQLite database. It gives MCP-capable LLM hosts structured calendar context and
data-health information without exposing raw SQL, filesystem access, imports,
or database mutations.

## Requirements

- Node.js 24.14 or newer
- pnpm 11.1 or newer
- An existing Warehouse SQLite database with the calendar import migration applied

## Setup

```bash
pnpm install
export WAREHOUSE_DATABASE_PATH="$HOME/.hominem/warehouse.db"
pnpm build
```

The server requires `WAREHOUSE_DATABASE_PATH`; it opens that file in SQLite
read-only mode and never applies migrations.

## Connect an MCP host

Build first, then configure a local stdio server in your MCP-capable host:

```json
{
  "command": "node",
  "args": ["/absolute/path/to/warehouse/dist/index.js"],
  "env": {
    "WAREHOUSE_DATABASE_PATH": "/absolute/path/to/warehouse.db"
  }
}
```

`pnpm start` launches the same server from a terminal. Diagnostics are written
to stderr; stdout is reserved for MCP JSON-RPC.

## Available tools

- `calendar_search` — bounded literal search of calendar titles, descriptions,
  and locations. Results exclude descriptions and all other event-body fields.
- `calendar_upcoming` — bounded list of non-cancelled upcoming occurrences.
- `warehouse_data_health` — schema readiness and non-sensitive calendar coverage.

Every calendar result includes stable occurrence/raw-event IDs plus source system
and source-file basename evidence. The server never returns raw ICS, attendee,
organizer, description, or absolute source-path data.

## Development

```bash
pnpm generate:types # requires WAREHOUSE_DATABASE_PATH
pnpm typecheck
pnpm lint
pnpm test
```

`src/db/generated.ts` is generated from a real Warehouse database with
`kysely-codegen`; do not edit it manually. Regenerate it after Warehouse schema
changes.

SQL migrations remain the canonical database contract in [`migrations/`](migrations).
