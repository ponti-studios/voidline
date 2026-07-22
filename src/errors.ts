import { ZodError } from "zod";

export class WarehouseToolError extends Error {
  constructor(
    readonly code: "INVALID_ARGUMENT" | "SCHEMA_UNAVAILABLE" | "WAREHOUSE_UNAVAILABLE",
    message: string,
    options?: ErrorOptions,
  ) {
    super(message, options);
    this.name = "WarehouseToolError";
  }
}

export function toolError(error: unknown): { content: Array<{ type: "text"; text: string }>; isError: true } {
  if (error instanceof ZodError) {
    return {
      content: [
        {
          type: "text",
          text: JSON.stringify({ code: "INVALID_ARGUMENT", message: "Invalid tool input.", details: error.flatten() }),
        },
      ],
      isError: true,
    };
  }

  if (error instanceof WarehouseToolError) {
    return {
      content: [{ type: "text", text: JSON.stringify({ code: error.code, message: error.message }) }],
      isError: true,
    };
  }

  return {
    content: [
      {
        type: "text",
        text: JSON.stringify({ code: "WAREHOUSE_UNAVAILABLE", message: "Warehouse query failed." }),
      },
    ],
    isError: true,
  };
}
