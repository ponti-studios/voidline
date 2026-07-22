import { mkdtemp, readFile, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join, resolve } from "node:path";

import Sqlite from "better-sqlite3";

const repositoryRoot = resolve(import.meta.dirname, "..");

function migrationUp(sql: string): string {
  const match = /-- \+goose StatementBegin\n([\s\S]*?)\n-- \+goose StatementEnd/.exec(sql);
  if (!match?.[1]) {
    throw new Error("Migration has no Goose Up statement block.");
  }
  return match[1];
}

export type TestWarehouse = {
  path: string;
  close: () => Promise<void>;
};

export async function createTestWarehouse(options: { calendarSchema?: boolean } = {}): Promise<TestWarehouse> {
  const directory = await mkdtemp(join(tmpdir(), "warehouse-mcp-"));
  const path = join(directory, "warehouse.db");
  const database = new Sqlite(path);
  const initialSchema = await readFile(join(repositoryRoot, "migrations/00001_initial_schema.sql"), "utf8");
  database.exec(migrationUp(initialSchema));

  if (options.calendarSchema ?? true) {
    const calendarSchema = await readFile(
      join(repositoryRoot, "migrations/00011_add_calendar_import_tables.sql"),
      "utf8",
    );
    database.exec(migrationUp(calendarSchema));
  }

  return {
    path,
    close: async () => {
      database.close();
      await rm(directory, { force: true, recursive: true });
    },
  };
}

export function seedCalendar(databasePath: string): void {
  const database = new Sqlite(databasePath);
  database
    .prepare(
      `INSERT INTO cal_import_batches (id, imported_at, file_count, event_count)
       VALUES (?, ?, ?, ?)`,
    )
    .run("batch-1", "2026-07-01T12:00:00.000Z", 1, 3);

  const insertRaw = database.prepare(
    `INSERT INTO calendar_events_raw (import_batch_id, source_system, source_file, uid, summary, description, location, raw)
     VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
  );
  const insertOccurrence = database.prepare(
    `INSERT INTO calendar_event_occurrences
       (raw_event_id, uid, occurrence_key, occurrence_start_utc, occurrence_end_utc, occurrence_date,
        is_all_day, is_cancelled, summary, description, location, expansion_version)
     VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
  );

  const visible = insertRaw.run(
    "batch-1",
    "google",
    "/private/calendar/team.ics",
    "visible-uid",
    "Project Maple",
    "private-note-only",
    "Studio",
    "BEGIN:VEVENT",
  );
  insertOccurrence.run(
    visible.lastInsertRowid,
    "visible-uid",
    "visible-2026",
    "2026-07-10T10:00:00.000Z",
    "2026-07-10T11:00:00.000Z",
    null,
    0,
    0,
    "Project Maple",
    "private-note-only",
    "Studio",
    "fixture",
  );

  const percent = insertRaw.run(
    "batch-1",
    "apple",
    "/private/calendar/budget.ics",
    "percent-uid",
    "Budget 100%",
    null,
    null,
    "BEGIN:VEVENT",
  );
  insertOccurrence.run(
    percent.lastInsertRowid,
    "percent-uid",
    "percent-2026",
    "2026-07-11T10:00:00.000Z",
    "2026-07-11T11:00:00.000Z",
    null,
    0,
    0,
    "Budget 100%",
    null,
    null,
    "fixture",
  );

  const cancelled = insertRaw.run(
    "batch-1",
    "google",
    "/private/calendar/cancelled.ics",
    "cancelled-uid",
    "Cancelled Project Maple",
    null,
    null,
    "BEGIN:VEVENT",
  );
  insertOccurrence.run(
    cancelled.lastInsertRowid,
    "cancelled-uid",
    "cancelled-2026",
    "2026-07-12T10:00:00.000Z",
    "2026-07-12T11:00:00.000Z",
    null,
    0,
    1,
    "Cancelled Project Maple",
    null,
    null,
    "fixture",
  );
  database.close();
}

export function processEnvironment(databasePath: string): Record<string, string> {
  const environment: Record<string, string> = {};
  for (const [key, value] of Object.entries(process.env)) {
    if (value !== undefined) {
      environment[key] = value;
    }
  }
  environment.WAREHOUSE_DATABASE_PATH = databasePath;
  return environment;
}
