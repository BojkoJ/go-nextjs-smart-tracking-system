-- ---------------------------------------------------------------------------------------------------------
--                                     DATABASE SCHEMA
-- ---------------------------------------------------------------------------------------------------------
-- Tento soubor definuje strukturu databáze.
-- Spuštění: kubectl exec -n infrastructure <postgres-pod> -- psql -U tracking_user -d tracking_db -f schema.sql
-- ---------------------------------------------------------------------------------------------------------

-- TABULKA: assets
-- Reprezentuje firemní aktiva (přepravní lodní kontejnery).
-- Obsahuje metadata a business pravidla konkrétního aktiva (teplotní limity, status životního cyklu).
CREATE TABLE IF NOT EXISTS assets (
    id              TEXT PRIMARY KEY,          -- UUID generované aplikací, ne databází
    name            TEXT NOT NULL,
    max_temperature DOUBLE PRECISION NOT NULL, -- business pravidlo: horní teplotní limit (°C)
    min_temperature DOUBLE PRECISION NOT NULL, -- business pravidlo: dolní teplotní limit (°C)
    status          TEXT NOT NULL DEFAULT 'new', -- životní cyklus: new → active → maintenance → decommissioned
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- TABULKA: telemetry
-- Append-only log senzorových dat ze všech aktiv.
-- Záznamy se nikdy neupravují, jen přibývají - proto BIGSERIAL místo UUID pro výkon.
CREATE TABLE IF NOT EXISTS telemetry (
    id           BIGSERIAL PRIMARY KEY,        -- auto-increment int64, rychlejší insert než UUID pro append-only log
    asset_id     TEXT NOT NULL REFERENCES assets(id),
    latitude     DOUBLE PRECISION NOT NULL,
    longitude    DOUBLE PRECISION NOT NULL,
    temperature  DOUBLE PRECISION NOT NULL,
    humidity     DOUBLE PRECISION NOT NULL,
    is_locked    BOOLEAN NOT NULL,
    timestamp_ns BIGINT NOT NULL,              -- Unix timestamp v nanosekundách (jednodušší než google.protobuf.Timestamp)
    trace_id     TEXT NOT NULL,                -- OpenTelemetry Trace ID - sleduje cestu zprávy celým systémem
    recorded_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- TABULKA: alerts
-- Výjimečné události vyhodnocené Processor Service (překročení teploty, odemčený kontejner atd.)
-- Oddělená tabulka od telemetrie - jiný životní cyklus, jiní konzumenti (on-call tým vs. dashboard).
CREATE TABLE IF NOT EXISTS alerts (
    id         TEXT PRIMARY KEY,               -- UUID generované aplikací
    asset_id   TEXT NOT NULL REFERENCES assets(id),
    type       TEXT NOT NULL,                  -- viz domain.AlertType: temperature_exceeded_max_limit atd.
    message    TEXT NOT NULL,                  -- lidsky čitelná zpráva sestavená Processor Service
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- INDEXY
-- Optimalizace pro nejčastější dotazy Query Service.
-- Bez indexů by Postgres skenoval celou tabulku (O(n)), s indexy hledá v O(log n).

-- LoadLastTelemetry a LoadHistory filtrují vždy dle asset_id a řadí dle timestamp_ns
CREATE INDEX IF NOT EXISTS idx_telemetry_asset_id ON telemetry(asset_id, timestamp_ns DESC);

-- LoadAlertsForAsset filtruje dle asset_id
CREATE INDEX IF NOT EXISTS idx_alerts_asset_id ON alerts(asset_id);
