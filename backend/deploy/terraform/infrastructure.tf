// definice NATS přes Helm:
// tady budeme definovat objekt, který Terraformu řekne: "Vezmi tenhle balíček z Internetu a nainstaluj ho do namespace infra"
resource "helm_release" "nats" {
  // jméno, které uvidíme v Helmu
  name = var.helm_release_nats_name

  // Repozitář, kde chceme lokalizovat příslušný Helm chart
  repository = var.helm_release_nats_repo

  // jméno toho příslušného Helm chartu, který si chceme vytáhnout z repozitáře
  chart = var.helm_release_nats_chart

  // tímhle terraformu řekneme: "instaluj to tam, kde jsi vytvořil namespace infrastructure"
  namespace = kubernetes_namespace.infra.metadata[0].name

  // tím zapneme "paměť" pro naše zprávy
  // blok "set" dělá to, že přepíše výchozí hodnotu v Helm chartu, která je jinak false

  set = [
    {
      // jméno, té hodnoty, kterou chceme přepsat
      name = "config.jetstream.enabled"
      // nová value
      value = "true"
      // Proč jsme tuhle hodnotu v našem helm chartu "nats" přepisovali:
      // V našem projektu potřebujeme, aby NATS měl zapnutou funkci Jetstream, která nám umožní trvalé ukládání zpráv.
    },
    // to stejné znovu, ale pro hodnotu "cluster.enabled":
    {
      name = "config.cluster.enabled"
      value = "false" // chceme jeden jediný NATS server, nepotřebujeme cluster NATS serverů - tím šetříme omezené systémové prostředky
    }
  ]
}
