package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/BojkoJ/go-nextjs-smart-tracking-system/backend/internal/core/domain"
	"github.com/BojkoJ/go-nextjs-smart-tracking-system/backend/internal/core/ports"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ---------------------------------------------------------------------------------------------------------
//                                         ADAPTERS - REPOSITORY/POSTGRES_REPO
// ---------------------------------------------------------------------------------------------------------
// Proč pgx místo stdlib database/dql? pgx je nativní Postgres driver - nepotřebuje bridge vrstu,
// přímo komunikuje s Postgres protokolem.
// Přibližně 2-3x rychlejší než database/sql bridge,
// podporuje Postgres-specific typy nativn, a pgxpool (connection pool) má lepší default chování.
// ---------------------------------------------------------------------------------------------------------

type PostgresRepo struct {
	PostgresPool *pgxpool.Pool
}

var _ ports.AssetRepository = (*PostgresRepo)(nil)
var _ ports.TelemetryRepository = (*PostgresRepo)(nil)
var _ ports.AlertRepository = (*PostgresRepo)(nil)

func NewPostgresRepo(ctx context.Context, connStr string) (*PostgresRepo, error) {
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		return nil, fmt.Errorf("creating postgres pool: %w", err)
	}
	// Proč pool.Ping():
	// pgxpool.New samotné spojení nenaváže - je lazy.
	// Ping ověří, že databáze skutečně odpovídá. Bez Ping bychom zjistili problém až při prvním dotazu - pozdě
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("pinging postgres: %w", err)
	}
	return &PostgresRepo{PostgresPool: pool}, nil
}

// ------ Save funkce: ---------------------------------------------------------------------------------------------------------------------

// SaveAsset uloží do PostgreSQL databáze jeden Asset (aktivum)
func (pgRepo *PostgresRepo) SaveAsset(ctx context.Context, asset domain.Asset) error {
	_, err := pgRepo.PostgresPool.Exec(ctx, "INSERT INTO assets (id, name, max_temperature, min_temperature, status, created_at) values ($1, $2, $3, $4, $5, $6)",
		asset.ID, asset.Name, asset.MaxTemperature, asset.MinTemperature, asset.Status, asset.CreatedAt)
	if err != nil {
		return fmt.Errorf("saving asset to PostgreSQL: %w", err)
	}
	return nil
}

// SaveTelemetry uloží do PostgreSQL databáze jeden záznam Telemetrie
func (pgRepo *PostgresRepo) SaveTelemetry(ctx context.Context, telemetry domain.TelemetryData) error {
	_, err := pgRepo.PostgresPool.Exec(ctx, "INSERT INTO telemetry (asset_id, latitude, longitude, temperature, humidity, is_locked, timestamp_ns, trace_id) values ($1, $2, $3, $4, $5, $6, $7, $8)",
		telemetry.AssetID, telemetry.Latitude, telemetry.Longitude, telemetry.Temperature, telemetry.Humidity, telemetry.IsLocked, telemetry.Timestamp.UnixNano(), telemetry.TraceID)
	if err != nil {
		return fmt.Errorf("saving telemetry to PostgreSQL: %w", err)
	}
	return nil
}

// SaveAlert uloží do PostgreSQL databáze jeden Alert
func (pgRepo *PostgresRepo) SaveAlert(ctx context.Context, alert domain.Alert) error {
	_, err := pgRepo.PostgresPool.Exec(ctx, "INSERT INTO alerts (id, asset_id, type, message, created_at) values ($1, $2, $3, $4, $5)",
		alert.ID, alert.AssetID, alert.Type, alert.Message, alert.CreatedAt)
	if err != nil {
		return fmt.Errorf("saving alert to PostgreSQL: %w", err)
	}
	return nil
}

// ------ List funkce: ---------------------------------------------------------------------------------------------------------------------

// ListAssets získá z PostgreSQL databáze všechny Assety (aktiva) a vrátí je jako slice
func (pgRepo *PostgresRepo) ListAssets(ctx context.Context) ([]domain.Asset, error) {
	var assets []domain.Asset

	rows, err := pgRepo.PostgresPool.Query(ctx, "SELECT id, name, max_temperature, min_temperature, status, created_at FROM assets")
	if err != nil {
		return nil, fmt.Errorf("querying all assets: %w", err)
	}

	defer rows.Close()

	// neboli while rows.Next() == true
	for rows.Next() {
		var a domain.Asset
		err := rows.Scan(&a.ID, &a.Name, &a.MaxTemperature, &a.MinTemperature, &a.Status, &a.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("scanning row into help variable: %w", err)
		}
		assets = append(assets, a)
	}

	return assets, nil
}

// ListAlerts získá z PostgreSQL databáze všechny Alerty a vrátí je jako slice
func (pgRepo *PostgresRepo) ListAlerts(ctx context.Context) ([]domain.Alert, error) {
	var alerts []domain.Alert

	rows, err := pgRepo.PostgresPool.Query(ctx, "SELECT id, asset_id, type, message, created_at FROM alerts")
	if err != nil {
		return nil, fmt.Errorf("querying all alerts: %w", err)
	}

	defer rows.Close()

	// neboli while rows.Next() == true
	for rows.Next() {
		var a domain.Alert
		err := rows.Scan(&a.ID, &a.AssetID, &a.Type, &a.Message, &a.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("scanning row into help variable: %w", err)
		}
		alerts = append(alerts, a)
	}

	return alerts, nil
}

// ListTelemetryHistory získá z PostgreSQL všechny záznamy Telemetrie pro daný Asset (aktivum) a vrátí je jako slice
func (pgRepo *PostgresRepo) ListTelemetryHistory(ctx context.Context, assetID string) ([]domain.TelemetryData, error) {
	var telemetryEntries []domain.TelemetryData

	rows, err := pgRepo.PostgresPool.Query(ctx, "SELECT asset_id, latitude, longitude, temperature, humidity, is_locked, timestamp_ns, trace_id from telemetry WHERE asset_id = $1 ORDER BY timestamp_ns DESC", assetID)
	if err != nil {
		return nil, fmt.Errorf("querying all telemetries for one asset: %w", err)
	}

	defer rows.Close()

	// neboli while rows.Next() == true
	for rows.Next() {
		var td domain.TelemetryData
		var timeStampNs int64

		err := rows.Scan(&td.AssetID, &td.Latitude, &td.Longitude, &td.Temperature, &td.Humidity, &td.IsLocked, &timeStampNs, &td.TraceID)
		if err != nil {
			return nil, fmt.Errorf("scanning row into help variable: %w", err)
		}
		td.Timestamp = time.Unix(0, timeStampNs)
		telemetryEntries = append(telemetryEntries, td)
	}

	return telemetryEntries, nil
}

// ListAllAlertsForAsset získá z PostgreSQL všechny Alerty pro daný Asset (aktivum) a vrátí je jako slice
func (pgRepo *PostgresRepo) ListAllAlertsForAsset(ctx context.Context, assetID string) ([]domain.Alert, error) {
	var alerts []domain.Alert

	rows, err := pgRepo.PostgresPool.Query(ctx, "SELECT id, asset_id, type, message, created_at FROM alerts WHERE asset_id = $1", assetID)
	if err != nil {
		return nil, fmt.Errorf("querying all alerts for one asset: %w", err)
	}

	defer rows.Close()

	// neboli while rows.Next() == true
	for rows.Next() {
		var a domain.Alert
		err := rows.Scan(&a.ID, &a.AssetID, &a.Type, &a.Message, &a.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("scanning row into help variable: %w", err)
		}
		alerts = append(alerts, a)
	}

	return alerts, nil
}

// ------ Get funkce: ----------------------------------------------------------------------------------------------------------------------

// GetAssetByID získá z PostgreSQL databáze jeden konkrétní Asset (aktivum) podle předaného assetID
func (pgRepo *PostgresRepo) GetAssetByID(ctx context.Context, assetID string) (*domain.Asset, error) {
	row := pgRepo.PostgresPool.QueryRow(ctx, "SELECT id, name, max_temperature, mit_temperature, status, created_at FROM asets WHERE id = $1", assetID)

	var a domain.Asset
	err := row.Scan(&a.ID, &a.Name, &a.MaxTemperature, &a.MinTemperature, &a.Status, &a.CreatedAt)
	if err != nil {
		// V SQL je "záznam nenalezen" validní výsledek - ne chyba aplikace.
		if errors.Is(err, pgx.ErrNoRows) { // pokud je Error "no rows" - nezískali jsme z PostgeSQL žádné řádky
			return nil, nil // Záznam neexistuje
		}
		return nil, fmt.Errorf("getting one asset by it's ID: %w", err)
	}

	return &a, nil
}

// GetLastTelemetry získá z PostgreSQL databáze poslední záznam telemetrie, který do tabulky telemetry byl uložen
func (pgRepo *PostgresRepo) GetLastTelemetry(ctx context.Context, assetID string) (*domain.TelemetryData, error) {
	row := pgRepo.PostgresPool.QueryRow(ctx, "SELECT asset_id, latitude, longitude, temperature, humidity, is_locked, timestamp_ns, trace_id from telemetry WHERE asset_id = $1 ORDER BY timestamp_ns DESC LIMIT 1", assetID)

	var td domain.TelemetryData
	var timeStampNs int64

	err := row.Scan(&td.AssetID, &td.Latitude, &td.Longitude, &td.Temperature, &td.Humidity, &td.IsLocked, &timeStampNs, &td.TraceID)
	td.Timestamp = time.Unix(0, timeStampNs)
	if err != nil {
		// V SQL je "záznam nenalezen" validní výsledek - ne chyba aplikace.
		if errors.Is(err, pgx.ErrNoRows) { // pokud je Error "no rows" - nezískali jsme z PostgeSQL žádné řádky
			return nil, nil // Záznam neexistuje
		}
		return nil, fmt.Errorf("getting one last telemetry for asset:%s: %w", assetID, err)
	}

	return &td, nil
}
