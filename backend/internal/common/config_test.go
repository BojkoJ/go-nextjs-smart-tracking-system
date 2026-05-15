package common

import (
	"testing"
)

func TestLoad_MissingNATSURL_ReturnsError(t *testing.T) {
	t.Setenv("NATS_URL", "")
	t.Setenv("POSTGRES_URL", "postgres://user:pass@localhost/db")
	if _, err := Load(); err == nil {
		t.Fatal("expected error when NATS_URL is missing, got nil")
	}
}

func TestLoad_MissingPostgresURL_ReturnsError(t *testing.T) {
	t.Setenv("NATS_URL", "nats://localhost:4222")
	t.Setenv("POSTGRES_URL", "")
	if _, err := Load(); err == nil {
		t.Fatal("expected error when POSTGRES_URL is missing, got nil")
	}
}

func TestLoad_BothRequired_NoError(t *testing.T) {
	t.Setenv("NATS_URL", "nats://localhost:4222")
	t.Setenv("POSTGRES_URL", "postgres://user:pass@localhost/db")
	if _, err := Load(); err != nil {
		t.Fatalf("expected no error when required vars are set, got: %v", err)
	}
}

func TestLoad_DefaultGRPCPort(t *testing.T) {
	t.Setenv("NATS_URL", "nats://localhost:4222")
	t.Setenv("POSTGRES_URL", "postgres://user:pass@localhost/db")
	t.Setenv("GRPC_PORT", "")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.GRPCPort != "50051" {
		t.Errorf("expected default GRPCPort %q, got %q", "50051", cfg.GRPCPort)
	}
}

func TestLoad_DefaultHTTPPort(t *testing.T) {
	t.Setenv("NATS_URL", "nats://localhost:4222")
	t.Setenv("POSTGRES_URL", "postgres://user:pass@localhost/db")
	t.Setenv("HTTP_PORT", "")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.HTTPPort != "8080" {
		t.Errorf("expected default HTTPPort %q, got %q", "8080", cfg.HTTPPort)
	}
}

func TestLoad_DefaultIngestAddr(t *testing.T) {
	t.Setenv("NATS_URL", "nats://localhost:4222")
	t.Setenv("POSTGRES_URL", "postgres://user:pass@localhost/db")
	t.Setenv("INGEST_ADDR", "")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.IngestAddr != "localhost:50051" {
		t.Errorf("expected default IngestAddr %q, got %q", "localhost:50051", cfg.IngestAddr)
	}
}

func TestLoad_CustomValues_OverrideDefaults(t *testing.T) {
	t.Setenv("NATS_URL", "nats://nats-server:4222")
	t.Setenv("POSTGRES_URL", "postgres://tracking_user:secret@postgresql/tracking_db")
	t.Setenv("GRPC_PORT", "9090")
	t.Setenv("HTTP_PORT", "9091")
	t.Setenv("INGEST_ADDR", "ingest-service.tracking-system.svc:50051")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.NATSUrl != "nats://nats-server:4222" {
		t.Errorf("NATSUrl: want %q, got %q", "nats://nats-server:4222", cfg.NATSUrl)
	}
	if cfg.PostgresURL != "postgres://tracking_user:secret@postgresql/tracking_db" {
		t.Errorf("PostgresURL mismatch: %q", cfg.PostgresURL)
	}
	if cfg.GRPCPort != "9090" {
		t.Errorf("GRPCPort: want %q, got %q", "9090", cfg.GRPCPort)
	}
	if cfg.HTTPPort != "9091" {
		t.Errorf("HTTPPort: want %q, got %q", "9091", cfg.HTTPPort)
	}
	if cfg.IngestAddr != "ingest-service.tracking-system.svc:50051" {
		t.Errorf("IngestAddr: want %q, got %q", "ingest-service.tracking-system.svc:50051", cfg.IngestAddr)
	}
}

func TestLoad_ReturnsPointer_NotNilOnSuccess(t *testing.T) {
	t.Setenv("NATS_URL", "nats://localhost:4222")
	t.Setenv("POSTGRES_URL", "postgres://user:pass@localhost/db")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config pointer")
	}
}
