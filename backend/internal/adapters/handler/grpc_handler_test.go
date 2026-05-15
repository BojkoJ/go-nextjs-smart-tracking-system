package handler

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/BojkoJ/go-nextjs-smart-tracking-system/backend/internal/core/domain"
	pb "github.com/BojkoJ/go-nextjs-smart-tracking-system/backend/proto"
)

type mockIngestionService struct {
	ingestFn func(ctx context.Context, td domain.TelemetryData) error
	lastCall domain.TelemetryData
}

func (m *mockIngestionService) IngestTelemetry(ctx context.Context, td domain.TelemetryData) error {
	m.lastCall = td
	if m.ingestFn != nil {
		return m.ingestFn(ctx, td)
	}
	return nil
}

func newTestGRPCHandler(svc *mockIngestionService) *GRPCHandler {
	return NewGRPCHandler(svc, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestSendTelemetry_HappyPath_ReturnsSuccessTrue(t *testing.T) {
	h := newTestGRPCHandler(&mockIngestionService{})
	resp, err := h.SendTelemetry(context.Background(), &pb.TelemetryRequest{
		AssetId:     "asset-1",
		Latitude:    35.10,
		Longitude:   129.04,
		Temperature: 22.5,
		Humidity:    38.0,
		IsLocked:    true,
		TimestampNs: 1_000_000_000,
		TraceId:     "trace-abc",
	})
	if err != nil {
		t.Fatalf("unexpected gRPC-level error: %v", err)
	}
	if !resp.Success {
		t.Errorf("expected Success=true, got false: %s", resp.Message)
	}
	if resp.Message != "OK" {
		t.Errorf("expected message %q, got %q", "OK", resp.Message)
	}
}

func TestSendTelemetry_MapsAllFieldsToTelemetryData(t *testing.T) {
	svc := &mockIngestionService{}
	h := newTestGRPCHandler(svc)

	ns := int64(1_716_700_000_000_000_000)
	_, _ = h.SendTelemetry(context.Background(), &pb.TelemetryRequest{
		AssetId:     "asset-42",
		Latitude:    51.98,
		Longitude:   4.05,
		Temperature: 25.5,
		Humidity:    40.0,
		IsLocked:    false,
		TimestampNs: ns,
		TraceId:     "trace-xyz",
	})

	got := svc.lastCall
	if got.AssetID != "asset-42" {
		t.Errorf("AssetID: want %q, got %q", "asset-42", got.AssetID)
	}
	if got.Latitude != 51.98 {
		t.Errorf("Latitude: want %v, got %v", 51.98, got.Latitude)
	}
	if got.Longitude != 4.05 {
		t.Errorf("Longitude: want %v, got %v", 4.05, got.Longitude)
	}
	if got.Temperature != 25.5 {
		t.Errorf("Temperature: want %v, got %v", 25.5, got.Temperature)
	}
	if got.Humidity != 40.0 {
		t.Errorf("Humidity: want %v, got %v", 40.0, got.Humidity)
	}
	if got.IsLocked {
		t.Error("IsLocked: want false, got true")
	}
	if got.TraceID != "trace-xyz" {
		t.Errorf("TraceID: want %q, got %q", "trace-xyz", got.TraceID)
	}
	if want := time.Unix(0, ns); !got.Timestamp.Equal(want) {
		t.Errorf("Timestamp: want %v, got %v", want, got.Timestamp)
	}
}

func TestSendTelemetry_ServiceError_ReturnsSuccessFalse_NoGRPCError(t *testing.T) {
	svc := &mockIngestionService{
		ingestFn: func(_ context.Context, _ domain.TelemetryData) error {
			return errors.New("validation failed")
		},
	}
	h := newTestGRPCHandler(svc)

	resp, err := h.SendTelemetry(context.Background(), &pb.TelemetryRequest{AssetId: "asset-1"})
	if err != nil {
		t.Fatalf("service error must not propagate as gRPC error, got: %v", err)
	}
	if resp.Success {
		t.Error("expected Success=false when service returns error")
	}
}

func TestSendTelemetry_ServiceError_MessageContainsErrorText(t *testing.T) {
	errMsg := "ingesting telemetry, got no AssetID"
	svc := &mockIngestionService{
		ingestFn: func(_ context.Context, _ domain.TelemetryData) error { return errors.New(errMsg) },
	}
	h := newTestGRPCHandler(svc)

	resp, _ := h.SendTelemetry(context.Background(), &pb.TelemetryRequest{})
	if resp.Message != errMsg {
		t.Errorf("expected message %q, got %q", errMsg, resp.Message)
	}
}

func TestSendTelemetry_ZeroTimestampNs_MapsToUnixEpoch(t *testing.T) {
	svc := &mockIngestionService{}
	h := newTestGRPCHandler(svc)

	_, _ = h.SendTelemetry(context.Background(), &pb.TelemetryRequest{AssetId: "x", TimestampNs: 0})
	if !svc.lastCall.Timestamp.Equal(time.Unix(0, 0)) {
		t.Errorf("zero TimestampNs should map to Unix epoch, got %v", svc.lastCall.Timestamp)
	}
}
