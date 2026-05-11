package ports

import (
	"context"

	"github.com/BojkoJ/go-nextjs-smart-tracking-system/backend/internal/core/domain"
)

// ---------------------------------------------------------------------------------------------------------
//                                                SERVICES
// ---------------------------------------------------------------------------------------------------------
// Processor Service bude volat IngestionService interface - ne konkrétní struct.
// V produkci dostane reálnou implementaci, v testech dostane fake (mock).
// Tento pattern se jmenuje Dependency Injection a je v Go velmi běžný: závislosti se předávají zvenku, ne vytvářejí uvnitř.
// ---------------------------------------------------------------------------------------------------------

// IngestionService přijímá data ze simulátoru
type IngestionService interface {
	IngestTelemetry(ctx context.Context, telemetry domain.TelemetryData) error
}

// ProcessingService zpracovává data a business logiku
type ProcessingService interface {
	ProcessTelemetry(ctx context.Context, telemetry domain.TelemetryData) error
}
