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

// -------- Observability stack --------

// Prometheus + Grafana (kube-prometheus-stack = Prometheus Operator + Grafana + AlertManager + exporters)
resource "helm_release" "kube_prometheus_stack" {
  name       = "kube-prometheus-stack"
  repository = "https://prometheus-community.github.io/helm-charts"
  chart      = "kube-prometheus-stack"
  namespace  = kubernetes_namespace.monitoring.metadata[0].name
  timeout    = 600

  set = [
    {
      name  = "grafana.adminPassword"
      value = var.grafana_password
    },
    {
      name  = "grafana.service.type"
      value = "NodePort"
    },
    {
      name  = "prometheus.service.type"
      value = "NodePort"
    }
  ]
}

// Grafana Tempo — distribuovaný tracing backend, přijímá OTLP přes gRPC na portu 4317
resource "helm_release" "tempo" {
  name       = "tempo"
  repository = "https://grafana.github.io/helm-charts"
  chart      = "tempo"
  namespace  = kubernetes_namespace.monitoring.metadata[0].name

  set = [
    {
      name  = "tempo.receivers.otlp.protocols.grpc.endpoint"
      value = "0.0.0.0:4317"
    }
  ]
}

// Loki + Promtail — agregace logů ze všech podů (Promtail = DaemonSet scraping /var/log/pods)
resource "helm_release" "loki_stack" {
  name       = "loki"
  repository = "https://grafana.github.io/helm-charts"
  chart      = "loki-stack"
  namespace  = kubernetes_namespace.monitoring.metadata[0].name

  set = [
    {
      name  = "grafana.enabled"
      value = "false" // Grafana již běží v kube-prometheus-stack
    },
    {
      name  = "prometheus.enabled"
      value = "false" // Prometheus již běží v kube-prometheus-stack
    }
  ]
}

// definujeme si PostgreSQL přes Helm
// Jakmile Helm nasadíme, k8s service pro PostgreSQL bude dostupná pod DNS jménem: postgresql.infrastructure.svc.cluster.local:5432
// Formát je: <helm-release-name>.<namespace>.svc.cluster.local:<port>. Toto budeme používat jako connection string v Go aplikacích.
resource "helm_release" "postgresql" {
  // jméno, které uvidíme v Helmu
  name = var.helm_release_postgres_name
  // Repozitář, kde chceme lokalizovat příslušný Helm chart
  repository = var.helm_release_postgres_repo
  // jméno toho příslušného Helm chartu, který si chceme vytáhnout z repozitáře
  chart = var.helm_release_postgres_chart
  // tímhle terraformu řekneme: "instaluj to tam, kde jsi vytvořil namespace infrastructure"
  namespace = kubernetes_namespace.infra.metadata[0].name
  // Proč 600: Bitnami PostgreSQL image (~130 MB) trvá na Docker Hub 4-5 minut stáhnout při prvním pulu.
  // Default Terraform timeout je 300s → context deadline exceeded před Ready stavem podu.
  timeout   = 600

  // Bitname PostgreSQL chart potřebuje vědět credentials a nastavení.
  set = [
    {
      name = "auth.postgresPassword"
      value = var.postgres_password
    },
    {
      name  = "auth.username"
      value = var.postgres_user
    },
    {
      name = "auth.password"
      value = var.postgres_password
    },
    {
      name = "auth.database"
      value = var.postgres_db_name
    },
    // Proč toto: persistence = data přežijí restart podu díke PersistenceVolume.
    // V k3d lokálním clusteru je nastavení PersistenceVolume složité.
    // Pro development účely data v RAM stačí. V produkci bychom však to zapnuli.
    {
      name = "primary.persistence.enabled"
      value = "false"
    }
  ]
}