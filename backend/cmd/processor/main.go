package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/BojkoJ/go-nextjs-smart-tracking-system/backend/internal/adapters/eventbus"
	"github.com/BojkoJ/go-nextjs-smart-tracking-system/backend/internal/adapters/repository"
	"github.com/BojkoJ/go-nextjs-smart-tracking-system/backend/internal/common"
	"github.com/BojkoJ/go-nextjs-smart-tracking-system/backend/internal/core/domain"
	"github.com/BojkoJ/go-nextjs-smart-tracking-system/backend/internal/services"
)

// ---------------------------------------------------------------------------------------------------------
//                                         SERVICES - PROCESSOR
// ---------------------------------------------------------------------------------------------------------
// Proč processor v cmd/:
// Processor je samostatná spustitelná binárka - má vlastní main(). Každá mikroslužba v cmd/ je nezávislý proces.
// ---------------------------------------------------------------------------------------------------------
// Co Processor Service dělá:
// Je mozkem systému - konzumuje telemetrické zprávy z NATS JetStream fronty a zpracovává je.
// Ukládá telemetrii do PostgreSQL, kontroluje business pravidla a generuje Alerty při porušení limitů.
//
// Proč NATS a ne přímé volání z Ingestu:
// Asynchronní event-driven architektura odděluje příjem dat od jejich zpracování.
// Ingest odpovídá Simulatoru okamžitě, zpracování probíhá nezávisle.
// Pokud Processor spadne, NATS zprávy čekají ve frontě - žádná telemetrie se neztratí (at-least-once delivery).
//
// Proč Durable consumer v NATS:
// Durable consumer si pamatuje, kde ve frontě skončil i po restartu Processoru.
// Bez durable consumeru by po restartu Processor zpracoval jen nové zprávy - přišel by o backlog.
//
// Dependency Injection (DI) pattern:
// main() je kompoziční kořen - vytváří konkrétní implementace a předává je jako interfaces do vrstev níže.
// ---------------------------------------------------------------------------------------------------------

func main() {
	// vytvoříme strukturovaný JSON logger pro tuto mikroslužbu
	logger := common.NewLogger("processor")
	logger.Info("Processor Microservice starting")

	// načteme konfiguraci z environment proměnných (12-Factor App princip č.3)
	// Load() selže pokud chybí povinné proměnné NATS_URL nebo POSTGRES_URL
	config, err := common.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// vytvoříme PostgreSQL repozitář - adaptér pro komunikaci s databází
	// context.Background() je vhodný pro inicializaci (není vázaný na request)
	// po vytvoření poolu každá metoda repozitáře dostane svůj vlastní ctx z handleru/consumeru
	pgRepo, err := repository.NewPostgresRepo(context.Background(), config.PostgresURL)
	if err != nil {
		logger.Error("failed to create new postgres repo", "error", err)
		os.Exit(1)
	}

	// DI: pgRepo implementuje všechny 3 repository interfaces (AssetRepository, TelemetryRepository, AlertRepository)
	// předáváme ho třikrát - jednou za každý interface který ProcessingService potřebuje
	// ProcessingService vidí jen interfaces, ne konkrétní PostgresRepo struct
	processingService := services.NewProcessingService(pgRepo, pgRepo, pgRepo, logger)

	// vytvoříme NATS klienta - adaptér pro komunikaci s NATS JetStream message brokerem
	// processor potřebuje pouze EventConsumer (konzumovat zprávy), ne EventProducer
	natsClient, err := eventbus.NewNATSClient(config.NATSUrl, logger)
	if err != nil {
		logger.Error("failed to create new NATS client", "error", err)
		os.Exit(1)
	}

	// zaregistrujeme handler pro zprávy na subjektu "telemetry.ingest"
	// Subscribe je neblokující - spustí interní NATS goroutinu a vrátí se ihned
	// handler funkce se volá pro každou příchozí zprávu:
	//   - json.Unmarshal deserializuje []byte zpět na domain.TelemetryData (Ingest serializoval JSON)
	//   - ProcessTelemetry provede business logiku: uloží telemetrii, zkontroluje limity, generuje alerty
	//   - návratová hodnota error: nil = Ack (zpráva zpracována), error = Nak (NATS zprávu znovu doručí)
	err = natsClient.Subscribe("telemetry.ingest", func(ctx context.Context, data []byte) error {
		var telemetry domain.TelemetryData
		if err := json.Unmarshal(data, &telemetry); err != nil {
			return fmt.Errorf("unmarshalling telemetry message: %w", err)
		}
		return processingService.ProcessTelemetry(ctx, telemetry)
	})
	if err != nil {
		logger.Error("failed to subscribe to NATS subject", "error", err)
		os.Exit(1)
	}

	// signal.NotifyContext vytvoří context, který se zruší při obdržení SIGINT (Ctrl+C) nebo SIGTERM (kubectl delete pod)
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// zablokujeme hlavní goroutinu - processor běží dokud nepřijde shutdown signál
	// NATS consumer běží v pozadí ve vlastní goroutině uvnitř natsClient
	logger.Info("processor microservice running, consuming telemetry.ingest...")
	<-ctx.Done()

	// Drain() je NATS graceful shutdown - počká na zpracování všech in-flight zpráv před zavřením spojení
	// na rozdíl od Connection.Close() který buffered zprávy zahazuje
	// K8s čeká 30s po SIGTERM před SIGKILL - máme dost času na drain
	logger.Info("shutdown signal received, draining NATS connection...")
	if err := natsClient.Connection.Drain(); err != nil {
		// při shutdownu logujeme chybu ale neprovádíme os.Exit - program stejně končí
		logger.Error("failed to drain NATS connection", "error", err)
	}

	logger.Info("processor microservice stopped")
}
