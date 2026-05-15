"use client";

import useSWR from "swr";
import { fetchAssets, fetchLastTelemetry } from "@/lib/api";
import type { Asset, Telemetry } from "@/lib/schemas";

export function useAssets() {
  return useSWR<Asset[]>("assets", fetchAssets, { refreshInterval: 10_000 });
}

export function useLastTelemetry(assetId: string | null) {
  return useSWR<Telemetry>(
    assetId ? `telemetry/${assetId}` : null,
    () => fetchLastTelemetry(assetId!),
    { refreshInterval: 5_000 }
  );
}
