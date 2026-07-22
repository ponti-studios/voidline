import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";

import { searchCalendar, upcomingCalendar } from "./calendar.js";
import {
  calendarSearchInputSchema,
  calendarSearchOutputSchema,
  calendarUpcomingInputSchema,
  calendarUpcomingOutputSchema,
  warehouseDataHealthInputSchema,
  warehouseDataHealthOutputSchema,
} from "./contracts.js";
import { openWarehouseDatabase } from "./db/database.js";
import { toolError } from "./errors.js";
import { warehouseDataHealth } from "./health.js";

function toolResult<T extends Record<string, unknown>>(value: T) {
  return {
    content: [{ type: "text" as const, text: JSON.stringify(value) }],
    structuredContent: value,
  };
}

export function createWarehouseMcpServer(environment: NodeJS.ProcessEnv = process.env) {
  const warehouse = openWarehouseDatabase(environment);
  const server = new McpServer({ name: "warehouse-mcp", version: "1.0.0" });

  server.registerTool(
    "calendar_search",
    {
      description: "Search calendar occurrence titles, descriptions, and locations with bounded, literal matching. Returns metadata only.",
      inputSchema: calendarSearchInputSchema,
      outputSchema: calendarSearchOutputSchema,
    },
    async (input) => {
      try {
        const parsedInput = calendarSearchInputSchema.parse(input);
        return toolResult(await searchCalendar(warehouse.db, parsedInput));
      } catch (error) {
        return toolError(error);
      }
    },
  );

  server.registerTool(
    "calendar_upcoming",
    {
      description: "List upcoming non-cancelled calendar occurrences in a bounded time window. Returns metadata only.",
      inputSchema: calendarUpcomingInputSchema,
      outputSchema: calendarUpcomingOutputSchema,
    },
    async (input) => {
      try {
        const parsedInput = calendarUpcomingInputSchema.parse(input);
        return toolResult(await upcomingCalendar(warehouse.db, parsedInput));
      } catch (error) {
        return toolError(error);
      }
    },
  );

  server.registerTool(
    "warehouse_data_health",
    {
      description: "Report Warehouse calendar schema readiness and non-sensitive data coverage.",
      inputSchema: warehouseDataHealthInputSchema,
      outputSchema: warehouseDataHealthOutputSchema,
    },
    async (input) => {
      try {
        warehouseDataHealthInputSchema.parse(input);
        return toolResult(await warehouseDataHealth(warehouse.db));
      } catch (error) {
        return toolError(error);
      }
    },
  );

  return { server, close: warehouse.close };
}
