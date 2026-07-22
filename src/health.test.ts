import { describe, expect, it } from "vitest";

import { openWarehouseDatabase, warehouseDatabasePath } from "./db/database.js";
import { warehouseDataHealth } from "./health.js";
import { createTestWarehouse, processEnvironment, seedCalendar } from "../test/helpers.js";

describe("warehouseDataHealth", () => {
  it("fails before startup when the database path is unavailable", () => {
    expect(() => warehouseDatabasePath({})).toThrow("WAREHOUSE_DATABASE_PATH must be set.");
    expect(() => warehouseDatabasePath({ WAREHOUSE_DATABASE_PATH: "/missing/warehouse.db" })).toThrow(
      "Warehouse database does not exist",
    );
  });

  it("reports calendar readiness, counts, sources, and non-sensitive warnings", async () => {
    const fixture = await createTestWarehouse();
    seedCalendar(fixture.path);
    const warehouse = openWarehouseDatabase(processEnvironment(fixture.path));

    try {
      const health = await warehouseDataHealth(warehouse.db);
      expect(health).toMatchObject({
        database: { available: true },
        calendar: {
          ready: true,
          rawEventCount: 3,
          occurrenceCount: 3,
          importBatchCount: 1,
          latestImportAt: "2026-07-01T12:00:00.000Z",
        },
      });
      expect(health.calendar.sourceSystems).toEqual(
        expect.arrayContaining([
          { name: "google", count: 2 },
          { name: "apple", count: 1 },
        ]),
      );
    } finally {
      await warehouse.close();
      await fixture.close();
    }
  });

  it("reports a missing calendar schema without querying unavailable tables", async () => {
    const fixture = await createTestWarehouse({ calendarSchema: false });
    const warehouse = openWarehouseDatabase(processEnvironment(fixture.path));

    try {
      const health = await warehouseDataHealth(warehouse.db);
      expect(health.calendar.ready).toBe(false);
      expect(health.calendar.missingTables).toEqual([
        "calendar_events_raw",
        "calendar_event_occurrences",
        "cal_import_batches",
      ]);
    } finally {
      await warehouse.close();
      await fixture.close();
    }
  });
});
