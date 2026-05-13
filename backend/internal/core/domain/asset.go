package domain

import "time"

// ---------------------------------------------------------------------------------------------------------
//                                          DOMAIN LAYER
// ---------------------------------------------------------------------------------------------------------
// Proč domain layer existuje a proč začínáme psát kód právě zde:
// Dodržujeme practice zvaný "Clean Architecture", který ve své knize popsal Robert C. Martin
// Clean Architecture má základní myšlenku: závislosti (dependencies) tečou dovnitř.
// Domain je v tomto "středový kruh" - nezávisí na ničem (je nejvíce vnořený)
// tzn., že nezná NATS, nezná PostgreSQL, nezná HTTP. Je to čistá Business Logika.
// Proto do /backend/internal/core/domain souborů nebudeme importovat žádné externí knihovny.
// Jedinou povolenou vyjímkou je Go stdlib (balíček time), protože čas je business koncept, ne technologie.
// ---------------------------------------------------------------------------------------------------------

// AssetStatus je typ reprezentující stav firemního aktiva (lodního kontejneru)
// V Go není enum jako v Javě, ale můžeme si vytvořit vlastní pojmenovaný typ nad string.
// AssetStatus a string jsou tedy dvě různé věci
type AssetStatus string

// Byznys logika životního cyklu lodního kontejneru.
// Kontejner se narodí jako new, zákazník ho aktivuje -> active,
// najede hodně km nebo nastane problém -> maintenance,
// fyzicky dosluhuje -> decomissioned.
const (
	StatusNew            AssetStatus = "new"            // Stav: nový
	StatusActive         AssetStatus = "active"         // Stav: aktivní
	StatusMaintenance    AssetStatus = "maintenance"    // Stav: v údržbě
	StatusDecommissioned AssetStatus = "decommissioned" // Stav: vyřezený
)

// Asset reprezentuje jedno firemní aktivum (přepravní lodní kontejner)
// Proč MaxTemperature, MinTemperature a MaxHumidity zde: nejsou to konfigurační hodnoty služby,
// je to business pravidlo konkrétního kontejneru (aktiva).
// Kontejner s vakcínami má max 8°C a min 2°C (běžné chlazené vakcíny), kontejner s elektronikou zase jiné.
// MaxHumidity: vysoká vlhkost poškozuje elektroniku i vakcíny - každé aktivum má vlastní limit (%).
// Tato pravidla patří do doménové vrstvy, ne do Processor Service. Processor Service je jen čte a porovnává.
type Asset struct {
	ID             string // Zde string typ, protože budeme používat UUID, to generujeme v aplikaci, ne v Databázi
	Name           string
	MaxTemperature float64
	MinTemperature float64
	MaxHumidity    float64 // business pravidlo: maximální povolená relativní vlhkost (%)
	Status         AssetStatus
	CreatedAt      time.Time
}
