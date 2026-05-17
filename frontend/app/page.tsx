"use client";

import { useState, useCallback, useEffect, useRef } from "react";
import dynamic from "next/dynamic";
import { Radio, AlertTriangle } from "lucide-react";
import { useAssets } from "@/hooks/useAssets";
import { ContainerPanel } from "@/components/ContainerPanel";
import type { Asset } from "@/lib/schemas";

const ShipMap = dynamic(
  () => import("@/components/ShipMap").then((m) => m.ShipMap),
  { ssr: false }
);

function useAllPositions(assets: Asset[]) {
  const [positions, setPositions] = useState<
    { assetId: string; lat: number; lon: number }[]
  >([]);
  const assetsRef = useRef(assets);
  assetsRef.current = assets;

  useEffect(() => {
    if (assets.length === 0) return;

    const doFetch = async () => {
      const results = await Promise.allSettled(
        assetsRef.current.map(async (asset) => {
          try {
            const res = await fetch(
              `${process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080"}/assets/${asset.ID}/telemetry`
            );
            if (!res.ok) return null;
            const data = await res.json();
            return { assetId: asset.ID, lat: data.Latitude, lon: data.Longitude };
          } catch {
            return null;
          }
        })
      );
      const valid = results
        .filter((r) => r.status === "fulfilled" && r.value !== null)
        .map(
          (r) =>
            (r as PromiseFulfilledResult<{ assetId: string; lat: number; lon: number }>).value
        );
      setPositions(valid);
    };

    doFetch();
    const id = setInterval(doFetch, 10_000);
    return () => clearInterval(id);
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [assets.length]);

  return positions;
}

export default function Home() {
  const { data: assets, error } = useAssets();
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const positions = useAllPositions(assets ?? []);

  const handleShipClick = useCallback((assetId: string) => {
    setSelectedId(assetId);
  }, []);

  const handleClose = useCallback(() => setSelectedId(null), []);

  return (
    <div className="flex flex-col h-screen bg-zinc-950 text-zinc-100">
      {/* Header */}
      <header className="flex items-center justify-between px-5 py-2.5 border-b border-zinc-800 bg-zinc-900 shrink-0">
        <div className="flex items-center gap-3">
          <Radio size={16} className="text-sky-400" />
          <span className="text-sm font-semibold tracking-tight text-zinc-100">
            Smart Tracking System
          </span>
          <span className="text-[10px] font-mono text-zinc-600 uppercase tracking-widest ml-2">
            Fleet Monitor
          </span>
        </div>

        <div className="flex items-center gap-4 text-xs text-zinc-500">
          {error && (
            <span className="flex items-center gap-1.5 text-red-400">
              <AlertTriangle size={12} />
              API unreachable
            </span>
          )}
          <span className="font-mono">
            {assets?.length ?? 0} containers tracked
          </span>
          <span className="flex items-center gap-1.5">
            <span className="h-1.5 w-1.5 rounded-full bg-emerald-500 animate-pulse" />
            Live
          </span>
        </div>
      </header>

      {/* Main */}
      <div className="flex flex-1 min-h-0">
        <ShipMap
          positions={positions}
          onShipClick={handleShipClick}
          selectedId={selectedId}
        />

        {selectedId !== null && assets && (
          <ContainerPanel
            assets={assets}
            selectedId={selectedId}
            onSelectId={setSelectedId}
            onClose={handleClose}
          />
        )}
      </div>
    </div>
  );
}
