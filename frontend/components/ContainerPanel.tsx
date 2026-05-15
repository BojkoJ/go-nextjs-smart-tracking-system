"use client";

import { X, Package } from "lucide-react";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { StatsBar } from "./StatsBar";
import { TelemetryLog } from "./TelemetryLog";
import { useLastTelemetry } from "@/hooks/useAssets";
import { useTelemetryHistory } from "@/hooks/useContainerDetail";
import type { Asset, AssetStatus } from "@/lib/schemas";

interface Props {
  assets: Asset[];
  selectedId: string;
  onSelectId: (id: string) => void;
  onClose: () => void;
}

const STATUS_DOT: Record<AssetStatus, string> = {
  active: "bg-emerald-500",
  new: "bg-blue-500",
  maintenance: "bg-amber-500",
  decommissioned: "bg-zinc-600",
};

export function ContainerPanel({ assets, selectedId, onSelectId, onClose }: Props) {
  const selectedAsset = assets.find((a) => a.ID === selectedId);
  const { data: telemetry } = useLastTelemetry(selectedId);
  const { data: history } = useTelemetryHistory(selectedId);

  return (
    <div className="flex flex-col h-full bg-zinc-950 border-l border-zinc-800 w-[480px] shrink-0">
      {/* Panel header */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-zinc-800 bg-zinc-900">
        <div className="flex items-center gap-2 text-zinc-400">
          <Package size={14} />
          <span className="text-[10px] font-medium uppercase tracking-widest">
            Container
          </span>
        </div>

        <div className="flex items-center gap-3 flex-1 mx-4">
          <Select value={selectedId} onValueChange={onSelectId}>
            <SelectTrigger className="h-7 text-xs font-mono bg-zinc-800 border-zinc-700 text-zinc-200 focus:ring-0 focus:ring-offset-0">
              <SelectValue />
            </SelectTrigger>
            <SelectContent className="bg-zinc-900 border-zinc-700 max-h-64">
              {assets.map((asset) => (
                <SelectItem
                  key={asset.ID}
                  value={asset.ID}
                  className="text-xs font-mono text-zinc-300 focus:bg-zinc-800 focus:text-zinc-100"
                >
                  <span className="flex items-center gap-2">
                    <span
                      className={`h-1.5 w-1.5 rounded-full shrink-0 ${STATUS_DOT[asset.Status]}`}
                    />
                    {asset.Name}
                  </span>
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        <button
          onClick={onClose}
          className="text-zinc-600 hover:text-zinc-300 transition-colors"
        >
          <X size={16} />
        </button>
      </div>

      {/* Stats bar */}
      {selectedAsset && telemetry ? (
        <StatsBar asset={selectedAsset} telemetry={telemetry} />
      ) : (
        <div className="h-14 border-b border-zinc-800 bg-zinc-950 flex items-center px-4">
          <span className="text-xs text-zinc-600 font-mono">Loading telemetry...</span>
        </div>
      )}

      {/* Log header */}
      <div className="flex items-center justify-between px-4 py-2 border-b border-zinc-800">
        <span className="text-[10px] font-medium uppercase tracking-widest text-zinc-500">
          Telemetry Log
        </span>
        <span className="text-[10px] font-mono text-zinc-600">
          {history?.length ?? 0} entries — newest first
        </span>
      </div>

      {/* Telemetry log */}
      <TelemetryLog history={history ?? []} />
    </div>
  );
}
