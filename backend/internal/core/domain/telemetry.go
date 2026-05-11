package domain

import "time"

// ---------------------------------------------------------------------------------------------------------
//                                          DOMAIN LAYER
// ---------------------------------------------------------------------------------------------------------

// TelemetryData reprezentuje jedno čtení senzorů aktiva
// Proč TraceID zde v doménové vrstvě: TraceID jsou metadata, která putují se zprávou přes celý systém.
// Pokud bychom ho neuložili do TelemetryData, ztratíme kontext a Grafana Tempo by ztratilo kontext.
// Tím, že ho dáme do doménové vrstvy, říkáme "každý kus dat ze světa má svou stopu"
// Je to architektonické rozhodnutí - observability je součást kontraktu, ne afterhtought.
type TelemetryData struct {
	AssetID     string
	Latitude    float64
	Longitude   float64
	Temperature float64
	Humidity    float64
	IsLocked    bool
	Timestamp   time.Time
	TraceID     string
}
