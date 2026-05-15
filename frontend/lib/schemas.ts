import { z } from "zod";

export const AssetStatusSchema = z.enum([
  "new",
  "active",
  "maintenance",
  "decommissioned",
]);

export const AssetSchema = z.object({
  ID: z.string(),
  Name: z.string(),
  MaxTemperature: z.number(),
  MinTemperature: z.number(),
  MaxHumidity: z.number(),
  Status: AssetStatusSchema,
  CreatedAt: z.string(),
});

export const TelemetrySchema = z.object({
  AssetID: z.string(),
  Latitude: z.number(),
  Longitude: z.number(),
  Temperature: z.number(),
  Humidity: z.number(),
  IsLocked: z.boolean(),
  Timestamp: z.string(),
  TraceID: z.string(),
});

export const AlertTypeSchema = z.enum([
  "temperature_exceeded_max_limit",
  "temperature_exceeded_min_limit",
  "humidity_exceeded_max_limit",
  "container_unlocked",
  "maintenance_required",
]);

export const AlertSchema = z.object({
  ID: z.string(),
  AssetID: z.string(),
  Type: AlertTypeSchema,
  Message: z.string(),
  CreatedAt: z.string(),
});

export type Asset = z.infer<typeof AssetSchema>;
export type Telemetry = z.infer<typeof TelemetrySchema>;
export type Alert = z.infer<typeof AlertSchema>;
export type AssetStatus = z.infer<typeof AssetStatusSchema>;
