package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/BojkoJ/go-nextjs-smart-tracking-system/backend/internal/core/domain"
)

// --- mocks ---

type mockAssetRepoH struct {
	listFn    func(ctx context.Context) ([]domain.Asset, error)
	getByIDFn func(ctx context.Context, id string) (*domain.Asset, error)
}

func (m *mockAssetRepoH) ListAssets(ctx context.Context) ([]domain.Asset, error) {
	if m.listFn != nil {
		return m.listFn(ctx)
	}
	return nil, nil
}
func (m *mockAssetRepoH) GetAssetByID(ctx context.Context, id string) (*domain.Asset, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}
func (m *mockAssetRepoH) SaveAsset(_ context.Context, _ domain.Asset) error { return nil }
func (m *mockAssetRepoH) UpdateAssetStatus(_ context.Context, _ string, _ domain.AssetStatus) error {
	return nil
}

type mockTelemetryRepoH struct {
	getLastFn     func(ctx context.Context, id string) (*domain.TelemetryData, error)
	listHistoryFn func(ctx context.Context, id string, limit, offset int) ([]domain.TelemetryData, error)
}

func (m *mockTelemetryRepoH) SaveTelemetry(_ context.Context, _ domain.TelemetryData) error {
	return nil
}
func (m *mockTelemetryRepoH) GetLastTelemetry(ctx context.Context, id string) (*domain.TelemetryData, error) {
	if m.getLastFn != nil {
		return m.getLastFn(ctx, id)
	}
	return nil, nil
}
func (m *mockTelemetryRepoH) ListTelemetryHistory(ctx context.Context, id string, limit, offset int) ([]domain.TelemetryData, error) {
	if m.listHistoryFn != nil {
		return m.listHistoryFn(ctx, id, limit, offset)
	}
	return nil, nil
}

type mockAlertRepoH struct {
	listFn    func(ctx context.Context) ([]domain.Alert, error)
	listForFn func(ctx context.Context, id string) ([]domain.Alert, error)
}

func (m *mockAlertRepoH) SaveAlert(_ context.Context, _ domain.Alert) error { return nil }
func (m *mockAlertRepoH) ListAlerts(ctx context.Context) ([]domain.Alert, error) {
	if m.listFn != nil {
		return m.listFn(ctx)
	}
	return nil, nil
}
func (m *mockAlertRepoH) ListAllAlertsForAsset(ctx context.Context, id string) ([]domain.Alert, error) {
	if m.listForFn != nil {
		return m.listForFn(ctx, id)
	}
	return nil, nil
}

// --- helpers ---

func newTestHTTPHandler(ar *mockAssetRepoH, tr *mockTelemetryRepoH, alr *mockAlertRepoH) *HTTPHandler {
	return NewHTTPHandler(ar, tr, alr, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func sampleAsset() domain.Asset {
	return domain.Asset{
		ID:             "asset-1",
		Name:           "Test Container",
		MaxTemperature: 28.0,
		MinTemperature: 12.0,
		MaxHumidity:    45.0,
		Status:         domain.StatusActive,
		CreatedAt:      time.Now(),
	}
}

func do(h *HTTPHandler, method, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	h.Routes().ServeHTTP(rec, req)
	return rec
}

// --- GET /assets ---

func TestListAssets_OK_Returns200WithJSON(t *testing.T) {
	ar := &mockAssetRepoH{
		listFn: func(_ context.Context) ([]domain.Asset, error) {
			return []domain.Asset{sampleAsset()}, nil
		},
	}
	rec := do(newTestHTTPHandler(ar, &mockTelemetryRepoH{}, &mockAlertRepoH{}), http.MethodGet, "/assets")

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}
	var assets []domain.Asset
	if err := json.NewDecoder(rec.Body).Decode(&assets); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
	if len(assets) != 1 || assets[0].ID != "asset-1" {
		t.Errorf("unexpected assets response: %+v", assets)
	}
}

func TestListAssets_RepoError_Returns500(t *testing.T) {
	ar := &mockAssetRepoH{
		listFn: func(_ context.Context) ([]domain.Asset, error) { return nil, errors.New("db error") },
	}
	rec := do(newTestHTTPHandler(ar, &mockTelemetryRepoH{}, &mockAlertRepoH{}), http.MethodGet, "/assets")
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

// --- GET /assets/{id} ---

func TestGetAssetByID_Found_Returns200(t *testing.T) {
	asset := sampleAsset()
	ar := &mockAssetRepoH{
		getByIDFn: func(_ context.Context, _ string) (*domain.Asset, error) { return &asset, nil },
	}
	rec := do(newTestHTTPHandler(ar, &mockTelemetryRepoH{}, &mockAlertRepoH{}), http.MethodGet, "/assets/asset-1")

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	var got domain.Asset
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
	if got.ID != "asset-1" {
		t.Errorf("expected asset ID %q, got %q", "asset-1", got.ID)
	}
}

func TestGetAssetByID_NotFound_Returns404(t *testing.T) {
	ar := &mockAssetRepoH{
		getByIDFn: func(_ context.Context, _ string) (*domain.Asset, error) { return nil, nil },
	}
	rec := do(newTestHTTPHandler(ar, &mockTelemetryRepoH{}, &mockAlertRepoH{}), http.MethodGet, "/assets/nonexistent")
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestGetAssetByID_RepoError_Returns500(t *testing.T) {
	ar := &mockAssetRepoH{
		getByIDFn: func(_ context.Context, _ string) (*domain.Asset, error) { return nil, errors.New("db error") },
	}
	rec := do(newTestHTTPHandler(ar, &mockTelemetryRepoH{}, &mockAlertRepoH{}), http.MethodGet, "/assets/asset-1")
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

// --- GET /assets/{id}/telemetry ---

func TestGetLastTelemetry_Found_Returns200(t *testing.T) {
	td := &domain.TelemetryData{AssetID: "asset-1", Temperature: 22.0}
	tr := &mockTelemetryRepoH{
		getLastFn: func(_ context.Context, _ string) (*domain.TelemetryData, error) { return td, nil },
	}
	rec := do(newTestHTTPHandler(&mockAssetRepoH{}, tr, &mockAlertRepoH{}), http.MethodGet, "/assets/asset-1/telemetry")
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestGetLastTelemetry_NotFound_Returns404(t *testing.T) {
	tr := &mockTelemetryRepoH{
		getLastFn: func(_ context.Context, _ string) (*domain.TelemetryData, error) { return nil, nil },
	}
	rec := do(newTestHTTPHandler(&mockAssetRepoH{}, tr, &mockAlertRepoH{}), http.MethodGet, "/assets/asset-1/telemetry")
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestGetLastTelemetry_RepoError_Returns500(t *testing.T) {
	tr := &mockTelemetryRepoH{
		getLastFn: func(_ context.Context, _ string) (*domain.TelemetryData, error) { return nil, errors.New("db error") },
	}
	rec := do(newTestHTTPHandler(&mockAssetRepoH{}, tr, &mockAlertRepoH{}), http.MethodGet, "/assets/asset-1/telemetry")
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

// --- GET /assets/{id}/telemetry/history ---

func TestListTelemetryHistory_OK_Returns200(t *testing.T) {
	tr := &mockTelemetryRepoH{
		listHistoryFn: func(_ context.Context, _ string, _, _ int) ([]domain.TelemetryData, error) {
			return []domain.TelemetryData{{AssetID: "asset-1"}}, nil
		},
	}
	rec := do(newTestHTTPHandler(&mockAssetRepoH{}, tr, &mockAlertRepoH{}), http.MethodGet, "/assets/asset-1/telemetry/history")
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestListTelemetryHistory_RepoError_Returns500(t *testing.T) {
	tr := &mockTelemetryRepoH{
		listHistoryFn: func(_ context.Context, _ string, _, _ int) ([]domain.TelemetryData, error) {
			return nil, errors.New("db error")
		},
	}
	rec := do(newTestHTTPHandler(&mockAssetRepoH{}, tr, &mockAlertRepoH{}), http.MethodGet, "/assets/asset-1/telemetry/history")
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

// --- GET /alerts ---

func TestListAlerts_OK_Returns200(t *testing.T) {
	alr := &mockAlertRepoH{
		listFn: func(_ context.Context) ([]domain.Alert, error) {
			return []domain.Alert{{ID: "alert-1", Type: domain.AlertTemperatureMax}}, nil
		},
	}
	rec := do(newTestHTTPHandler(&mockAssetRepoH{}, &mockTelemetryRepoH{}, alr), http.MethodGet, "/alerts")
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestListAlerts_RepoError_Returns500(t *testing.T) {
	alr := &mockAlertRepoH{
		listFn: func(_ context.Context) ([]domain.Alert, error) { return nil, errors.New("db error") },
	}
	rec := do(newTestHTTPHandler(&mockAssetRepoH{}, &mockTelemetryRepoH{}, alr), http.MethodGet, "/alerts")
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

// --- GET /assets/{id}/alerts ---

func TestListAllAlertsForAsset_OK_Returns200(t *testing.T) {
	alr := &mockAlertRepoH{
		listForFn: func(_ context.Context, id string) ([]domain.Alert, error) {
			return []domain.Alert{{ID: "alert-1", AssetID: id}}, nil
		},
	}
	rec := do(newTestHTTPHandler(&mockAssetRepoH{}, &mockTelemetryRepoH{}, alr), http.MethodGet, "/assets/asset-1/alerts")
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestListAllAlertsForAsset_RepoError_Returns500(t *testing.T) {
	alr := &mockAlertRepoH{
		listForFn: func(_ context.Context, _ string) ([]domain.Alert, error) { return nil, errors.New("db error") },
	}
	rec := do(newTestHTTPHandler(&mockAssetRepoH{}, &mockTelemetryRepoH{}, alr), http.MethodGet, "/assets/asset-1/alerts")
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

// --- routing sanity ---

func TestRoutes_UnknownPath_Returns404(t *testing.T) {
	h := newTestHTTPHandler(&mockAssetRepoH{}, &mockTelemetryRepoH{}, &mockAlertRepoH{})
	rec := do(h, http.MethodGet, "/nonexistent")
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for unknown route, got %d", rec.Code)
	}
}
