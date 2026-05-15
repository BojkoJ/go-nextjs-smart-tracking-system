package main

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"math/rand"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/BojkoJ/go-nextjs-smart-tracking-system/backend/internal/common"
	pb "github.com/BojkoJ/go-nextjs-smart-tracking-system/backend/proto"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ---------------------------------------------------------------------------------------------------------
//                                         SERVICES - SIMULATOR
// ---------------------------------------------------------------------------------------------------------
// Proč simulator v cmd/:
// Simulator je samostatná spustitelná binárka - má vlastní main(). Není sdílenou knihovnou, je to service.
// Veškerá logika simulace žije zde - simulator nesdílí žádný internal/ kód s ostatními službami,
// importuje pouze proto/ (gRPC client) a common/ (logger).
// ---------------------------------------------------------------------------------------------------------
// SIMULAČNÍ SCÉNÁŘ: Přeprava v létě (červen–srpen)
//
// Zákazník:  IQM Quantum Computers, Espoo, Finsko (výrobce supravodivých kvantových počítačů)
// Náklad:    Niobium targets (Ulvac / Canon Anelva, Japonsko) + PCB substráty (Jižní Korea / Taiwan)
//            Supravodivé vrstvy jsou extrémně citlivé na vlhkost a teplotní výkyvy.
// Trasa:     Busan (Jižní Korea) -> Rotterdam (Nizozemí)
//            Délka trasy: ~20 000 km | Doba plavby: ~28 dní | rychlost: cca ~7m/s (pro naše účely zrychlíme 20x - 140m/s)
//
// LIMITY NÁKLADU (business pravidla uložená na každém Asset záznamu v DB):
//   MaxTemperature: 28.0 °C  - nad touto hodnotou hrozí oxidace niobiového povrchu
//   MinTemperature: 12.0 °C  - pod touto hodnotou kondenzuje vlhkost na PCB substrátech
//   MaxHumidity:    45.0 % RH - supravodivé vrstvy degradují při vyšší relativní vlhkosti (RH - Relative Humidity)
//
// POČÁTEČNÍ HODNOTY GOROUTIN:
//   Teplota:  20.0 °C  (střed bezpečného rozsahu, klimatizovaný kontejner)
//   Vlhkost:  35.0 % RH (bezpečná hodnota, s rezervou pod limitem)
//   IsLocked: true
//   GPS:      50 kontejnerů všechny na stejné lodi se stejným počátečním waypointem
//
// SIMULACE FLUKTUACE TEPLOTY (každý telemetrický záznam = každá sekunda):
//   Základ:    sinusoida podle hodiny dne: base = 20 + 6 * sin(hourAngle)  -> rozsah 14–26 °C
//   Šum:       +-0.8 °C náhodná fluktuace (senzorový šum + proudění vzduchu v kontejneru)
//   Výsledek:  teplota fluktuuje realisticky, alarm nastane pouze při simulované anomálii
//              (výpadek klimatizace = postupný drift nad MaxTemperature)
// ---------------------------------------------------------------------------------------------------------

type Waypoint struct {
	Latitude  float64
	Longitude float64
}

type ContainerState struct {
	ID                string
	IsLocked          bool
	TemperatureOffset float64
	HumidityOffset    float64
}

// GPS WAYPOINTS TRASY:
var routeWaypoints = []Waypoint{
	{Latitude: 35.10, Longitude: 129.04}, // Busan - přístav
	{Latitude: 15.00, Longitude: 115.00}, // Jihočínské moře
	{Latitude: 2.55, Longitude: 101.15},  // Malacký průliv
	{Latitude: 5.33, Longitude: 80.54},   // Indický oceán
	{Latitude: 12.80, Longitude: 51.30},  // Adenský záliv
	{Latitude: 29.97, Longitude: 32.55},  // Suezský průplav
	{Latitude: 36.40, Longitude: 14.50},  // Středozemní moře
	{Latitude: 35.95, Longitude: -5.45},  // Gibraltar
	{Latitude: 49.79, Longitude: -2.89},  // Lamanšský průliv
	{Latitude: 51.98, Longitude: 4.05},   // Rotterdam - přístav
}

var (
	segmentLengths   []float64 // délky segmentů mezi waypointy
	totalRouteLength float64   // celková délka cesty z Busanu do Rotterdamu
)

// haversine používá Haversinův vzorec pro pohyb (posouvání souřadnic) a pro zjištění celkové délky cesty v init,
// protože tento vzorec počítá narozdíl od Euklidovské vzádlenosti  skutečnou vzdálenost po povrchu koule,
// pohyb na mapě je pak fyzikálně správný.
func haversine(a, b Waypoint) float64 {
	const earthRadiusKm = 6371.0
	latitude1 := a.Latitude * math.Pi / 180.0
	latitude2 := b.Latitude * math.Pi / 180.0

	dLatitude := (b.Latitude - a.Latitude) * math.Pi / 180.0
	dLongitude := (b.Longitude - a.Longitude) * math.Pi / 180.0

	// Haversinův vzorec:
	x := math.Sin(dLatitude/2)*math.Sin(dLatitude/2) +
		math.Cos(latitude1)*math.Cos(latitude2)*
			math.Sin(dLongitude/2)*math.Sin(dLongitude/2)

	return earthRadiusKm * 2.0 * math.Atan2(math.Sqrt(x), math.Sqrt(1.0-x))
}

// init se spustí jednou při startu programu, před main()
// délky segmentů jsou konstantní. Počítat je 50x (v každé goroutině) každou vteřinu by bylo zbytečné.
// Package-level proměnné jsou sdílené read-only po inicializaci -> není potřeba mutex.
func init() {
	for i := 0; i < len(routeWaypoints)-1; i++ {
		length := haversine(routeWaypoints[i], routeWaypoints[i+1])
		segmentLengths = append(segmentLengths, length)
		totalRouteLength += length
	}
}

func computeShipPosition(startTime time.Time) (latitude, longitude float64) {
	// i přesto, že používáme Haversinův vzorec zde budeme používat lineární interpolaci
	// Haversinův vzorec řeší délku segmentů (správná rychlost). Interpolace pozice lineárně v stupních je přibližná,
	// ale pro segmenty délky 1000-3000km je odchylka <0.5%, což je na mapě nepostřehnutelné.

	// používáme math.Min(..., 1.0) clamp, protože by jinak po dosažení Rotterdamu (progress == 1.0) loď chtěla jet dál
	// snažila by se jít za poslední waypoint, což by způsobila crash

	// konstanty:
	const (
		shipSpeedKnots    = 15.0  // rychlost simulované nákladní lodi v uzlech, 15 uzlů je cca ~7.71 metrů za vteřinu
		amplifier         = 20.0  // zrychlení - chceme to 20-krát zrychlit, aby loď neplula 4 týdny jako v realitě.
		kmPerNauticalMile = 1.852 // jedna námořní míle (nautical mile) je 1.852 kilometrů
	)

	// výpočet rychlosti v kilometrech za vteřinu:
	speedKmPerSec := shipSpeedKnots * amplifier * kmPerNauticalMile / 3600.0
	// výpočet počtu vteřin plavby
	totalSeconds := totalRouteLength / speedKmPerSec

	// počet uplynutých vteřin od předaného startTime
	elapsed := time.Since(startTime).Seconds()
	// počet uplynutých vteřin z plavby, clamped na 1.0 (100%), aby loď nechtěla jet dál za Rotterdam
	progress := math.Min(elapsed/totalSeconds, 1.0)

	// target je vzdálenost, kterou loď urazila aktuálně od startu, v kilometrech
	target := progress * totalRouteLength
	// accumulated je vzdálenost, kterou loď urazila v průběhu iterace přes segmenty
	accumulated := 0.0
	// loopujeme přes délky jednotlivých úseků mezi waypointy, abychom zjistili, na kterém segmentu se loď nachází
	for i, length := range segmentLengths {
		// pokud je accumulated + length větší nebo rovno target, znamená to, že loď se nachází na tomto segmentu
		if accumulated+length >= target {
			// t je poměr, jak daleko na tomto segmentu se loď nachází (0.0 = začátek segmentu, 1.0 = konec segmentu)
			t := (target - accumulated) / length
			// lineární interpolace mezi waypointy a, b:
			a, b := routeWaypoints[i], routeWaypoints[i+1]
			// vracíme interpolované souřadnice:
			// jedna interpolované souřadnice se počítá jako: start + t * (end - start)
			// start je v tomto případě souřadnice prvního waypointu (a), end je souřadnice druhého waypointu (b)
			// protože loď je ve waypointu a, a waypoint b je cíl daného úseku (či segmentu),
			// t nám říká, jak daleko na tomto úseku se nachází (0.0 = na a přesně, 1.0 = na b přesně, 0.5 = přesně v polovině mezi a a b)
			return a.Latitude + t*(b.Latitude-a.Latitude),
				a.Longitude + t*(b.Longitude-a.Longitude)
		}
		// k uražené vzdálenosti přičteme délku tohoto segmentu a pokračujeme na další segment
		accumulated += length
	}
	// chování tohoto loopu: prochází úseky (segmenty) mezi waypointy, pokud to není segment, na kterém se loď nachází,
	// loop přičítá délku segmentu k accumulated a pokračuje dál. Až najde segment, na kterém se loď nachází počítá interpolaci a vrátí souřadnice.
	// Proč interpolace, i když máme definovanou funkci haversine, která počítá vzdálenost po povrchu koule:
	// Protože haversine nám dává správnou délku segmentů (správná rychlost), ale pro pohyb na mapě nám stačí lineární interpolace v souřadnicích.

	// poslední waypoint je cílový bod Rotterdamu
	last := routeWaypoints[len(routeWaypoints)-1]
	// vracíme z této funkce souřadnice posledního waypointu, pokud by se náhodou stalo, že progress == 1.0 a loď by chtěla jet dál za Rotterdam
	// což by se ale technicky díky clampu nemělo stát, ale pro jistotu tu máme tento fallback, aby nedošlo k chybě a crashi programu
	return last.Latitude, last.Longitude
}

func computeTemperature(offset float64) float64 {
	// použijeme sinusoidu, ne random walk,
	// protože random walk by dával šum bez trendu,
	// skutečná teplota v klimatizovaných lodních kontejnerech kopířuje denní cyklus: chladnější v noci, teplejší přes den.
	// Sinusoida přesně toto chování modeluje.
	now := time.Now()

	// bez minutové složky v hour by teplota skákala každou hodinu (.Hour() totiž vrací integer), s minutami je průběh hladký
	// hour obsahuje aktuální hodinu a minuty jako desetinnou část (např. 16:30 by bylo 16.5)
	hour := float64(now.Hour()) + float64(now.Minute())/60.0

	// co je hourAngle: je to úhel v radiánech, který odpovídá aktuální hodině dne, přičemž 0rad = půlnoc, πrad = poledne, 2πrad = opět půlnoc
	hourAngle := 2.0 * math.Pi * hour / 24.0

	// co je base: base je základní teplota v danou hodinu, která se mění podle časové sinusoidy.
	// Proč 20.0 a 6.0: 20.0 je střední teplota (průměr mezi 14 a 26), 6.0 rozdíl mezi středem a minimem/maximem (26-20=6, 20-14=6)
	// tímto dosáhneme toho, že teplota bude kolísat mezi 14 °C (v noci) a 26 °C (přes den), což je realistické pro klimatizovaný kontejner,
	// vzhledem k trase (Busan -> Rotterdam, převážně mírné pásmo) a ročnímu období (léto) to dává smysl.
	base := 20.0 + 6.0*math.Sin(hourAngle)

	// šum přidává náhodnou fluktuaci kolem base teploty, aby simulace nebyla příliš "čistá" a aby se občas mohly vyskytnout náhodné odchylky,
	// které jsou běžné u reálných senzorů.
	noise := (rand.Float64()*2.0 - 1.0) * 0.8 // náhodný šum v rozsahu -0.8 až +0.8 °C

	// co je offset: offset simuluje různé umístění kontejneru v lodi (jiná zóna, jiné proudění vzduchu).
	return base + noise + offset
}

func computeHumidity(startTime time.Time, offset float64) float64 {
	// proč jiná perioda (12h) než u teploty (24h):
	// vlhkost reaguje na jiné faktory: kondenzace přes noc, větrání při výkladce palub, odlišná klimatizační fáze.
	// 12h perioda modeluje tato dvoufázová okna.

	// počet uplynulých hodin od startu, včetně desetinné části pro minuty
	elapsed := time.Since(startTime).Hours()
	// toto je základní vlhkost, která se mění podle 12hodinové sinusoidy
	// zase: 35.0 je střední vlhkost (průměr mezi 25 a 45),
	// 5.0 je rozdíl mezi středem a minimem/maximem (45-35=10, 35-25=10, ale protože sinusoidu posouváme o 12h, tak 10.0/2=5.0)
	base := 35.0 + 5.0*math.Sin(2.0*math.Pi*elapsed/12.0)
	// šum pro vlhkost je větší než pro teplotu, protože vlhkost je obecně více variabilní a citlivá na okolní podmínky (větrání, kondenzace)
	// tento výpočet: generuje náhodné číslo mezi -1.0 a +1.0 (rand.Float64()*2.0 - 1.0), které pak škálujeme na rozsah -1.5 až +1.5 % RH
	noise := (rand.Float64()*2.0 - 1.0) * 1.5

	// výsledek je základní vlhkost plus náhodný šum a případný offset pro simulaci anomálie (např. zvýšená vlhkost kvůli kondenzaci)
	res := base + noise + offset

	// pro případ, že vlhkost by byla menší než 20% nebo větší než 60%:
	// použijeme clamp, aby výsledná vlhkost zůstala v realistickém rozsahu pro námořní kontejnery.
	return math.Max(20.0, math.Min(60.0, res))
}

func runContainer(id int, startTime time.Time, client pb.TelemetryServiceClient, wg *sync.WaitGroup, ctx context.Context, logger *slog.Logger) {
	defer wg.Done() // oznámení WaitGroupě, že goroutina skončila

	state := ContainerState{
		ID:                fmt.Sprintf("asset-%d", id+1),
		IsLocked:          true,
		TemperatureOffset: rand.Float64()*4.0 - 2.0, // random offset mezi -2.0 a +2.0
		HumidityOffset:    rand.Float64()*6.0 - 3.0, // random offset mezi -3.0 a 3.0
	}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		// select je non-blocking check na channel - Go idiom pro context cancellation ve smyčkách.
		// Garantuje reakci na shutdown bez race condition.
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// zjistíme pozici lodě pro tuto vteřinu
			latitude, longitude := computeShipPosition(startTime)

			// spočítáme novou teplotu pro tuto vteřinu
			temperature := computeTemperature(state.TemperatureOffset)

			// spočítáme novou vlhkost pro tuto vteřinu
			humidity := computeHumidity(startTime, state.HumidityOffset)

			// Nový root span pro každý telemetrický event — trace propojí cestu přes Ingest → NATS → Processor → DB
			spanCtx, span := otel.Tracer("simulator").Start(ctx, "send-telemetry",
				trace.WithSpanKind(trace.SpanKindProducer),
			)

			request := pb.TelemetryRequest{
				AssetId:     state.ID,
				Latitude:    latitude,
				Longitude:   longitude,
				Temperature: temperature,
				Humidity:    humidity,
				IsLocked:    state.IsLocked,
				TimestampNs: time.Now().UnixNano(),
				// TraceID z aktivního spanu — otelgrpc propaguje tento kontext do gRPC metadat
				TraceId: span.SpanContext().TraceID().String(),
			}

			// každý grpc call dostane vlastní deadline (5 sekund) - bez deadline by při pomalém ingestu goroutina blokovala indefinitely
			// callCancel voláme přímo po návratu SendTelemetry, ne defer - defer v smyčce se akumuluje a nevolá se do konce funkce
			callCtx, callCancel := context.WithTimeout(spanCtx, 5*time.Second)
			response, err := client.SendTelemetry(callCtx, &request)
			callCancel()

			if err != nil {
				logger.Warn("had an error sending telemetry", "error", err)
			} else if !response.Success {
				logger.Warn("response when sending telemetry was not success", "message", response.Message)
			}
			span.End()
		}
	}
}

func main() {
	// vytvoříme si pro tuto mikroslužbu logger a logneme rovnou, že mirkoslužba začíná
	logger := common.NewLogger("simulator")
	logger.Info("Simulator Microservice starting", "containers", 50)

	otelShutdown, err := common.InitTracerProvider(context.Background(), "simulator")
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

	// místo deprecated grpc.Dial použijeme grpc.NewClient()
	// grpc.NewClient má lazy connection - nekontroluje dostupnost serveru, jen nakonfiguruje klienta.
	// Skutečné připojení proběhne až při prvním SendTelemetry callu.
	connection, err := grpc.NewClient(
		config.IngestAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		// otelgrpc.NewClientHandler() injektuje W3C trace context do gRPC metadat každého volání
		// Ingest server handler extrahuje context a vytvoří child span — propojení trace přes hranici služeb
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	if err != nil {
		logger.Error("failed to create gRPC client for ingest microservice", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := connection.Close(); err != nil {
			logger.Error("failed to close gRPC connection", "error", err)
		}
	}() // tyto závorky na konci okamžitě zavolají tuto anonymní funkci při ukončení main - defer

	// vytvoříme nového telemetry clienta
	client := pb.NewTelemetryServiceClient(connection)
	startTime := time.Now()

	// tento řádek dělá:
	// 1) signal.NotifyContext vytvoří nový context, který se automaticky zruší (cancel) při obdržení signálu os.Interrupt (Ctrl+C) nebo syscall.SIGTERM
	// (což je takzvaný "graceful shutdown"
	// proč signal.NotifyContext místo manuálního channelu:
	// go 1.16+ idiom: vrátí context, který canceluje při SIGTERM/SIGINT.
	// k8s posílá SIGTERM při kubectl delete pod -> goroutiny to odchytí přes ctx.Done() a čistě ukončí
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// vytvoříme si WaitGroupu pro goroutiny
	var wg sync.WaitGroup
	// 50x loopneme a vytvoříme goroutiny
	for i := range 50 {
		// wg.Add(1) před vytvořením goroutiny kvůli zabránění race condition:
		// goroutina nesmí stihnout Add před tím, než main zavolá wg.Wait(),
		// Add musí být voláno synchronně před spuštěním goroutiny
		wg.Add(1)

		// spustíme goroutinu s funkcí runContainer
		go runContainer(i, startTime, client, &wg, ctx, logger)
	}

	// tento řádek: operátor <- zablokuje hlavní goroutinu, dokud nedojde k zrušení contextu (ctx.Done()), což se stane při obdržení SIGINT/SIGTERM.
	<-ctx.Done()
	logger.Info("shutdown signal received, waiting for goroutines to finish...")
	// pomocí WaitGroupy počkáme na dokončení všech goroutin
	wg.Wait()
	logger.Info("simulator microservice stopped")

}
