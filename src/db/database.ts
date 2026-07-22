import { existsSync } from "node:fs";
import { resolve } from "node:path";

import Sqlite from "better-sqlite3";
import { Kysely, SqliteDialect } from "kysely";

import type { DB } from "./generated.js";

export class WarehouseDatabaseError extends Error {
  readonly code = "WAREHOUSE_UNAVAILABLE";

  constructor(message: string, options?: ErrorOptions) {
    super(message, options);
    this.name = "WarehouseDatabaseError";
  }
}

export type WarehouseDatabase = {
  db: Kysely<DB>;
  close: () => Promise<void>;
};

export function warehouseDatabasePath(environment: NodeJS.ProcessEnv = process.env): string {
  const configuredPath = environment.WAREHOUSE_DATABASE_PATH;
  if (!configuredPath) {
    throw new WarehouseDatabaseError("WAREHOUSE_DATABASE_PATH must be set.");
  }

  const databasePath = resolve(configuredPath);
  if (!existsSync(databasePath)) {
    throw new WarehouseDatabaseError(`Warehouse database does not exist at ${databasePath}.`);
  }

  return databasePath;
}

export function openWarehouseDatabase(
  environment: NodeJS.ProcessEnv = process.env,
): WarehouseDatabase {
  const databasePath = warehouseDatabasePath(environment);

  try {
    const sqlite = new Sqlite(databasePath, { fileMustExist: true, readonly: true });
    sqlite.pragma("query_only = ON");
    const db = new Kysely<DB>({ dialect: new SqliteDialect({ database: sqlite }) });

    return {
      db,
      close: () => db.destroy(),
    };
  } catch (error) {
    throw new WarehouseDatabaseError(`Unable to open Warehouse database at ${databasePath}.`, {
      cause: error,
    });
  }
}
