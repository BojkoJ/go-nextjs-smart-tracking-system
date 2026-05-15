"use client";

import { useEffect, useRef, useState } from "react";

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

const SHIP_PATHS =
  `<path d="M12 10.189V14"/>` +
  `<path d="M12 2v3"/>` +
  `<path d="M19 13V7a2 2 0 0 0-2-2H7a2 2 0 0 0-2 2v6"/>` +
  `<path d="M19.38 20A11.6 11.6 0 0 0 21 14l-8.188-3.639a2 2 0 0 0-1.624 0L3 14a11.6 11.6 0 0 0 2.81 7.76"/>` +
  `<path d="M2 21c.6.5 1.2 1 2.5 1 2.5 0 2.5-2 5-2 1.3 0 1.9.5 2.5 1s1.2 1 2.5 1c2.5 0 2.5-2 5-2 1.3 0 1.9.5 2.5 1"/>`;

const SHIP_SVG_DEFAULT =
  `<svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="#64748b" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">` +
  SHIP_PATHS + `</svg>`;

const SHIP_SVG_SELECTED =
  `<svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="#0ea5e9" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" style="filter:drop-shadow(0 0 5px #38bdf8)">` +
  SHIP_PATHS + `</svg>`;

export function ShipMap({ positions, onShipClick, selectedId }: Props) {
  const mapRef = useRef<HTMLDivElement>(null);
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const mapInstanceRef = useRef<any>(null);
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const markersRef = useRef<Map<string, any>>(new Map());
  const [leafletReady, setLeafletReady] = useState(false);

  useEffect(() => {
    if (typeof window === "undefined" || mapInstanceRef.current) return;

    const linkEl = document.createElement("link");
    linkEl.rel = "stylesheet";
    linkEl.href = "https://unpkg.com/leaflet@1.9.4/dist/leaflet.css";
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
        "https://{s}.basemaps.cartocdn.com/rastertiles/voyager/{z}/{x}/{y}{r}.png",
        {
          attribution:
            '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors &copy; <a href="https://carto.com/attributions">CARTO</a>',
          maxZoom: 19,
        }
      ).addTo(map);

      mapInstanceRef.current = map;
      setLeafletReady(true);
    };
    document.head.appendChild(script);
  }, []);

  useEffect(() => {
    const L = window.L;
    if (!leafletReady || !L || !mapInstanceRef.current || positions.length === 0) return;

    const map = mapInstanceRef.current;

    markersRef.current.forEach((m) => m.remove());
    markersRef.current.clear();

    const shipIcon = (isSelected: boolean) =>
      L.divIcon({
        className: "",
        html: isSelected ? SHIP_SVG_SELECTED : SHIP_SVG_DEFAULT,
        iconSize: [32, 32],
        iconAnchor: [16, 16],
      });

    const uniquePositions = new Map<string, ShipPosition>();
    positions.forEach((p) => {
      const key = `${p.lat.toFixed(2)}_${p.lon.toFixed(2)}`;
      if (!uniquePositions.has(key)) {
        uniquePositions.set(key, p);
      }
    });

    uniquePositions.forEach((pos, key) => {
      const selectedPos = selectedId
        ? positions.find((p) => p.assetId === selectedId)
        : null;
      const isSelected =
        selectedPos !== null &&
        selectedPos !== undefined &&
        selectedPos.lat === pos.lat &&
        selectedPos.lon === pos.lon;

      const marker = L.marker([pos.lat, pos.lon], {
        icon: shipIcon(isSelected),
      })
        .addTo(map)
        .on("click", () => onShipClick(pos.assetId));

      markersRef.current.set(key, marker);
    });

    if (uniquePositions.size > 0) {
      const first = uniquePositions.values().next().value;
      if (first) map.setView([first.lat, first.lon], 5);
    }
  }, [positions, selectedId, onShipClick, leafletReady]);

  return <div ref={mapRef} className="flex-1 min-h-0" />;
}
