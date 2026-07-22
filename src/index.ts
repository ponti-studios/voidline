import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";

import { createWarehouseMcpServer } from "./server.js";

async function main(): Promise<void> {
  const { server } = createWarehouseMcpServer();
  await server.connect(new StdioServerTransport());
}

main().catch((error: unknown) => {
  const message = error instanceof Error ? error.message : "Unable to start Warehouse MCP server.";
  console.error(message);
  process.exitCode = 1;
});
