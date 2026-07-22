import { z } from "zod";

function isIsoDate(value: string): boolean {
  const parsed = new Date(`${value}T00:00:00.000Z`);
  return !Number.isNaN(parsed.valueOf()) && parsed.toISOString().slice(0, 10) === value;
}

const isoDateSchema = z
  .string()
  .regex(/^\d{4}-\d{2}-\d{2}$/, "Expected an ISO date (YYYY-MM-DD).")
  .refine(isIsoDate, "Invalid ISO date.");

const limitSchema = z.number().int().min(1).max(50);

export const evidenceSchema = z.object({
  recordId: z.number().int(),
  table: z.literal("calendar_event_occurrences"),
  rawEventId: z.number().int(),
  sourceSystem: z.string().nullable(),
  sourceFile: z.string().nullable(),
});

export const calendarEventSchema = z.object({
  id: z.number().int(),
  uid: z.string(),
  title: z.string().nullable(),
  startsAt: z.string(),
  endsAt: z.string().nullable(),
  occurrenceDate: z.string().nullable(),
  isAllDay: z.boolean(),
  isCancelled: z.boolean(),
  location: z.string().nullable(),
  evidence: evidenceSchema,
});

export const calendarSearchInputSchema = z
  .object({
    query: z.string().trim().min(1).max(200),
    from: isoDateSchema.optional(),
    to: isoDateSchema.optional(),
    includeCancelled: z.boolean().default(false),
    limit: limitSchema.default(20),
  })
  .superRefine((value, context) => {
    if (value.from && value.to && value.from > value.to) {
      context.addIssue({
        code: z.ZodIssueCode.custom,
        message: "from must be on or before to.",
        path: ["to"],
      });
    }
  });

export const calendarSearchOutputSchema = z.object({
  events: z.array(calendarEventSchema),
  count: z.number().int().min(0),
});

export const calendarUpcomingInputSchema = z.object({
  from: isoDateSchema.optional(),
  days: z.number().int().min(1).max(90).default(14),
  limit: limitSchema.default(20),
});

export const calendarUpcomingOutputSchema = z.object({
  events: z.array(calendarEventSchema),
  count: z.number().int().min(0),
});

export const warehouseDataHealthInputSchema = z.object({});

export const warehouseDataHealthOutputSchema = z.object({
  database: z.object({ available: z.literal(true) }),
  calendar: z.object({
    ready: z.boolean(),
    missingTables: z.array(z.string()),
    rawEventCount: z.number().int().min(0).nullable(),
    occurrenceCount: z.number().int().min(0).nullable(),
    importBatchCount: z.number().int().min(0).nullable(),
    latestImportAt: z.string().nullable(),
    sourceSystems: z.array(z.object({ name: z.string().nullable(), count: z.number().int().min(0) })),
    warnings: z.array(z.string()),
  }),
});

export type CalendarSearchInput = z.output<typeof calendarSearchInputSchema>;
export type CalendarSearchOutput = z.output<typeof calendarSearchOutputSchema>;
export type CalendarUpcomingInput = z.output<typeof calendarUpcomingInputSchema>;
export type CalendarUpcomingOutput = z.output<typeof calendarUpcomingOutputSchema>;
export type WarehouseDataHealthOutput = z.output<typeof warehouseDataHealthOutputSchema>;
