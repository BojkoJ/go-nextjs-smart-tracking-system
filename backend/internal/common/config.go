package common

import (
	"fmt"
	"os"
)

// ---------------------------------------------------------------------------------------------------------
//                                    COMMON UTILITIES - CONFIG
// ---------------------------------------------------------------------------------------------------------
// Proč tyto soubory existují jako common:
// Každá ze čtyř služeb (ingest, processor, query, simulator) potřebuje načíst konfiguraci a logovat.
// Bez common by každá služba psala stejný kód znovu - porušení DRY principu (Don't Repeat Yourself).
// common je sdílená knihovna uvnitř projektu.
// ---------------------------------------------------------------------------------------------------------
// Proč config z env proměnných:
// To je 12-Factor App princip č.3: konfiguraci patří do prostředí, ne do kódu.
// V k3s předáme hodnoty přes env: v deployment manifestu.
// Lokálně nastavíme proměnné v terminálu. Kód se nemění, mění se prostředí.
// ---------------------------------------------------------------------------------------------------------

type Config struct {
	NATSUrl     string // Adresa NATS Serveru
	PostgresURL string // Connection String pro Postrges
	GRPCPort    string // port, na kterém Ingest Service poslouchá gRPC
	HTTPPort    string // port, na kterém Query Service poslouchá HTTP
	IngestAddr  string
}

func Load() (*Config, error) {
	// Vrací pointer proto, že Config je struct, který předáváme dál.

	cfg := &Config{
		NATSUrl:     os.Getenv("NATS_URL"),
		PostgresURL: os.Getenv("POSTGRES_URL"),
		GRPCPort:    os.Getenv("GRPC_PORT"),
		HTTPPort:    os.Getenv("HTTP_PORT"),
		IngestAddr:  os.Getenv("INGEST_ADDR"),
	}

	// os.Getenv vrátí prázdný string pokud proměnná neexistuje.
	// Musíme validovat pro každé povinné pole
	// Pro nepovinné pole pouze dáme default value

	if cfg.NATSUrl == "" {
		return nil, fmt.Errorf("NATS_URL Envirloment Variable is required")
	}
	if cfg.PostgresURL == "" {
		return nil, fmt.Errorf("POSTGRES_URL Envirloment Variable is required")
	}
	// Default values:
	if cfg.GRPCPort == "" {
		cfg.GRPCPort = "50051"
	}
	if cfg.HTTPPort == "" {
		cfg.HTTPPort = "8080"
	}
	if cfg.IngestAddr == "" {
		cfg.IngestAddr = "localhost:50051"
	}

	return cfg, nil
}
