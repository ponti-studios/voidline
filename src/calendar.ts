import { basename } from "node:path";

import { sql, type Kysely } from "kysely";

import {
  calendarSearchOutputSchema,
  calendarUpcomingOutputSchema,
  type CalendarSearchInput,
  type CalendarSearchOutput,
  type CalendarUpcomingInput,
  type CalendarUpcomingOutput,
} from "./contracts.js";
import type { DB } from "./db/generated.js";
import { WarehouseToolError } from "./errors.js";

const requiredCalendarTables = [
  "calendar_events_raw",
  "calendar_event_occurrences",
  "cal_import_batches",
] as const;

type CalendarRow = {
  id: number | null;
  rawEventId: number;
  uid: string;
  startsAt: string;
  endsAt: string | null;
  occurrenceDate: string | null;
  isAllDay: number;
  isCancelled: number;
  title: string | null;
  location: string | null;
  sourceSystem: string | null;
  sourceFile: string | null;
};

function escapeLike(value: string): string {
  return value.replaceAll("\\", "\\\\").replaceAll("%", "\\%").replaceAll("_", "\\_");
}

function endOfDay(value: string): string {
  return `${value}T23:59:59.999Z`;
}

function requireRecordId(value: number | null): number {
  if (value === null) {
    throw new WarehouseToolError("SCHEMA_UNAVAILABLE", "Calendar occurrence is missing its record ID.");
  }
  return value;
}

function mapCalendarRow(row: CalendarRow) {
  const id = requireRecordId(row.id);
  return {
    id,
    uid: row.uid,
    title: row.title,
    startsAt: row.startsAt,
    endsAt: row.endsAt,
    occurrenceDate: row.occurrenceDate,
    isAllDay: row.isAllDay === 1,
    isCancelled: row.isCancelled === 1,
    location: row.location,
    evidence: {
      recordId: id,
      table: "calendar_event_occurrences" as const,
      rawEventId: row.rawEventId,
      sourceSystem: row.sourceSystem,
      sourceFile: row.sourceFile ? basename(row.sourceFile) : null,
    },
  };
}

async function assertCalendarSchema(db: Kysely<DB>): Promise<void> {
  const existingTables = new Set((await db.introspection.getTables()).map((table) => table.name));
  const missingTables = requiredCalendarTables.filter((table) => !existingTables.has(table));

  if (missingTables.length > 0) {
    throw new WarehouseToolError(
      "SCHEMA_UNAVAILABLE",
      `Warehouse calendar schema is unavailable; missing ${missingTables.join(", ")}.`,
    );
  }
}

function eventQuery(db: Kysely<DB>) {
  return db
    .selectFrom("calendar_event_occurrences as occurrence")
    .innerJoin("calendar_events_raw as raw", "raw.id", "occurrence.raw_event_id")
    .select([
      "occurrence.id as id",
      "occurrence.raw_event_id as rawEventId",
      "occurrence.uid as uid",
      "occurrence.occurrence_start_utc as startsAt",
      "occurrence.occurrence_end_utc as endsAt",
      "occurrence.occurrence_date as occurrenceDate",
      "occurrence.is_all_day as isAllDay",
      "occurrence.is_cancelled as isCancelled",
      "occurrence.summary as title",
      "occurrence.location as location",
      "raw.source_system as sourceSystem",
      "raw.source_file as sourceFile",
    ]);
}

export async function searchCalendar(
  db: Kysely<DB>,
  input: CalendarSearchInput,
): Promise<CalendarSearchOutput> {
  await assertCalendarSchema(db);
  const pattern = `%${escapeLike(input.query.toLowerCase())}%`;
  const literalSearch = sql<boolean>`(
    lower(coalesce(occurrence.summary, '')) like ${pattern} escape '\\'
    or lower(coalesce(occurrence.description, '')) like ${pattern} escape '\\'
    or lower(coalesce(occurrence.location, '')) like ${pattern} escape '\\'
  )`;

  let query = eventQuery(db).where(literalSearch);
  if (!input.includeCancelled) {
    query = query.where("occurrence.is_cancelled", "=", 0);
  }
  if (input.from) {
    query = query.where("occurrence.occurrence_start_utc", ">=", input.from);
  }
  if (input.to) {
    query = query.where("occurrence.occurrence_start_utc", "<=", endOfDay(input.to));
  }

  const events = (await query.orderBy("occurrence.occurrence_start_utc", "asc").limit(input.limit).execute()).map(
    mapCalendarRow,
  );
  return calendarSearchOutputSchema.parse({ events, count: events.length });
}

export async function upcomingCalendar(
  db: Kysely<DB>,
  input: CalendarUpcomingInput,
  now: Date = new Date(),
): Promise<CalendarUpcomingOutput> {
  await assertCalendarSchema(db);
  const start = input.from ? `${input.from}T00:00:00.000Z` : now.toISOString();
  const until = new Date(Date.parse(start) + input.days * 86_400_000).toISOString();
  const events = (
    await eventQuery(db)
      .where("occurrence.is_cancelled", "=", 0)
      .where("occurrence.occurrence_start_utc", ">=", start)
      .where("occurrence.occurrence_start_utc", "<=", until)
      .orderBy("occurrence.occurrence_start_utc", "asc")
      .limit(input.limit)
      .execute()
  ).map(mapCalendarRow);

  return calendarUpcomingOutputSchema.parse({ events, count: events.length });
}

export { requiredCalendarTables };
