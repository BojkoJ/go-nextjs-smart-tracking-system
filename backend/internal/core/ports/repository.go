package ports

import (
	"context"

	"github.com/BojkoJ/go-nextjs-smart-tracking-system/backend/internal/core/domain"
)

// ---------------------------------------------------------------------------------------------------------
//                                             PORTS
// ---------------------------------------------------------------------------------------------------------
// Proč Ports: Domain Layer potřebuje komunikovat s vnějším světem: ukládat data, posílat zprávy,
// ale nesmí vědět jak. Řešením je, že se napíše "seznam požadavků" ve formě Go interfaců.
// To říká: "potřebuji někoho, kdo umí uložit Asset - nezajímá mě ale, jestli je to PostgreSQL, SQLite nebo in-memory
// Implementace (PostgreSQL, NATS atd.) bude v /backend/internal/adapters/. Ports jsou jen kontrakt.
// ---------------------------------------------------------------------------------------------------------

// Go intercaes: Pokud má struct všechny metody co interface vyžaduje, automaticky ho implementuje.
// Tato vlastnost umožňuje testovat snadno: v testech stačí napsat fake struct s potřebnými metodami

// Proč context.Context jako první parametr všude:
// Je to Go konvence, context nese dvě věci: deadline, cancellation (nadřazená operace byla zrušena, skonči i ty)
// Bez context by databázový dotaz mohl běžet donekonečna i když uživatel dávno odešel.

// V ports souborech poprvé importujeme z jiného balíčku uvnitř projektu.
// - budeme zde potřebovat domain layer (takže domain.Asset, domain.TelemetryData atd.)

// AssetRepository drží operace nad Asset Entitou
type AssetRepository interface {
	SaveAsset(ctx context.Context, asset domain.Asset) error                                // metoda pro uložení jednoho aktiva
	GetAssetByID(ctx context.Context, assetID string) (*domain.Asset, error)                // metoda pro načtení jednoho aktiva podle ID
	ListAssets(ctx context.Context) ([]domain.Asset, error)                                 // metoda pro načtení všech aktiv
	UpdateAssetStatus(ctx context.Context, assetID string, status domain.AssetStatus) error // aktualizace lifecycle statusu aktiva
}

// TelemetryRepository drží operace nad TelemetryData Entitou
type TelemetryRepository interface {
	SaveTelemetry(ctx context.Context, telemetry domain.TelemetryData) error                  // metoda pro uložení jednoho záznamu telemetrie
	GetLastTelemetry(ctx context.Context, assetID string) (*domain.TelemetryData, error)      //metoda pro načtení posledního záznamu telemetrie pro dané aktivum
	ListTelemetryHistory(ctx context.Context, assetID string) ([]domain.TelemetryData, error) // metoda pro načtení historie
}

type AlertRepository interface {
	SaveAlert(ctx context.Context, alert domain.Alert) error
	ListAlerts(ctx context.Context) ([]domain.Alert, error)
	ListAllAlertsForAsset(ctx context.Context, assetID string) ([]domain.Alert, error)
}
