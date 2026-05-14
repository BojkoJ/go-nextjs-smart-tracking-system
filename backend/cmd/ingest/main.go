package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/BojkoJ/go-nextjs-smart-tracking-system/backend/internal/adapters/eventbus"
	"github.com/BojkoJ/go-nextjs-smart-tracking-system/backend/internal/adapters/handler"
	"github.com/BojkoJ/go-nextjs-smart-tracking-system/backend/internal/common"
	"github.com/BojkoJ/go-nextjs-smart-tracking-system/backend/internal/services"
	pb "github.com/BojkoJ/go-nextjs-smart-tracking-system/backend/proto"
	"google.golang.org/grpc"
)

// ---------------------------------------------------------------------------------------------------------
//                                         SERVICES - INGEST
// ---------------------------------------------------------------------------------------------------------
// Proč ingest v cmd/:
// Ingest je samostatná spustitelná binárka - má vlastní main(). Každá mikroslužba v cmd/ je nezávislý proces.
// ---------------------------------------------------------------------------------------------------------
// Co Ingest Service dělá:
// Je vstupní branou systému pro příjem telemetrie.
// Přijímá gRPC volání od Simulatoru → předává data do NATS JetStream fronty → vrací odpověď Simulatoru.
// Samotné zpracování (ukládání do DB, generování alertů) řeší Processor Service asynchronně.
//
// Proč gRPC a ne HTTP:
// Simulator posílá 50 requestů za sekundu (1 za kontejner). gRPC je binární protokol (Protobuf) -
// přibližně 5-10x méně dat než JSON/HTTP. gRPC runtime navíc spouští každý příchozí request
// automaticky ve vlastní goroutině - bez jediného řádku navíc zvládáme 50 paralelních volání.
//
// Dependency Injection (DI) pattern:
// main() je kompoziční kořen - jediné místo, kde se vytváří konkrétní implementace a předávají
// jako interfaces do vrstev níže. Žádná vrstva (service, handler) nevytváří své závislosti sama.
// Výsledek: vrstvy jsou testovatelné a nezávislé na konkrétní infrastruktuře.
// ---------------------------------------------------------------------------------------------------------

func main() {
	// vytvoříme strukturovaný JSON logger pro tuto mikroslužbu
	logger := common.NewLogger("ingest")
	logger.Info("Ingest Microservice starting")

	// načteme konfiguraci z environment proměnných (12-Factor App princip č.3)
	// Load() selže pokud chybí povinné proměnné NATS_URL nebo POSTGRES_URL
	config, err := common.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// vytvoříme NATS klienta - adaptér pro komunikaci s NATS JetStream message brokerem
	// ingest potřebuje pouze EventProducer (publikovat zprávy), ne EventConsumer
	natsClient, err := eventbus.NewNATSClient(config.NATSUrl, logger)
	if err != nil {
		logger.Error("failed to create new NATS client", "error", err)
		os.Exit(1)
	}

	// DI: vytvoříme IngestionService a předáme mu NATS klienta jako EventProducer interface
	// service neví o konkrétním NATSClient - zná jen interface ports.EventProducer
	ingestionService := services.NewIngestionService(natsClient)

	// DI: vytvoříme gRPC handler a předáme mu IngestionService jako interface
	// handler překládá pb.TelemetryRequest -> domain.TelemetryData a volá service
	grpcHandler := handler.NewGRPCHandler(ingestionService, logger)

	// vytvoříme TCP listener na portu z konfigurace (default: 50051)
	// net.Listen odděluje Listen od Serve - ověříme dostupnost portu před spuštěním serveru
	listener, err := net.Listen("tcp", ":"+config.GRPCPort)
	if err != nil {
		logger.Error("failed to create TCP listener", "error", err, "port", config.GRPCPort)
		os.Exit(1)
	}

	// vytvoříme gRPC server
	grpcServer := grpc.NewServer()

	// zaregistrujeme náš handler do gRPC serveru - říkáme: "příchozí TelemetryService requesty zpracuj přes grpcHandler"
	pb.RegisterTelemetryServiceServer(grpcServer, grpcHandler)

	// signal.NotifyContext vytvoří context, který se zruší při obdržení SIGINT (Ctrl+C) nebo SIGTERM (kubectl delete pod)
	// defer cancel() uvolní zdroje contextu při ukončení main()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// spustíme gRPC server v goroutině, protože Serve() blokuje
	// bez goroutiny bychom nikdy nedošli k <-ctx.Done() a graceful shutdown by nefungoval
	go func() {
		logger.Info("gRPC server listening", "port", config.GRPCPort)
		if err := grpcServer.Serve(listener); err != nil {
			logger.Error("gRPC server error", "error", err)
		}
	}()

	// zablokujeme hlavní goroutinu dokud nepřijde shutdown signál
	<-ctx.Done()
	logger.Info("shutdown signal received, stopping gRPC server...")

	// GracefulStop počká na dokončení všech in-flight gRPC requestů před ukončením
	// na rozdíl od Stop() který requesty zahazuje okamžitě
	// K8s čeká 30s po SIGTERM před SIGKILL - máme dost času na clean ukončení
	grpcServer.GracefulStop()
	logger.Info("ingest microservice stopped")
}
