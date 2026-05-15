"use client";

import { ScrollArea } from "@/components/ui/scroll-area";
import { ChevronLeft, ChevronRight } from "lucide-react";
import type { Telemetry } from "@/lib/schemas";

const PAGE_SIZE = 150;

interface Props {
  history: Telemetry[];
  page: number;
  onPage: (p: number) => void;
}

function formatTs(iso: string): string {
  return new Date(iso).toISOString().replace("T", " ").slice(0, 19) + "Z";
}

export function TelemetryLog({ history, page, onPage }: Props) {
  const hasMore = history.length === PAGE_SIZE;
  const hasPrev = page > 0;

  if (history.length === 0 && page === 0) {
    return (
      <div className="flex flex-1 items-center justify-center text-xs text-zinc-600 font-mono">
        No telemetry data
      </div>
    );
  }

  return (
    <div className="flex flex-col flex-1 min-h-0">
      <ScrollArea className="flex-1 min-h-0">
        <div className="p-4 space-y-0">
          {history.map((entry, i) => (
            <div
              key={`${entry.Timestamp}-${i}`}
              className="flex gap-3 py-1.5 border-b border-zinc-900 last:border-0"
            >
              <span className="font-mono text-xs text-zinc-500 shrink-0 pt-px">
                {formatTs(entry.Timestamp)}
              </span>
              <span className="font-mono text-xs text-zinc-400 leading-relaxed">
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

      <div className="flex items-center justify-between px-4 py-2 border-t border-zinc-800 bg-zinc-950 shrink-0">
        <button
          onClick={() => onPage(page - 1)}
          disabled={!hasPrev}
          className="flex items-center gap-1 text-xs text-zinc-400 hover:text-zinc-200 disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
        >
          <ChevronLeft size={14} />
          Prev
        </button>
        <span className="text-xs font-mono text-zinc-600">
          Page {page + 1}{hasMore ? "" : " — last"}
        </span>
        <button
          onClick={() => onPage(page + 1)}
          disabled={!hasMore}
          className="flex items-center gap-1 text-xs text-zinc-400 hover:text-zinc-200 disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
        >
          Next
          <ChevronRight size={14} />
        </button>
      </div>
    </div>
  );
}
