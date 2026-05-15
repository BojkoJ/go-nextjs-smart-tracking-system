package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/BojkoJ/go-nextjs-smart-tracking-system/backend/internal/adapters/handler"
	"github.com/BojkoJ/go-nextjs-smart-tracking-system/backend/internal/adapters/repository"
	"github.com/BojkoJ/go-nextjs-smart-tracking-system/backend/internal/common"
)

// ---------------------------------------------------------------------------------------------------------
//                                         SERVICES - QUERY
// ---------------------------------------------------------------------------------------------------------
// Proč query v cmd/:
// Query je samostatná spustitelná binárka - má vlastní main(). Každá mikroslužba v cmd/ je nezávislý proces.
// ---------------------------------------------------------------------------------------------------------
// Co Query Service dělá:
// Obsluhuje REST HTTP požadavky od frontend aplikace - vrací assety, telemetrii a alerty z PostgreSQL.
//
// Proč Query čte přímo z DB a ne přes Processor:
// Tohle je CQRS pattern (Command Query Responsibility Segregation):
//   - Commands (zápis): Simulator → Ingest (gRPC) → NATS → Processor → PostgreSQL
//   - Queries (čtení): Frontend → Query (HTTP) → PostgreSQL přímo
// Výhoda: čtení neblokuje zápis, obě cesty se škálují nezávisle.
// Query Service nepotřebuje NATS - nemá co publikovat ani konzumovat.
//
// Proč http.Server struct a ne http.ListenAndServe():
// http.ListenAndServe() je convenience funkce bez graceful shutdown.
// http.Server struct poskytuje metodu Shutdown(ctx) která počká na dokončení
// aktivních requestů před zavřením serveru.
//
// Dependency Injection (DI) pattern:
// main() je kompoziční kořen - vytváří konkrétní implementace a předává je jako interfaces do vrstev níže.
// ---------------------------------------------------------------------------------------------------------

func main() {
	// vytvoříme strukturovaný JSON logger pro tuto mikroslužbu
	logger := common.NewLogger("query")
	logger.Info("Query Microservice starting")

	otelShutdown, err := common.InitTracerProvider(context.Background(), "query")
	if err != nil {
		logger.Error("failed to init tracer provider", "error", err)
		os.Exit(1)
	}
	defer func() { _ = otelShutdown(context.Background()) }()

	// načteme konfiguraci z environment proměnných (12-Factor App princip č.3)
	// Load() selže pokud chybí povinné proměnné NATS_URL nebo POSTGRES_URL
	config, err := common.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// vytvoříme PostgreSQL repozitář - adaptér pro komunikaci s databází
	// context.Background() je vhodný pro inicializaci (není vázaný na request)
	// po vytvoření poolu každá metoda repozitáře dostane svůj vlastní ctx z HTTP requestu
	pgRepo, err := repository.NewPostgresRepo(context.Background(), config.PostgresURL)
	if err != nil {
		logger.Error("failed to create new postgres repo", "error", err)
		os.Exit(1)
	}

	// DI: pgRepo implementuje všechny 3 repository interfaces (AssetRepository, TelemetryRepository, AlertRepository)
	// předáváme ho třikrát - jednou za každý interface který HTTPHandler potřebuje
	// HTTPHandler vidí jen interfaces, ne konkrétní PostgresRepo struct
	httpHandler := handler.NewHTTPHandler(pgRepo, pgRepo, pgRepo, logger)

	// vytvoříme HTTP server s adresou a routerem z HTTPHandleru
	// httpHandler.Routes() vrátí chi router se všemi zaregistrovanými endpointy
	httpServer := &http.Server{
		Addr:    ":" + config.HTTPPort,
		Handler: httpHandler.Routes(),
	}

	// signal.NotifyContext vytvoří context, který se zruší při obdržení SIGINT (Ctrl+C) nebo SIGTERM (kubectl delete pod)
	// defer cancel() uvolní zdroje contextu při ukončení main()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// spustíme HTTP server v goroutině, protože ListenAndServe() blokuje
	// bez goroutiny bychom nikdy nedošli k <-ctx.Done() a graceful shutdown by nefungoval
	// errors.Is(err, http.ErrServerClosed): ListenAndServe vrátí ErrServerClosed po zavolání Shutdown()
	// to je očekávaný stav, ne chyba - bez tohoto checku bychom logovali false alarm při každém shutdownu
	go func() {
		logger.Info("HTTP server listening", "port", config.HTTPPort)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("HTTP server error", "error", err)
		}
	}()

	// zablokujeme hlavní goroutinu dokud nepřijde shutdown signál
	<-ctx.Done()
	logger.Info("shutdown signal received, stopping HTTP server...")

	// vytvoříme nový context s timeoutem pro Shutdown - nezávislý na už zrušeném ctx
	// Shutdown(ctx) počká na dokončení aktivních HTTP requestů, max však 10 sekund
	// K8s čeká 30s po SIGTERM před SIGKILL - 10s timeout dává prostor pro dokončení DB queries
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server shutdown error", "error", err)
	}

	logger.Info("query microservice stopped")
}
