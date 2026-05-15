"use client";

import { Thermometer, Droplets, Lock, LockOpen } from "lucide-react";
import { Separator } from "@/components/ui/separator";
import type { Asset, Telemetry } from "@/lib/schemas";

interface Props {
  asset: Asset;
  telemetry: Telemetry;
}

interface StatItemProps {
  label: string;
  value: string;
  dim?: boolean;
}

function StatItem({ label, value, dim }: StatItemProps) {
  return (
    <div className="flex flex-col gap-0.5">
      <span className="text-[10px] font-medium uppercase tracking-widest text-zinc-500">
        {label}
      </span>
      <span
        className={`font-mono text-sm font-semibold ${dim ? "text-zinc-400" : "text-zinc-100"}`}
      >
        {value}
      </span>
    </div>
  );
}

export function StatsBar({ asset, telemetry }: Props) {
  const locked = telemetry.IsLocked;

  return (
    <div className="flex items-center gap-5 border-b border-zinc-800 bg-zinc-950 px-4 py-3 flex-wrap">
      <div className="flex items-center gap-2 text-zinc-400">
        <Thermometer size={14} />
        <span className="text-[10px] uppercase tracking-widest">Temp</span>
      </div>

      <StatItem label="Current" value={`${telemetry.Temperature.toFixed(1)}°C`} />
      <StatItem label="Max" value={`${asset.MaxTemperature.toFixed(1)}°C`} dim />
      <StatItem label="Min" value={`${asset.MinTemperature.toFixed(1)}°C`} dim />

      <Separator orientation="vertical" className="h-8 bg-zinc-800" />

      <div className="flex items-center gap-2 text-zinc-400">
        <Droplets size={14} />
        <span className="text-[10px] uppercase tracking-widest">Humidity</span>
      </div>

      <StatItem label="Current" value={`${telemetry.Humidity.toFixed(1)}%`} />
      <StatItem label="Max" value={`${asset.MaxHumidity.toFixed(1)}%`} dim />

      <Separator orientation="vertical" className="h-8 bg-zinc-800" />

      <div className="flex items-center gap-2">
        {locked ? (
          <Lock size={14} className="text-emerald-400" />
        ) : (
          <LockOpen size={14} className="text-red-400" />
        )}
        <span
          className={`text-xs font-mono font-semibold ${locked ? "text-emerald-400" : "text-red-400"}`}
        >
          {locked ? "LOCKED" : "UNLOCKED"}
        </span>
      </div>
    </div>
  );
}
