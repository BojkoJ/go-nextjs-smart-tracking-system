"use client";

import useSWR from "swr";
import { fetchTelemetryHistory, fetchAlerts } from "@/lib/api";
import type { Telemetry, Alert } from "@/lib/schemas";

export function useTelemetryHistory(assetId: string | null, page: number = 0) {
  return useSWR<Telemetry[]>(
    assetId ? `history/${assetId}/${page}` : null,
    () => fetchTelemetryHistory(assetId!, page),
    { refreshInterval: 8_000 }
  );
}

export function useAlerts(assetId: string | null) {
  return useSWR<Alert[]>(
    assetId ? `alerts/${assetId}` : null,
    () => fetchAlerts(assetId!),
    { refreshInterval: 15_000 }
  );
}
