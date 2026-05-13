package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/BojkoJ/go-nextjs-smart-tracking-system/backend/internal/core/domain"
	"github.com/BojkoJ/go-nextjs-smart-tracking-system/backend/internal/core/ports"
	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------------------------------------
//                                      SERVICES - PROCESSOR SERVICE
// ---------------------------------------------------------------------------------------------------------
// Proč tato vrstva existuje:
// Handler (gRPC) přijme data a potřebuje je zpracovat.
// Mohli bychom logiku psát přímo v handleru — ale pak by handler věděl o NATS, o Postgres, o business pravidlech.
// Místo toho handler zavolá service, která zná jen business logiku a komunikuje přes interfaces.
// Toto je Use Case vrstva z Clean Architecture.
// ---------------------------------------------------------------------------------------------------------
// processing.go:
// Processor Service je mozek systému - zde se implementují business pravidla.
// Přečte zprávu z NATS, deserializuje, zkontroluje pravidla, uloží do DB, případně vytvoří Alert.
// ---------------------------------------------------------------------------------------------------------

type ProcessingServiceImpl struct {
	assetRepo     ports.AssetRepository     // interface, který drží operace nad Asset Entitou
	telemetryRepo ports.TelemetryRepository // interface, který drží operace nad TelemetryData Entitou
	alertRepo     ports.AlertRepository     // interface, který drží operace nad Alert Entitou
	logger        *slog.Logger
}

var _ ports.ProcessingService = (*ProcessingServiceImpl)(nil)

// NewProcessingService je konstruktor, který příjme repozitáře zvenku a vytvoří instanci
func NewProcessingService(assetRepo ports.AssetRepository, telemetryRepo ports.TelemetryRepository, alertRepo ports.AlertRepository, logger *slog.Logger) *ProcessingServiceImpl {
	return &ProcessingServiceImpl{
		assetRepo:     assetRepo,
		telemetryRepo: telemetryRepo,
		alertRepo:     alertRepo,
		logger:        logger,
	}
}

func (processService *ProcessingServiceImpl) ProcessTelemetry(ctx context.Context, telemetry domain.TelemetryData) error {
	// 1) načtení assetu
	asset, err := processService.assetRepo.GetAssetByID(ctx, telemetry.AssetID)
	if err != nil && asset == nil {
		return fmt.Errorf("getting an asset by ID: %w", err)
	}
	if asset == nil {
		processService.logger.Warn("no asset found", "assetID", telemetry.AssetID)
		return nil
	}

	// 2) uložení telemetrie
	err = processService.telemetryRepo.SaveTelemetry(ctx, telemetry)
	if err != nil {
		return fmt.Errorf("saving new telemetry to PostgreSQL database: %w", err)
	}

	// 3) kontrola teplotních limitů a stavu zámku: business logika
	if telemetry.Temperature > asset.MaxTemperature {
		alert := domain.Alert{
			ID:        uuid.New().String(),
			AssetID:   asset.ID,
			Type:      domain.AlertTemperatureMax,
			Message:   fmt.Sprintf("ALERT: Asset exceeded it's maximum temperature(%f). It's current temp is: %f", asset.MaxTemperature, telemetry.Temperature),
			CreatedAt: time.Now(),
		}
		err := processService.alertRepo.SaveAlert(ctx, alert)
		if err != nil {
			return fmt.Errorf("saving new alert to PostgreSQL database: %w", err)
		}
	}
	if telemetry.Temperature < asset.MinTemperature {
		alert := domain.Alert{
			ID:        uuid.New().String(),
			AssetID:   asset.ID,
			Type:      domain.AlertTemperatureMin,
			Message:   fmt.Sprintf("ALERT: Asset exceeded it's minimum temperature(%f). It's current temp is: %f", asset.MinTemperature, telemetry.Temperature),
			CreatedAt: time.Now(),
		}
		err := processService.alertRepo.SaveAlert(ctx, alert)
		if err != nil {
			return fmt.Errorf("saving new alert to PostgreSQL database: %w", err)
		}
	}
	if !telemetry.IsLocked {
		alert := domain.Alert{
			ID:        uuid.New().String(),
			AssetID:   asset.ID,
			Type:      domain.AlertUnlocked,
			Message:   "ALERT: Asset's lock is unlocked!",
			CreatedAt: time.Now(),
		}
		err := processService.alertRepo.SaveAlert(ctx, alert)
		if err != nil {
			return fmt.Errorf("saving new alert to PostgreSQL database: %w", err)
		}
	}
	if telemetry.Humidity > asset.MaxHumidity {
		alert := domain.Alert{
			ID:        uuid.New().String(),
			AssetID:   asset.ID,
			Type:      domain.AlertHumidityMax,
			Message:   fmt.Sprintf("ALERT: Asset exceeded its maximum humidity(%.1f%%). Current humidity: %.1f%%", asset.MaxHumidity, telemetry.Humidity),
			CreatedAt: time.Now(),
		}
		err := processService.alertRepo.SaveAlert(ctx, alert)
		if err != nil {
			return fmt.Errorf("saving new alert to PostgreSQL database: %w", err)
		}
	}
	// zatím přeskočíme vzdálenostní výpočet.
	// TODO: Dokončit vzdálenostní výpočet - lifecycle management.
	return nil
}
