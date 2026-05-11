package common

import (
	"log/slog"
	"os"
)

// ---------------------------------------------------------------------------------------------------------
//                                    COMMON UTILITIES - LOGGER
// ---------------------------------------------------------------------------------------------------------
// Proč slog místo fmt.Println:
// Println vypíše prostý text, v K3s logu mezi stovkami řádků je to nedohledatelné. slog vypíše JSON.
// Grafana Loki pak umí filtrovat, např. logy, kde service=processor a level=ERROR
// ---------------------------------------------------------------------------------------------------------

func NewLogger(service string) *slog.Logger {
	// Každá služba si vytvoří svůj logger s vlastním jménem.
	// Logger pak ke každému záznamu automaticky přidá "service":"ingest" (nebo processor, simulator...)
	// V Grafaně pak filtr podle service funguje seamlessly.

	// JSON handler
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo, // loguj INFO a výše (ne DEBUG)
	})

	// vytvoříme logger a přidáme mu atribut service.
	// tricky part: metoda .With() přidává trvalé atributy ke každému budoucímu záznamu:
	return slog.New(handler).With("service", service)
}
