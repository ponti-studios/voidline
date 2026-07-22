import type { Kysely } from "kysely";

import { requiredCalendarTables } from "./calendar.js";
import { warehouseDataHealthOutputSchema, type WarehouseDataHealthOutput } from "./contracts.js";
import type { DB } from "./db/generated.js";

export async function warehouseDataHealth(db: Kysely<DB>): Promise<WarehouseDataHealthOutput> {
  const existingTables = new Set((await db.introspection.getTables()).map((table) => table.name));
  const missingTables = requiredCalendarTables.filter((table) => !existingTables.has(table));

  if (missingTables.length > 0) {
    return warehouseDataHealthOutputSchema.parse({
      database: { available: true },
      calendar: {
        ready: false,
        missingTables,
        rawEventCount: null,
        occurrenceCount: null,
        importBatchCount: null,
        latestImportAt: null,
        sourceSystems: [],
        warnings: ["Calendar tools are unavailable until the required calendar migration is applied."],
      },
    });
  }

  const [rawEvents, occurrences, batches, latestBatch, sourceSystems] = await Promise.all([
    db.selectFrom("calendar_events_raw").select((builder) => builder.fn.countAll<number>().as("count")).executeTakeFirstOrThrow(),
    db.selectFrom("calendar_event_occurrences").select((builder) => builder.fn.countAll<number>().as("count")).executeTakeFirstOrThrow(),
    db.selectFrom("cal_import_batches").select((builder) => builder.fn.countAll<number>().as("count")).executeTakeFirstOrThrow(),
    db.selectFrom("cal_import_batches").select("imported_at as importedAt").orderBy("imported_at", "desc").limit(1).executeTakeFirst(),
    db
      .selectFrom("calendar_events_raw")
      .select(["source_system as name", (builder) => builder.fn.countAll<number>().as("count")])
      .groupBy("source_system")
      .orderBy("count", "desc")
      .execute(),
  ]);

  const warnings: string[] = [];
  if (Number(rawEvents.count) === 0) {
    warnings.push("No calendar events have been imported.");
  }
  if (Number(occurrences.count) === 0 && Number(rawEvents.count) > 0) {
    warnings.push("Calendar raw events exist but no expanded occurrences are available.");
  }

  return warehouseDataHealthOutputSchema.parse({
    database: { available: true },
    calendar: {
      ready: true,
      missingTables: [],
      rawEventCount: Number(rawEvents.count),
      occurrenceCount: Number(occurrences.count),
      importBatchCount: Number(batches.count),
      latestImportAt: latestBatch?.importedAt ?? null,
      sourceSystems: sourceSystems.map((source) => ({ name: source.name, count: Number(source.count) })),
      warnings,
    },
  });
}
