import axios from "axios";
import {
  AssetSchema,
  TelemetrySchema,
  AlertSchema,
  type Asset,
  type Telemetry,
  type Alert,
} from "./schemas";
import { z } from "zod";

const client = axios.create({
  baseURL: process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080",
  timeout: 10_000,
});

export async function fetchAssets(): Promise<Asset[]> {
  const { data } = await client.get("/assets");
  return z.array(AssetSchema).parse(data);
}

export async function fetchLastTelemetry(assetId: string): Promise<Telemetry> {
  const { data } = await client.get(`/assets/${assetId}/telemetry`);
  return TelemetrySchema.parse(data);
}

export async function fetchTelemetryHistory(
  assetId: string
): Promise<Telemetry[]> {
  const { data } = await client.get(`/assets/${assetId}/telemetry/history`);
  return z.array(TelemetrySchema).parse(data);
}

export async function fetchAlerts(assetId: string): Promise<Alert[]> {
  const { data } = await client.get(`/assets/${assetId}/alerts`);
  return z.array(AlertSchema).parse(data);
}
