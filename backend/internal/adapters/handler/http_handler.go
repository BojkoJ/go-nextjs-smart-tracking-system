package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/BojkoJ/go-nextjs-smart-tracking-system/backend/internal/core/ports"
	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// ---------------------------------------------------------------------------------------------------------
//                                         ADAPTERS - HTTP HANDLER
// ---------------------------------------------------------------------------------------------------------
// Tento soubor:
// Překládá HTTP request -> volá repozitáře -> HTTP response.
// Za touto hranicí už nic neví o http.Request.
//
// Proč Query Service čte přímo z DB a ne přes Processing Service:
// Tohle je zjednodušený CQRS pattern (Command Query Responsibility Segregation):
// zápis (Commands) jde přes gRPC -> NATS -> Processor,
// čtení (Queries) jde přímo do DB.
// Výhoda: čtení neblokuje zápis, obě cesty se škálují nezávisle.
//
// Použijeme chi (http lightweight http router)
// ---------------------------------------------------------------------------------------------------------

type HTTPHandler struct {
	assetRepo     ports.AssetRepository
	telemetryRepo ports.TelemetryRepository
	alertRepo     ports.AlertRepository
	logger        *slog.Logger
}

// NewHTTPHandler je konstruktor, který příjme repozitáře a logger zvenku a vytvoří instanci http handleru
func NewHTTPHandler(assetRepo ports.AssetRepository, telemetryRepo ports.TelemetryRepository, alertRepo ports.AlertRepository, logger *slog.Logger) *HTTPHandler {
	return &HTTPHandler{
		assetRepo:     assetRepo,
		telemetryRepo: telemetryRepo,
		alertRepo:     alertRepo,
		logger:        logger,
	}
}

// Routes zaregistruje všechny endpointy a vrátí http.Handler
// v cmd/query/main.go pak jen předáme výsledek go http.ListenAndServer
func (handler *HTTPHandler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/assets", handler.listAssets)
	r.Get("/assets/{id}", handler.getAssetByID)
	r.Get("/assets/{id}/telemetry", handler.getLastTelemetry)
	r.Get("/assets/{id}/telemetry/history", handler.listTelemetryHistory)
	r.Get("/alerts", handler.listAlerts)
	r.Get("/assets/{id}/alerts", handler.listAllAlertsForAsset)
	r.Handle("/metrics", promhttp.Handler())

	return r
}

// /assets
func (handler *HTTPHandler) listAssets(writer http.ResponseWriter, request *http.Request) {
	// 1) volání repa
	assets, err := handler.assetRepo.ListAssets(request.Context())
	if err != nil {
		handler.logger.Error("failed to list assets", "error", err)
		http.Error(writer, "internal server error", http.StatusInternalServerError) // 500
		return
	}

	// 2) nastavení Content-Type header
	writer.Header().Set("Content-Type", "application/json")

	// 3) zakódování a zapsání
	// Encoder zapisuje přímo do ResponseWriter streamově - nealokuje mezipaměť pro celý JSON.
	// Pro velký slice assetů/telemetrie je to efektivnější, než json.Marshal()
	if err := json.NewEncoder(writer).Encode(assets); err != nil {
		handler.logger.Error("failed to encode response", "error", err)
	}
}

// /assets/{id}
func (handler *HTTPHandler) getAssetByID(writer http.ResponseWriter, request *http.Request) {
	id := chi.URLParam(request, "id")

	// 1) volání repa
	asset, err := handler.assetRepo.GetAssetByID(request.Context(), id)
	if err != nil {
		handler.logger.Error("failed to get asset by ID", "error", err)
		http.Error(writer, "internal server error", http.StatusInternalServerError) // 500
		return
	}
	if asset == nil {
		handler.logger.Warn("asset with this ID does not exist")
		http.NotFound(writer, request)
		return
	}

	// 2) nastavení Content-Type header
	writer.Header().Set("Content-Type", "application/json")

	// 3) zakódování a zapsání
	// Encoder zapisuje přímo do ResponseWriter streamově - nealokuje mezipaměť pro celý JSON.
	// Pro velký slice assetů/telemetrie je to efektivnější, než json.Marshal()
	if err := json.NewEncoder(writer).Encode(asset); err != nil {
		handler.logger.Error("failed to encode response", "error", err)
	}
}

// /assets/{id}/telemetry
func (handler *HTTPHandler) getLastTelemetry(writer http.ResponseWriter, request *http.Request) {
	id := chi.URLParam(request, "id")

	// 1) volání repa
	telemetry, err := handler.telemetryRepo.GetLastTelemetry(request.Context(), id)
	if err != nil {
		handler.logger.Error("failed to get last telemetry for asset", "error", err)
		http.Error(writer, "internal server error", http.StatusInternalServerError) // 500
		return
	}
	if telemetry == nil {
		handler.logger.Warn("telemetry with for this asset does not exist")
		http.NotFound(writer, request)
		return
	}

	// 2) nastavení Content-Type header
	writer.Header().Set("Content-Type", "application/json")

	// 3) zakódování a zapsání
	// Encoder zapisuje přímo do ResponseWriter streamově - nealokuje mezipaměť pro celý JSON.
	// Pro velký slice assetů/telemetrie je to efektivnější, než json.Marshal()
	if err := json.NewEncoder(writer).Encode(telemetry); err != nil {
		handler.logger.Error("failed to encode response", "error", err)
	}
}

// /assets/{id}/telemetry/history
func (handler *HTTPHandler) listTelemetryHistory(writer http.ResponseWriter, request *http.Request) {
	id := chi.URLParam(request, "id")

	// 1) volání repa
	telemetryEntries, err := handler.telemetryRepo.ListTelemetryHistory(request.Context(), id)
	if err != nil {
		handler.logger.Error("failed to get telemetry history for asset", "error", err)
		http.Error(writer, "internal server error", http.StatusInternalServerError) // 500
		return
	}

	// 2) nastavení Content-Type header
	writer.Header().Set("Content-Type", "application/json")

	// 3) zakódování a zapsání
	// Encoder zapisuje přímo do ResponseWriter streamově - nealokuje mezipaměť pro celý JSON.
	// Pro velký slice assetů/telemetrie je to efektivnější, než json.Marshal()
	if err := json.NewEncoder(writer).Encode(telemetryEntries); err != nil {
		handler.logger.Error("failed to encode response", "error", err)
	}
}

// /alerts
func (handler *HTTPHandler) listAlerts(writer http.ResponseWriter, request *http.Request) {
	// 1) volání repa
	alerts, err := handler.alertRepo.ListAlerts(request.Context())
	if err != nil {
		handler.logger.Error("failed to list alerts", "error", err)
		http.Error(writer, "internal server error", http.StatusInternalServerError) // 500
		return
	}

	// 2) nastavení Content-Type header
	writer.Header().Set("Content-Type", "application/json")

	// 3) zakódování a zapsání
	// Encoder zapisuje přímo do ResponseWriter streamově - nealokuje mezipaměť pro celý JSON.
	// Pro velký slice assetů/telemetrie je to efektivnější, než json.Marshal()
	if err := json.NewEncoder(writer).Encode(alerts); err != nil {
		handler.logger.Error("failed to encode response", "error", err)
	}
}

// /assets/{id}/alerts
func (handler *HTTPHandler) listAllAlertsForAsset(writer http.ResponseWriter, request *http.Request) {
	id := chi.URLParam(request, "id")

	// 1) volání repa
	alerts, err := handler.alertRepo.ListAllAlertsForAsset(request.Context(), id)
	if err != nil {
		handler.logger.Error("failed to get alerts for asset", "error", err)
		http.Error(writer, "internal server error", http.StatusInternalServerError) // 500
		return
	}

	// 2) nastavení Content-Type header
	writer.Header().Set("Content-Type", "application/json")

	// 3) zakódování a zapsání
	// Encoder zapisuje přímo do ResponseWriter streamově - nealokuje mezipaměť pro celý JSON.
	// Pro velký slice assetů/telemetrie je to efektivnější, než json.Marshal()
	if err := json.NewEncoder(writer).Encode(alerts); err != nil {
		handler.logger.Error("failed to encode response", "error", err)
	}
}
