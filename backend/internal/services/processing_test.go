package services

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/BojkoJ/go-nextjs-smart-tracking-system/backend/internal/core/domain"
)

// --- mocks ---

type mockAssetRepo struct {
	getByIDFn func(ctx context.Context, id string) (*domain.Asset, error)
}

func (m *mockAssetRepo) GetAssetByID(ctx context.Context, id string) (*domain.Asset, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}
func (m *mockAssetRepo) SaveAsset(_ context.Context, _ domain.Asset) error    { return nil }
func (m *mockAssetRepo) ListAssets(_ context.Context) ([]domain.Asset, error) { return nil, nil }

type mockTelemetryRepo struct {
	saveFn func(ctx context.Context, t domain.TelemetryData) error
}

func (m *mockTelemetryRepo) SaveTelemetry(ctx context.Context, t domain.TelemetryData) error {
	if m.saveFn != nil {
		return m.saveFn(ctx, t)
	}
	return nil
}
func (m *mockTelemetryRepo) GetLastTelemetry(_ context.Context, _ string) (*domain.TelemetryData, error) {
	return nil, nil
}
func (m *mockTelemetryRepo) ListTelemetryHistory(_ context.Context, _ string) ([]domain.TelemetryData, error) {
	return nil, nil
}

type mockAlertRepo struct {
	savedAlerts []domain.Alert
	saveFn      func(ctx context.Context, alert domain.Alert) error
}

func (m *mockAlertRepo) SaveAlert(ctx context.Context, alert domain.Alert) error {
	if m.saveFn != nil {
		return m.saveFn(ctx, alert)
	}
	m.savedAlerts = append(m.savedAlerts, alert)
	return nil
}
func (m *mockAlertRepo) ListAlerts(_ context.Context) ([]domain.Alert, error) { return nil, nil }
func (m *mockAlertRepo) ListAllAlertsForAsset(_ context.Context, _ string) ([]domain.Alert, error) {
	return nil, nil
}

// --- helpers ---

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func defaultAsset() *domain.Asset {
	return &domain.Asset{
		ID:             "asset-1",
		Name:           "Test Container",
		MaxTemperature: 28.0,
		MinTemperature: 12.0,
		MaxHumidity:    45.0,
		Status:         domain.StatusActive,
	}
}

func normalTelemetry() domain.TelemetryData {
	return domain.TelemetryData{
		AssetID:     "asset-1",
		Temperature: 20.0, // within [12, 28]
		Humidity:    35.0, // within [0, 45]
		IsLocked:    true,
	}
}

func buildSvc(asset *domain.Asset, assetErr error, saveTelemErr error, alertRepo *mockAlertRepo) *ProcessingServiceImpl {
	assetRepo := &mockAssetRepo{
		getByIDFn: func(_ context.Context, _ string) (*domain.Asset, error) { return asset, assetErr },
	}
	telemRepo := &mockTelemetryRepo{
		saveFn: func(_ context.Context, _ domain.TelemetryData) error { return saveTelemErr },
	}
	return NewProcessingService(assetRepo, telemRepo, alertRepo, discardLogger())
}

// --- tests ---

func TestProcessTelemetry_AssetNotFound_ReturnsNil(t *testing.T) {
	svc := buildSvc(nil, nil, nil, &mockAlertRepo{})
	if err := svc.ProcessTelemetry(context.Background(), normalTelemetry()); err != nil {
		t.Errorf("expected nil for unknown asset, got: %v", err)
	}
}

func TestProcessTelemetry_AssetRepoError_ReturnsWrappedError(t *testing.T) {
	want := errors.New("db connection lost")
	svc := buildSvc(nil, want, nil, &mockAlertRepo{})
	err := svc.ProcessTelemetry(context.Background(), normalTelemetry())
	if err == nil {
		t.Fatal("expected error when asset repo fails, got nil")
	}
	if !errors.Is(err, want) {
		t.Errorf("expected error chain to contain repo error, got: %v", err)
	}
}

func TestProcessTelemetry_SaveTelemetryError_ReturnsWrappedError(t *testing.T) {
	want := errors.New("disk full")
	svc := buildSvc(defaultAsset(), nil, want, &mockAlertRepo{})
	err := svc.ProcessTelemetry(context.Background(), normalTelemetry())
	if err == nil {
		t.Fatal("expected error when saving telemetry fails, got nil")
	}
	if !errors.Is(err, want) {
		t.Errorf("expected error chain to contain save error, got: %v", err)
	}
}

func TestProcessTelemetry_TemperatureAboveMax_CreatesMaxAlert(t *testing.T) {
	alerts := &mockAlertRepo{}
	svc := buildSvc(defaultAsset(), nil, nil, alerts)

	td := normalTelemetry()
	td.Temperature = 29.0 // > MaxTemperature 28.0
	if err := svc.ProcessTelemetry(context.Background(), td); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(alerts.savedAlerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(alerts.savedAlerts))
	}
	if alerts.savedAlerts[0].Type != domain.AlertTemperatureMax {
		t.Errorf("expected alert type %q, got %q", domain.AlertTemperatureMax, alerts.savedAlerts[0].Type)
	}
}

func TestProcessTelemetry_TemperatureBelowMin_CreatesMinAlert(t *testing.T) {
	alerts := &mockAlertRepo{}
	svc := buildSvc(defaultAsset(), nil, nil, alerts)

	td := normalTelemetry()
	td.Temperature = 11.0 // < MinTemperature 12.0
	if err := svc.ProcessTelemetry(context.Background(), td); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(alerts.savedAlerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(alerts.savedAlerts))
	}
	if alerts.savedAlerts[0].Type != domain.AlertTemperatureMin {
		t.Errorf("expected alert type %q, got %q", domain.AlertTemperatureMin, alerts.savedAlerts[0].Type)
	}
}

func TestProcessTelemetry_Unlocked_CreatesUnlockedAlert(t *testing.T) {
	alerts := &mockAlertRepo{}
	svc := buildSvc(defaultAsset(), nil, nil, alerts)

	td := normalTelemetry()
	td.IsLocked = false
	if err := svc.ProcessTelemetry(context.Background(), td); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(alerts.savedAlerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(alerts.savedAlerts))
	}
	if alerts.savedAlerts[0].Type != domain.AlertUnlocked {
		t.Errorf("expected alert type %q, got %q", domain.AlertUnlocked, alerts.savedAlerts[0].Type)
	}
}

func TestProcessTelemetry_HumidityAboveMax_CreatesHumidityAlert(t *testing.T) {
	alerts := &mockAlertRepo{}
	svc := buildSvc(defaultAsset(), nil, nil, alerts)

	td := normalTelemetry()
	td.Humidity = 46.0 // > MaxHumidity 45.0
	if err := svc.ProcessTelemetry(context.Background(), td); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(alerts.savedAlerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(alerts.savedAlerts))
	}
	if alerts.savedAlerts[0].Type != domain.AlertHumidityMax {
		t.Errorf("expected alert type %q, got %q", domain.AlertHumidityMax, alerts.savedAlerts[0].Type)
	}
}

func TestProcessTelemetry_MultipleViolations_CreatesAllAlerts(t *testing.T) {
	alerts := &mockAlertRepo{}
	svc := buildSvc(defaultAsset(), nil, nil, alerts)

	td := domain.TelemetryData{
		AssetID:     "asset-1",
		Temperature: 35.0,  // > MaxTemperature
		Humidity:    50.0,  // > MaxHumidity
		IsLocked:    false, // unlocked
	}
	if err := svc.ProcessTelemetry(context.Background(), td); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(alerts.savedAlerts) != 3 {
		t.Fatalf("expected 3 alerts (temp_max, unlocked, humidity_max), got %d", len(alerts.savedAlerts))
	}
	byType := make(map[domain.AlertType]bool)
	for _, a := range alerts.savedAlerts {
		byType[a.Type] = true
	}
	for _, expected := range []domain.AlertType{domain.AlertTemperatureMax, domain.AlertUnlocked, domain.AlertHumidityMax} {
		if !byType[expected] {
			t.Errorf("missing expected alert type %q", expected)
		}
	}
}

func TestProcessTelemetry_AlertContainsCorrectAssetID(t *testing.T) {
	alerts := &mockAlertRepo{}
	svc := buildSvc(defaultAsset(), nil, nil, alerts)

	td := normalTelemetry()
	td.Temperature = 35.0
	if err := svc.ProcessTelemetry(context.Background(), td); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if alerts.savedAlerts[0].AssetID != "asset-1" {
		t.Errorf("alert AssetID: want %q, got %q", "asset-1", alerts.savedAlerts[0].AssetID)
	}
}

func TestProcessTelemetry_AlertHasNonEmptyID(t *testing.T) {
	alerts := &mockAlertRepo{}
	svc := buildSvc(defaultAsset(), nil, nil, alerts)

	td := normalTelemetry()
	td.Temperature = 35.0
	if err := svc.ProcessTelemetry(context.Background(), td); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if alerts.savedAlerts[0].ID == "" {
		t.Error("alert ID should be a non-empty UUID")
	}
}

func TestProcessTelemetry_SaveAlertError_ReturnsWrappedError(t *testing.T) {
	want := errors.New("alert write failed")
	alerts := &mockAlertRepo{
		saveFn: func(_ context.Context, _ domain.Alert) error { return want },
	}
	svc := buildSvc(defaultAsset(), nil, nil, alerts)

	td := normalTelemetry()
	td.Temperature = 35.0
	err := svc.ProcessTelemetry(context.Background(), td)
	if err == nil {
		t.Fatal("expected error when saving alert fails, got nil")
	}
	if !errors.Is(err, want) {
		t.Errorf("expected error chain to contain alert save error, got: %v", err)
	}
}

func TestProcessTelemetry_WithinAllLimits_NoAlerts(t *testing.T) {
	alerts := &mockAlertRepo{}
	svc := buildSvc(defaultAsset(), nil, nil, alerts)

	if err := svc.ProcessTelemetry(context.Background(), normalTelemetry()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(alerts.savedAlerts) != 0 {
		t.Errorf("expected 0 alerts for normal telemetry, got %d", len(alerts.savedAlerts))
	}
}

func TestProcessTelemetry_ExactlyAtMaxTemperature_NoAlert(t *testing.T) {
	alerts := &mockAlertRepo{}
	svc := buildSvc(defaultAsset(), nil, nil, alerts)

	td := normalTelemetry()
	td.Temperature = 28.0 // exactly at MaxTemperature — condition is >, not >=
	if err := svc.ProcessTelemetry(context.Background(), td); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(alerts.savedAlerts) != 0 {
		t.Errorf("exactly at limit should not trigger alert, got %d alerts", len(alerts.savedAlerts))
	}
}

func TestProcessTelemetry_ExactlyAtMinTemperature_NoAlert(t *testing.T) {
	alerts := &mockAlertRepo{}
	svc := buildSvc(defaultAsset(), nil, nil, alerts)

	td := normalTelemetry()
	td.Temperature = 12.0 // exactly at MinTemperature — condition is <, not <=
	if err := svc.ProcessTelemetry(context.Background(), td); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(alerts.savedAlerts) != 0 {
		t.Errorf("exactly at limit should not trigger alert, got %d alerts", len(alerts.savedAlerts))
	}
}
