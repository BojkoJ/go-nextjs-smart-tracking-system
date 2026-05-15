"use client";

import { useEffect, useRef } from "react";
import type { Telemetry } from "@/lib/schemas";

interface ShipPosition {
  assetId: string;
  lat: number;
  lon: number;
}

interface Props {
  positions: ShipPosition[];
  onShipClick: (assetId: string) => void;
  selectedId: string | null;
}

declare global {
  interface Window {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    L: any;
  }
}

export function ShipMap({ positions, onShipClick, selectedId }: Props) {
  const mapRef = useRef<HTMLDivElement>(null);
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const mapInstanceRef = useRef<any>(null);
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const markersRef = useRef<Map<string, any>>(new Map());

  useEffect(() => {
    if (typeof window === "undefined" || mapInstanceRef.current) return;

    const linkEl = document.createElement("link");
    linkEl.rel = "stylesheet";
    linkEl.href =
      "https://unpkg.com/leaflet@1.9.4/dist/leaflet.css";
    document.head.appendChild(linkEl);

    const script = document.createElement("script");
    script.src = "https://unpkg.com/leaflet@1.9.4/dist/leaflet.js";
    script.onload = () => {
      const L = window.L;
      if (!mapRef.current || mapInstanceRef.current) return;

      const map = L.map(mapRef.current, {
        center: [20, 0],
        zoom: 3,
        zoomControl: true,
      });

      L.tileLayer(
        "https://{s}.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}{r}.png",
        {
          attribution: "&copy; OpenStreetMap &copy; CARTO",
          maxZoom: 19,
        }
      ).addTo(map);

      mapInstanceRef.current = map;
    };
    document.head.appendChild(script);
  }, []);

  useEffect(() => {
    const L = window.L;
    if (!L || !mapInstanceRef.current || positions.length === 0) return;

    const map = mapInstanceRef.current;

    markersRef.current.forEach((m) => m.remove());
    markersRef.current.clear();

    const shipIcon = (isSelected: boolean) => {
      const bg = isSelected ? "#38bdf8" : "#64748b";
      const borderColor = isSelected ? "#0ea5e9" : "#475569";
      const shadow = isSelected ? "0 0 8px #38bdf8" : "none";
      return L.divIcon({
        className: "",
        html: `<div style="width:12px;height:12px;border-radius:2px;background:${bg};border:2px solid ${borderColor};box-shadow:${shadow}"></div>`,
        iconSize: [12, 12],
        iconAnchor: [6, 6],
      });
    };

    // Wszystkie pody mají stejnou pozici (jsou na jedné lodi) — zobraz jeden marker
    // Ale uživatel klikne a vybere jeden z 50 kontejnerů
    const uniquePositions = new Map<string, ShipPosition>();
    positions.forEach((p) => {
      const key = `${p.lat.toFixed(2)}_${p.lon.toFixed(2)}`;
      if (!uniquePositions.has(key)) {
        uniquePositions.set(key, p);
      }
    });

    uniquePositions.forEach((pos, key) => {
      const isSelected =
        selectedId !== null &&
        positions.find((p) => p.assetId === selectedId)?.lat === pos.lat &&
        positions.find((p) => p.assetId === selectedId)?.lon === pos.lon;

      const marker = L.marker([pos.lat, pos.lon], {
        icon: shipIcon(isSelected ?? false),
      })
        .addTo(map)
        .on("click", () => onShipClick(pos.assetId));

      markersRef.current.set(key, marker);
    });
  }, [positions, selectedId, onShipClick]);

  return (
    <div ref={mapRef} className="flex-1 min-h-0" />
  );
}
