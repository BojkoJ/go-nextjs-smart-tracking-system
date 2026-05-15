"use client";

import { ScrollArea } from "@/components/ui/scroll-area";
import type { Telemetry } from "@/lib/schemas";

interface Props {
  history: Telemetry[];
}

function formatTs(iso: string): string {
  return new Date(iso).toISOString().replace("T", " ").slice(0, 19) + "Z";
}

export function TelemetryLog({ history }: Props) {
  const sorted = [...history].sort(
    (a, b) => new Date(b.Timestamp).getTime() - new Date(a.Timestamp).getTime()
  );

  if (sorted.length === 0) {
    return (
      <div className="flex flex-1 items-center justify-center text-xs text-zinc-600 font-mono">
        No telemetry data
      </div>
    );
  }

  return (
    <ScrollArea className="flex-1 min-h-0">
      <div className="p-4 space-y-0">
        {sorted.map((entry, i) => (
          <div
            key={`${entry.Timestamp}-${i}`}
            className="flex gap-3 py-1.5 border-b border-zinc-900 last:border-0"
          >
            <span className="font-mono text-[11px] text-zinc-600 shrink-0 pt-px">
              {formatTs(entry.Timestamp)}
            </span>
            <span className="font-mono text-[11px] text-zinc-400 leading-relaxed">
              temp=
              <span className="text-zinc-200">{entry.Temperature.toFixed(2)}°C</span>
              {"  "}
              humid=
              <span className="text-zinc-200">{entry.Humidity.toFixed(2)}%</span>
              {"  "}
              lock=
              <span className={entry.IsLocked ? "text-emerald-400" : "text-red-400"}>
                {entry.IsLocked ? "locked" : "unlocked"}
              </span>
              {"  "}
              lat=
              <span className="text-zinc-300">{entry.Latitude.toFixed(4)}</span>
              {"  "}
              lon=
              <span className="text-zinc-300">{entry.Longitude.toFixed(4)}</span>
            </span>
          </div>
        ))}
      </div>
    </ScrollArea>
  );
}
