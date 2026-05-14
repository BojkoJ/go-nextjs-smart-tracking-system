-- ---------------------------------------------------------------------------------------------------------
--                                     DATABASE SEED
-- ---------------------------------------------------------------------------------------------------------
-- Tento soubor naplní databázi počátečními daty pro simulační scénář IQM Quantum Computers.
--
-- SCÉNÁŘ: IQM Quantum Computers, Espoo, Finsko
-- Přeprava 50 lodních kontejnerů s kvantovými komponenty (Niobium targets + PCB substráty)
-- Trasa: Busan (Jižní Korea) → Rotterdam (Nizozemí), ~20 000 km, ~28 dní
--
-- BUSINESS PRAVIDLA (limity uložené na každém assetu):
--   max_temperature: 28.0 °C  — nad touto hodnotou hrozí oxidace niobiového povrchu
--   min_temperature: 12.0 °C  — pod touto hodnotou kondenzuje vlhkost na PCB substrátech
--   max_humidity:    45.0 % RH — supravodivé vrstvy degradují při vyšší relativní vlhkosti
--
-- Spuštění (kubectl):
--   kubectl exec -i -n infrastructure postgresql-0 -- psql -U tracking_user -d tracking_db < backend/db/seed.sql
--
-- Spuštění (lokálně s port-forward):
--   kubectl port-forward -n infrastructure svc/postgresql 5432:5432
--   psql -U tracking_user -d tracking_db -h localhost < backend/db/seed.sql
--
-- Idempotentní: ON CONFLICT DO NOTHING umožňuje opakované spuštění bez chyby.
-- ---------------------------------------------------------------------------------------------------------

INSERT INTO assets (id, name, max_temperature, min_temperature, max_humidity, status)
SELECT
    'asset-' || i,
    'IQM Container  #' || i,
    28.0,
    12.0,
    45.0,
    'active'
FROM generate_series(1, 50) AS i
ON CONFLICT (id) DO NOTHING;