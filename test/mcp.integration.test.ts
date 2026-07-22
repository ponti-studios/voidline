import { resolve } from "node:path";

import { Client } from "@modelcontextprotocol/sdk/client/index.js";
import { StdioClientTransport } from "@modelcontextprotocol/sdk/client/stdio.js";
import { CallToolResultSchema } from "@modelcontextprotocol/sdk/types.js";
import { describe, expect, it } from "vitest";

import { createTestWarehouse, processEnvironment, seedCalendar } from "./helpers.js";

describe("Warehouse MCP stdio server", () => {
  it("discovers and invokes the metadata-only calendar tools over MCP", async () => {
    const fixture = await createTestWarehouse();
    seedCalendar(fixture.path);
    const client = new Client({ name: "warehouse-mcp-test", version: "1.0.0" });
    const transport = new StdioClientTransport({
      command: process.execPath,
      args: [resolve(process.cwd(), "dist/index.js")],
      env: processEnvironment(fixture.path),
      stderr: "pipe",
    });

    try {
      await client.connect(transport);
      const tools = await client.listTools();
      expect(tools.tools.map((tool) => tool.name).sort()).toEqual([
        "calendar_search",
        "calendar_upcoming",
        "warehouse_data_health",
      ]);

      const response = await client.callTool({
        name: "calendar_search",
        arguments: { query: "private-note-only" },
      }, CallToolResultSchema);
      const result = CallToolResultSchema.parse(response);
      const content = result.content[0];
      expect(content?.type).toBe("text");
      if (content?.type !== "text") {
        throw new Error("calendar_search returned non-text content.");
      }
      const parsed: unknown = JSON.parse(content.text);
      expect(parsed).toMatchObject({
        count: 1,
        events: [
          {
            title: "Project Maple",
            evidence: { sourceFile: "team.ics", sourceSystem: "google" },
          },
        ],
      });
      expect(JSON.stringify(parsed)).not.toContain("private-note-only");
    } finally {
      await client.close();
      await fixture.close();
    }
  });

  it("returns a stable schema error when calendar tables are unavailable", async () => {
    const fixture = await createTestWarehouse({ calendarSchema: false });
    const client = new Client({ name: "warehouse-mcp-test", version: "1.0.0" });
    const transport = new StdioClientTransport({
      command: process.execPath,
      args: [resolve(process.cwd(), "dist/index.js")],
      env: processEnvironment(fixture.path),
      stderr: "pipe",
    });

    try {
      await client.connect(transport);
      const response = await client.callTool(
        { name: "calendar_search", arguments: { query: "Maple" } },
        CallToolResultSchema,
      );
      const result = CallToolResultSchema.parse(response);
      const content = result.content[0];
      expect(result.isError).toBe(true);
      if (content?.type !== "text") {
        throw new Error("calendar_search returned non-text content.");
      }
      expect(JSON.parse(content.text)).toMatchObject({ code: "SCHEMA_UNAVAILABLE" });
    } finally {
      await client.close();
      await fixture.close();
    }
  });
});
