package main

import (
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go"
)

func main() {
	// máme běžící pody s NATS Jetstreamem, dáme si pro testovací účely port forward:
	// kubectl port-forward -n infrastructure svc/nats-server 4222:4222
	// tímto se připojíme k NATS serveru, který běží v k3s clusteru, přes localhost:4222

	// máme port-forwarding z localhost:4222 do nats-serveru, který běží v k3s clusteru
	nc, err := nats.Connect("nats://127.0.0.1:4222")
	if err != nil {
		log.Fatal(err)
	}
	// potřebujeme zajistit, že se připojení zavře na konci funkce - defer
	defer nc.Close()

	// nats je sám o sobě jen "pošťák" (pošle a zapomene)
	// my ale chceme JetStream - ten zprávy ukládá a umožňuje nám je později znovu načíst
	// přes objekt jetStreamContext se pak dělají všechny pokročilé operace - jako vytváření streamů, publikování zpráv, atd.
	jetStreamContext, err := nc.JetStream()
	if err != nil {
		log.Fatal(err)
	}

	// V JetStreamu musíme nejdříve vytvořit "stream" - to je jako schránka, do které budeme posílat zprávy
	streamInfo, err := jetStreamContext.AddStream(&nats.StreamConfig{Name: "ORDERS", Subjects: []string{"orders.*"}})
	// Name je jméno streamu
	// Subjects jsou vzory, které určují, jaké zprávy do streamu patří - v našem případě všechny zprávy, které začínají "orders.":
	// takže: orders.new, orders.cancelled, orders.completed, atd.
	if err != nil {
		log.Fatal(err)
	}

	// musíme ještě využít streamInfo, aby se stream skutečně vytvořil
	if streamInfo == nil {
		log.Fatal("Failed to create stream")
	}

	// teď už jen stačí poslat data
	_, err = jetStreamContext.Publish("orders.new", []byte("Ahoj z Go!"))
	// proč []byte? NATS nezajímá co posíláme (text, JSON, obrázek...). Chce prostě jen pole bajtů.
	if err != nil {
		log.Fatal(err)
	}

	// úspěšně odesíláme zprávu do streamu "ORDERS" s předmětem "orders.new" a obsahem "Ahoj z Go!"
	// zkusíme si ji ještě zkonzumovat

	// použijeme synchronní odběr - to znamená, že budeme čekat na zprávu, kterou nám JetStream pošle

	// sub je objekt subscription, který nám umožní přijímat zprávy z JetStreamu
	sub, err := jetStreamContext.SubscribeSync("orders.*")
	if err != nil {
		log.Fatal(err)
	}

	// teď když máme sub, řekneme mu ať nám dá zprávu, která je v pořadí
	// jako parametr se předává časový limit, jak dlouho má program čekat než to vzdá
	msg, err := sub.NextMsg(time.Second * 5) // 5 sekund
	if err != nil {
		log.Fatal(err)
	}

	// msg je už ta zpráva, kterou jsme poslali - můžeme si ji vytisknout, ale předtím ještě přečtení a potvrzení zprávy (ack)
	// zprávu jsme posílali jako pole bytů ([]byte), NATS nám ji vrátí zase jako pole bytů v msg.Data
	// abychom to uměli přečíst v konzoli, musíme ty bajty převést zpět na text:
	fmt.Printf("%s", string(msg.Data))

	// nakonec musíme NATS serveru říct, že jsme zprávu zpracovali - na samotné zprávě zavoláme metodu .Ack()
	err = msg.Ack()
	if err != nil {
		log.Fatal(err)
	}
}
