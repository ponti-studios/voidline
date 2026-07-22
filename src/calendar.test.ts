import { describe, expect, it } from "vitest";

import { searchCalendar, upcomingCalendar } from "./calendar.js";
import { calendarSearchInputSchema } from "./contracts.js";
import { openWarehouseDatabase } from "./db/database.js";
import { createTestWarehouse, processEnvironment, seedCalendar } from "../test/helpers.js";

describe("calendar queries", () => {
  it("rejects invalid calendar dates and reversed date ranges", () => {
    expect(
      calendarSearchInputSchema.safeParse({
        query: "Maple",
        from: "2026-02-30",
      }).success,
    ).toBe(false);
    expect(
      calendarSearchInputSchema.safeParse({
        query: "Maple",
        from: "2026-07-11",
        to: "2026-07-10",
      }).success,
    ).toBe(false);
  });

  it("searches descriptions without returning private event-body fields", async () => {
    const fixture = await createTestWarehouse();
    seedCalendar(fixture.path);
    const warehouse = openWarehouseDatabase(processEnvironment(fixture.path));

    try {
      const result = await searchCalendar(warehouse.db, {
        query: "private-note-only",
        includeCancelled: false,
        limit: 20,
      });

      expect(result.count).toBe(1);
      expect(result.events[0]).toMatchObject({
        title: "Project Maple",
        location: "Studio",
        evidence: {
          table: "calendar_event_occurrences",
          sourceFile: "team.ics",
          sourceSystem: "google",
        },
      });
      expect(result.events[0]).not.toHaveProperty("description");
      expect(JSON.stringify(result)).not.toContain("private-note-only");
      expect(JSON.stringify(result)).not.toContain("/private/calendar");
    } finally {
      await warehouse.close();
      await fixture.close();
    }
  });

  it("treats LIKE wildcards as literal search characters", async () => {
    const fixture = await createTestWarehouse();
    seedCalendar(fixture.path);
    const warehouse = openWarehouseDatabase(processEnvironment(fixture.path));

    try {
      const result = await searchCalendar(warehouse.db, {
        query: "100%",
        includeCancelled: false,
        limit: 20,
      });

      expect(result.events).toHaveLength(1);
      expect(result.events[0]?.title).toBe("Budget 100%");
    } finally {
      await warehouse.close();
      await fixture.close();
    }
  });

  it("filters cancelled events and returns upcoming events in start-time order", async () => {
    const fixture = await createTestWarehouse();
    seedCalendar(fixture.path);
    const warehouse = openWarehouseDatabase(processEnvironment(fixture.path));

    try {
      const search = await searchCalendar(warehouse.db, {
        query: "Project Maple",
        includeCancelled: false,
        limit: 20,
      });
      const upcoming = await upcomingCalendar(
        warehouse.db,
        { from: "2026-07-10", days: 3, limit: 20 },
        new Date("2026-01-01T00:00:00.000Z"),
      );

      expect(search.events).toHaveLength(1);
      expect(upcoming.events.map((event) => event.startsAt)).toEqual([
        "2026-07-10T10:00:00.000Z",
        "2026-07-11T10:00:00.000Z",
      ]);
    } finally {
      await warehouse.close();
      await fixture.close();
    }
  });

  it("enforces the read-only database boundary", async () => {
    const fixture = await createTestWarehouse();
    seedCalendar(fixture.path);
    const warehouse = openWarehouseDatabase(processEnvironment(fixture.path));

    try {
      await expect(
        warehouse.db
          .updateTable("calendar_event_occurrences")
          .set({ summary: "mutated" })
          .where("id", "=", 1)
          .execute(),
      ).rejects.toThrow();
      const result = await searchCalendar(warehouse.db, {
        query: "Project Maple",
        includeCancelled: false,
        limit: 20,
      });
      expect(result.events[0]?.title).toBe("Project Maple");
    } finally {
      await warehouse.close();
      await fixture.close();
    }
  });
});
