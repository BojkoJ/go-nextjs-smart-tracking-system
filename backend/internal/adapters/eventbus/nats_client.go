package eventbus

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/BojkoJ/go-nextjs-smart-tracking-system/backend/internal/core/ports"
	"github.com/nats-io/nats.go"
)

// ---------------------------------------------------------------------------------------------------------
//                                         ADAPTERS - EVENTBUS/NATS-CLIENT
// ---------------------------------------------------------------------------------------------------------
// Proč tento soubor implementuje dva interfacy? NATSClient je single connection k NATS serveru.
// Ingest Service z něj potřebuje jen Publish.
// Processor Service jen Subscribe.
// Přesto je logické mít jednu implementaci - jedno spojení, dva způsoby použití.
// ---------------------------------------------------------------------------------------------------------

type NATSClient struct {
	Connection       *nats.Conn
	JetStreamContext nats.JetStreamContext
	logger           *slog.Logger
}

var _ ports.EventProducer = (*NATSClient)(nil)
var _ ports.EventConsumer = (*NATSClient)(nil)

func NewNATSClient(url string, logger *slog.Logger) (*NATSClient, error) {
	natsConn, err := nats.Connect(url)
	if err != nil {
		return nil, fmt.Errorf("connecting to NATS: %w", err)
	}

	jetStream, err := natsConn.JetStream()
	if err != nil {
		return nil, fmt.Errorf("creating JetStream context: %w", err)
	}

	// Vytvoříme stream pokud neexistuje
	// Idempotence: pokud stream už existuje (třeba po restartu služby), AddStream by vrátil error.
	// Takže nejdříve zkusíme najít, teprve pokud neexistuje, vytvoříme.
	// "telemetry.>" znamená "všechny subjekty začínající "telemetry."
	_, err = jetStream.StreamInfo("TELEMETRY")
	if err != nil {
		// Neexistuje
		_, err := jetStream.AddStream(&nats.StreamConfig{
			Name:     "TELEMETRY",
			Subjects: []string{"telemetry.>"},
			Storage:  nats.FileStorage,
		})
		if err != nil {
			return nil, fmt.Errorf("creating TELEMETRY stream: %w", err)
		}
	}

	return &NATSClient{Connection: natsConn, JetStreamContext: jetStream, logger: logger}, nil
}

func (client *NATSClient) PublishMessage(_ context.Context, subject string, data []byte) error {
	// NATSClient nemá vlastní service jméno - sdílí ho totiž services Ingest i Processor, takže zde nevytváříme nový logge,
	// protože ten by potřeboval jméno služby (common.NewLogger(..)), ale bereme ho přes parametr

	// ctx zatím nepoužíváme
	_, err := client.JetStreamContext.Publish(subject, data)
	if err != nil {
		return fmt.Errorf("publishing message to JetStream: %w", err)
	}
	return nil
}

func (client *NATSClient) Subscribe(subject string, handler func(context.Context, []byte) error) error {
	_, err := client.JetStreamContext.Subscribe(subject, func(msg *nats.Msg) {
		if err := handler(context.Background(), msg.Data); err != nil {
			client.logger.Error("failed to process message", "subject", subject, "error", err)
			if nakErr := msg.Nak(); nakErr != nil {
				client.logger.Error("failed to nak message", "error", nakErr)
			}
			return
		}
		if ackErr := msg.Ack(); ackErr != nil {
			client.logger.Error("failed to ack message", "error", ackErr)
		}
	}, nats.Durable("processor-consumer"))

	return err
}
