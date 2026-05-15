package services

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"testing"

	"github.com/BojkoJ/go-nextjs-smart-tracking-system/backend/internal/core/domain"
)

type mockEventProducer struct {
	publishFn func(ctx context.Context, subject string, data []byte) error
}

func (m *mockEventProducer) PublishMessage(ctx context.Context, subject string, data []byte) error {
	if m.publishFn != nil {
		return m.publishFn(ctx, subject, data)
	}
	return nil
}

func validTelemetry() domain.TelemetryData {
	return domain.TelemetryData{
		AssetID:     "asset-1",
		Latitude:    35.10,
		Longitude:   129.04,
		Temperature: 22.5,
		Humidity:    38.0,
		IsLocked:    true,
		TraceID:     "trace-abc",
	}
}

func TestIngestTelemetry_EmptyAssetID(t *testing.T) {
	svc := NewIngestionService(&mockEventProducer{})
	td := validTelemetry()
	td.AssetID = ""
	if err := svc.IngestTelemetry(context.Background(), td); err == nil {
		t.Fatal("expected error for empty AssetID, got nil")
	}
}

func TestIngestTelemetry_NaNTemperature(t *testing.T) {
	svc := NewIngestionService(&mockEventProducer{})
	td := validTelemetry()
	td.Temperature = math.NaN()
	if err := svc.IngestTelemetry(context.Background(), td); err == nil {
		t.Fatal("expected error for NaN temperature, got nil")
	}
}

func TestIngestTelemetry_LatitudeTooHigh(t *testing.T) {
	svc := NewIngestionService(&mockEventProducer{})
	td := validTelemetry()
	td.Latitude = 90.1
	if err := svc.IngestTelemetry(context.Background(), td); err == nil {
		t.Fatal("expected error for latitude > 90")
	}
}

func TestIngestTelemetry_LatitudeTooLow(t *testing.T) {
	svc := NewIngestionService(&mockEventProducer{})
	td := validTelemetry()
	td.Latitude = -90.1
	if err := svc.IngestTelemetry(context.Background(), td); err == nil {
		t.Fatal("expected error for latitude < -90")
	}
}

func TestIngestTelemetry_LongitudeTooHigh(t *testing.T) {
	svc := NewIngestionService(&mockEventProducer{})
	td := validTelemetry()
	td.Longitude = 180.1
	if err := svc.IngestTelemetry(context.Background(), td); err == nil {
		t.Fatal("expected error for longitude > 180")
	}
}

func TestIngestTelemetry_LongitudeTooLow(t *testing.T) {
	svc := NewIngestionService(&mockEventProducer{})
	td := validTelemetry()
	td.Longitude = -180.1
	if err := svc.IngestTelemetry(context.Background(), td); err == nil {
		t.Fatal("expected error for longitude < -180")
	}
}

func TestIngestTelemetry_BoundaryLatitudeIsValid(t *testing.T) {
	svc := NewIngestionService(&mockEventProducer{})
	for _, lat := range []float64{90.0, -90.0} {
		td := validTelemetry()
		td.Latitude = lat
		if err := svc.IngestTelemetry(context.Background(), td); err != nil {
			t.Errorf("expected no error at boundary latitude %v, got: %v", lat, err)
		}
	}
}

func TestIngestTelemetry_BoundaryLongitudeIsValid(t *testing.T) {
	svc := NewIngestionService(&mockEventProducer{})
	for _, lon := range []float64{180.0, -180.0} {
		td := validTelemetry()
		td.Longitude = lon
		if err := svc.IngestTelemetry(context.Background(), td); err != nil {
			t.Errorf("expected no error at boundary longitude %v, got: %v", lon, err)
		}
	}
}

func TestIngestTelemetry_PublisherError_IsWrapped(t *testing.T) {
	want := errors.New("nats connection lost")
	svc := NewIngestionService(&mockEventProducer{
		publishFn: func(_ context.Context, _ string, _ []byte) error { return want },
	})
	err := svc.IngestTelemetry(context.Background(), validTelemetry())
	if err == nil {
		t.Fatal("expected error when publisher fails, got nil")
	}
	if !errors.Is(err, want) {
		t.Errorf("expected error chain to contain publisher error, got: %v", err)
	}
}

func TestIngestTelemetry_PublishesCorrectSubject(t *testing.T) {
	var gotSubject string
	svc := NewIngestionService(&mockEventProducer{
		publishFn: func(_ context.Context, subject string, _ []byte) error {
			gotSubject = subject
			return nil
		},
	})
	if err := svc.IngestTelemetry(context.Background(), validTelemetry()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotSubject != "telemetry.ingest" {
		t.Errorf("expected subject %q, got %q", "telemetry.ingest", gotSubject)
	}
}

func TestIngestTelemetry_PayloadIsValidJSON(t *testing.T) {
	var gotData []byte
	svc := NewIngestionService(&mockEventProducer{
		publishFn: func(_ context.Context, _ string, data []byte) error {
			gotData = data
			return nil
		},
	})
	td := validTelemetry()
	if err := svc.IngestTelemetry(context.Background(), td); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var out domain.TelemetryData
	if err := json.Unmarshal(gotData, &out); err != nil {
		t.Fatalf("published payload is not valid JSON: %v", err)
	}
	if out.AssetID != td.AssetID {
		t.Errorf("AssetID mismatch in payload: want %q, got %q", td.AssetID, out.AssetID)
	}
}

func TestIngestTelemetry_ContextPassedToPublisher(t *testing.T) {
	type ctxKey struct{}
	sentinel := "sentinel-value"
	var gotCtx context.Context
	svc := NewIngestionService(&mockEventProducer{
		publishFn: func(ctx context.Context, _ string, _ []byte) error {
			gotCtx = ctx
			return nil
		},
	})
	ctx := context.WithValue(context.Background(), ctxKey{}, sentinel)
	_ = svc.IngestTelemetry(ctx, validTelemetry())
	if gotCtx == nil || gotCtx.Value(ctxKey{}) != sentinel {
		t.Error("context was not propagated to publisher")
	}
}
