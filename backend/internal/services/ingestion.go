package services

import (
	"context"
	"encoding/json"
	"fmt"
	"math"

	"github.com/BojkoJ/go-nextjs-smart-tracking-system/backend/internal/core/domain"
	"github.com/BojkoJ/go-nextjs-smart-tracking-system/backend/internal/core/ports"
)

// ---------------------------------------------------------------------------------------------------------
//                                      SERVICES - INGESTION SERVICE
// ---------------------------------------------------------------------------------------------------------
// Proč tato vrstva existuje:
// Handler (gRPC) přijme data a potřebuje je zpracovat.
// Mohli bychom logiku psát přímo v handleru — ale pak by handler věděl o NATS, o Postgres, o business pravidlech.
// Místo toho handler zavolá service, která zná jen business logiku a komunikuje přes interfaces.
// Toto je Use Case vrstva z Clean Architecture.
// ---------------------------------------------------------------------------------------------------------
// ingestion.go:
// Ingest Service má jednu zodpovědnost: příjmout data ze Simulatoru, validovat je a hodit do NATS.
// Nezná Postgres, nezná HTTP. Jen přijme a předá dál — "fire and  forget".
// ---------------------------------------------------------------------------------------------------------

// IngestionServiceImpl má jen jednu závislost: producer, což je interface, ne konkrétní NATS struct.
// Dependency Injection pattern
type IngestionServiceImpl struct {
	producer ports.EventProducer
}

var _ ports.IngestionService = (*IngestionServiceImpl)(nil)

// NewIngestionService je konstruktor, který příjme producer zvenku a vytvoří instanci této mikroslužby.
func NewIngestionService(producer ports.EventProducer) *IngestionServiceImpl {
	return &IngestionServiceImpl{producer: producer}
}

func (ingestService *IngestionServiceImpl) IngestTelemetry(ctx context.Context, telemetry domain.TelemetryData) error {
	// 1) validace: zvalidujeme základní business pravidla
	if telemetry.AssetID == "" {
		return fmt.Errorf("ingesting telemetry, got no AssetID")
	}
	if math.IsNaN(telemetry.Temperature) {
		return fmt.Errorf("ingesting telemetry, temperature is not a number")
	}
	if telemetry.Latitude > 90 || telemetry.Latitude < -90 {
		return fmt.Errorf("ingesting telemetry, latitude is a nonsensical value")
	}
	if telemetry.Longitude > 180 || telemetry.Longitude < -180 {
		return fmt.Errorf("ingesting telemetry, longitude is a nonsensical value")
	}

	// 2) serializace na []byte:
	// gRPC handler (adaptér): příjme gRPC data, zmapuje a zavolá IngestTelemetry.
	// V momentě, kdy IngestTelemetry dostane data, Protobuf je už deserializoval - máme čistý Go struct.
	// Ingestion Service pak dotane domain.TelemetryData a potřebuje je poslat do NATS jako []byte.
	// proto serializujeme znovu - ale tentokrát ne Protobuf, ale JSON.
	// NATS není gRPC, nezná Protobuf, posílá jen bajty.
	jsonEncodedTelemetry, err := json.Marshal(telemetry)
	if err != nil {
		return fmt.Errorf("serializing telemetry: %w", err)
	}

	err = ingestService.producer.PublishMessage(ctx, "telemetry.ingest", jsonEncodedTelemetry)
	if err != nil {
		return fmt.Errorf("publishing a message to NATS JetStream: %w", err)
	}

	return nil
}
