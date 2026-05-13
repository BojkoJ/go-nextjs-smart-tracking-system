package handler

import (
	"context"
	"log/slog"
	"time"

	"github.com/BojkoJ/go-nextjs-smart-tracking-system/backend/internal/core/domain"
	"github.com/BojkoJ/go-nextjs-smart-tracking-system/backend/internal/core/ports"
	pb "github.com/BojkoJ/go-nextjs-smart-tracking-system/backend/proto"
)

// ---------------------------------------------------------------------------------------------------------
//                                         ADAPTERS - GRPC HANDLER
// ---------------------------------------------------------------------------------------------------------
// Proč tento soubor:
// Handler je vstupní brána systému - přijímá data z vnějšku (Simulator) a předává dovnitř (service).
// Je to adaptér, který překládá mezi gRPC světem  (Protobuf structs) a doménovým světem (domain structs).
// Nic za touto hranicí nesmí vidět *pb.TelemetryRequest.
// ---------------------------------------------------------------------------------------------------------

type GRPCHandler struct {
	pb.UnimplementedTelemetryServiceServer // gRPC forward-compatibility pattern
	service                                ports.IngestionService
	logger                                 *slog.Logger
}

var _ pb.TelemetryServiceServer = (*GRPCHandler)(nil)

// jiný compile-timechceck: implementujeme VYGENEROVANÝ interface z proto:
// backend/proto/telemetry_grpc.pb.go - interface TelemetryServiceServer.
// GRPCHandler musí implementovat:
//	- SendTelemetry(context.Context, *TelemetryRequest) (*TelemetryResponse, error)

// NewGRPCHandler je konstruktor, který příjme službu a logger zvenku a vytvoří instanci grpc handleru
func NewGRPCHandler(service ports.IngestionService, logger *slog.Logger) *GRPCHandler {
	return &GRPCHandler{
		service: service,
		logger:  logger,
	}
}

func (handler *GRPCHandler) SendTelemetry(ctx context.Context, request *pb.TelemetryRequest) (*pb.TelemetryResponse, error) {
	// 1) mapování *pb.TelemetryRequest -> domain.TelemetryData
	// Proč mapujeme místo přímého předání req:
	// Adapter boundary pravidlo - za touto hranicí nesmí existovat žádná závislost na pb balíčku.
	// Service a domain o Protobuf nevědí.
	// Kdybychom předali "request" dál, celý vnitřní systém by závisel na vygenerovaném kódu.
	telemetry := domain.TelemetryData{
		AssetID:     request.AssetId,
		Latitude:    request.Latitude,
		Longitude:   request.Longitude,
		Temperature: request.Temperature,
		Humidity:    request.Humidity,
		IsLocked:    request.IsLocked,
		Timestamp:   time.Unix(0, request.TimestampNs),
		TraceID:     request.TraceId,
	}

	// 2) zavolání service
	err := handler.service.IngestTelemetry(ctx, telemetry)
	if err != nil {
		handler.logger.Error("error ingesting telemetry", "error", err)
		// nevracíme jako chybu zabalme do response
		// validation error není gRPC protocol error: simulator má vědět, že data jsou špatná, ne že komunikace selhala
		return &pb.TelemetryResponse{Success: false, Message: err.Error()}, nil
	}

	return &pb.TelemetryResponse{Success: true, Message: "OK"}, nil
}
