package ports

import (
	"context"
)

// ---------------------------------------------------------------------------------------------------------
//                                             EVENT BUS
// ---------------------------------------------------------------------------------------------------------
// EventBus pracuje na nižší úrovni - posílá surová data jako []byte.
// Je to záměrné: EventProducer neví co posílá, jen pošle byty na daný subject.
// Serializaci (co je v těch bytech) řeší vrstva nad ním.
// ---------------------------------------------------------------------------------------------------------

// EventProducer bude publikovat zprávy
type EventProducer interface {
	// PublishMessage přijímá kontext, název NATS topicu a data
	PublishMessage(ctx context.Context, subject string, data []byte) error
}

type EventConsumer interface {
	// Subscribe je operace, která nastaví callback jednou a ten pak běží napořád.
	// Context pro individuální zprávy nese handler funkce sama (vidíme to v signatuře handler func(..).
	Subscribe(subject string, handler func(ctx context.Context, data []byte) error) error
}
