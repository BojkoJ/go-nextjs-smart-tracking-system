terraform {
  required_version = ">= 1.0.0" // požadovaná verze samotného Terraformu, náš kód vyžaduje verzi ALESPOŇ 1.0.0
  // Říkáme tím: "Tento kód je napsaný pro moderní Terraform.
  // Pokud by to někdo zkusil spustit na prastaré verzi 0.12, rovnou to zastav, ať se nic nerozbije."


  // Terraform v tuto chvíli neví, že chceme pomocí něj ovládat k3d.
  // Musíme mu to říct uvnitř dalšího vnořeného bloku: Seznam požadovaných providerů/ovladačů
  // k3d ovladač abychom postavili kontejnery simulující server (2 agenty, 1 control plane)
  // kubernetes ovladač abychom do toho simulovaného serveru poslali instrukce 
  required_providers {
    // Každému ovladači dáme custom jméno a otevřeme pro něj další sub-blok
    k3d = {
      source = "moio/k3d" // Zdroj - kde na internetu (terraform registry) ovladač leží
      version = "~> 0.0.12" // Verze - Jakou verzi tohoto ovladače chceme, zkusíme třeba 0.0.7
    }

    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "2.38.0"
    }

    // Chceme si pomocí Terraformu do namespace "infrastricture" nainstalovat NATS (kvůli Jetstream)
    // Na to použijeme Helm ovladač (provider), který nám umožní říct: "Nainstaluj mi tento Helm chart do tohoto namespace"
    // Helm je vlastně něco jako App Store pro Kubernetes - dají se pomocí něho instalovat věci
    helm = {
      source  = "hashicorp/helm"
      version = "3.0.2"
    }
  }
}

// Tímto jsme pro Terraform napsali seznam věcí co chcem aby stáhl (Terraformovský ovladač pro k3d").
// Teď potřebujeme aby si je stáhl z internetu. Příkaz: "terraform init" (ve složce ./infrastructure)
// Tento příkaz stáhne ten ovladač, uloží si ho do skryté složky ".terraform" (jeho binárku)
// a také si vytvoří lockfile - .terraform.lock.hcl

// Po spuštění příkazu máme nastavený Terraform, máme stažený ovladač.
// Teď musíme napsat to nejdůležitější – co chceme vytvořit? - k3s cluster!
// v Terraformu se k tomu používá blok "resource":
// resource "TYP_ZDROJE" "MOJE_JMÉNO_ZDROJE" {.....}
// TYP_ZDROJE: To definuje ten ovladač (provider). Pro k3d je to vždy k3d_cluster.
// MOJE_JMÉNO_ZDROJE: náš custom název (třeba CLUSTER_PRO_SEM_PROJEKT atd.)
resource "k3d_cluster" "tracking_system_k3s_cluster" {
  // V rámci tohoto bloku teď budeme definovat, jak má ten cluster vypadat "fyzicky".
  name = "tracking-system-k3s-cluster" // To je jméno, které uvidíme když dáme příkaz "k3d cluster ls". Nesmí používat podtržítka - kubernetes má radši pomlčky.
  servers = 1 // Počet "mozků" (control plane). Pro naše účely stačí jeden.
  agents = 2 // Počet dělníků (worker nodes). Pro začátek 2, ať si můžeme zkusit odolnost.
  image   = "rancher/k3s:v1.31.5-k3s1"

  // Toto  jsou "zadní vrátka" pro administrátory.
  // Říkáme, že mozek Kubernetes (Control Plane) má poslouchat na počítači (localhostu) na portu 6445.
  kube_api {
    host_ip   = "127.0.0.1"
    host_port = 6445
  }

  port { // Důležité pro frontend a backend. Musíme říct: "Vezmi port 8080 na Ubuntu a propoj ho s portem 80 uvnitř clusteru".
    host_port = 8080
    container_port = 80
    node_filters = ["loadbalancer"] // Tímto řekneme k3d, že tento port má otevřít na té vstupní bráně clusteru.
  }
}


// Blok provider nic nevytváří, ale nastavuje
// tento blok bude dělat nastavení našeho kubernetes (hashicorp/kubernetes nainstalované v terraform/required_providers)
provider "kubernetes" {
  // path.module zajistí, že Terraform hledá přesně ve složce, kde právě jsme
  // tenhle .yaml se vytvoří příkazem:
  // k3d kubeconfig get tracking-system-k3s-cluster > k3d-config.yaml
  config_path = "${path.module}/k3d-config.yaml"
}

// Záměrně pod provider kubernetes si dáme provider pro helm:
provider "helm" {
  // musíme helmu říct, kde najde můj cluster - použijeme k tomu vnořený blok kubernetes
  kubernetes = {
    config_path = "${path.module}/k3d-config.yaml"
  }
}

// nadefinujeme si namespace infrastructure, ve kterém poběží věci které "slouží" (redis, postrgesql, message broker atd)
resource "kubernetes_namespace" "infra" {
  metadata {
    // toto uvidíme když dáme kubectl get ns
    name = "infrastructure"
  }
}

resource "kubernetes_namespace" "apps" {
  metadata {
    name = "tracking-system"
  }
}

// definice NATS přes Helm:
// tady budeme definovat objekt, který Terraformu řekne: "Vezmi tenhle balíček z Internetu a nainstaluj ho do namespace infra"
resource "helm_release" "nats" {
  // jméno, které uvidíme v Helmu
  name = "nats-server"

  // Repozitář, kde chceme lokalizovat příslušný Helm chart
  repository = "https://nats-io.github.io/k8s/helm/charts/"

  // jméno toho příslušného Helm chartu, který si chceme vytáhnout z repozitáře
  chart = "nats"

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


// tímto řekneme terraformu: "Použij ten ovladač kubernetes a vytvoř v něm objekt typu deployment"
// "hello_world" je vnitřní název pro Terraform. Pokud bychom později na tento deployment chtěli odkazovat, použijeme toto jméno.
resource "kubernetes_deployment" "hello_world" {

  // Tímto explicitně říkáme: "Nesahej na Kubernetes objekty, dokud není zdroj k3d_cluster v pořádku a plně načtený."
  // Terraform díky tomu nejdříve načte ty správné certifikáty do providera a až pak se pokusí o připojení.
  depends_on = [k3d_cluster.tracking_system_k3s_cluster]

  metadata {
    // toto je název přímo uvnitř kubernetes. když později napíšeme v terminálu kubectl get deployments, uvidíme tam toto jméno
    name = "hello-world-deployment"
    namespace = kubernetes_namespace.apps.metadata[0].name
  }

  spec {
    // v tomto je síla kubernetes, říkáme: "chci aby mi na těch mých dvou agentech běžely vždy 2 kopie této aplikace".
    // Pokud jedna spadne, kubernetes ji sám restartuje"
    replicas = 2

    // Toto je jako "lepidlo". Deployment musí vědět, které kontejnery pod něj patří.
    // říkáme mu: "Sleduj a ovládej všechny kontejnery, které mají na sobě label app="hello-world"
    selector {
      match_labels = {
        app = "hello-world"
      }
    }

    // v spec, pod selector se píše template - to je návod, podle kterého kubernetes vyrobí jednotlivé kopie (pods) mojí aplikace
    template {
      metadata {
        // zde se to přesně musí shodovat s tím co je v v selector -> match_labels
        // Protože tím říkáme: "Tento deployment vyrábí kontejnery s tímto labelem, aby je mohl později poznat a ovládat"
        labels = {
          app = "hello-world"
        }
      }
      // teď musíme říct, co v tom Podu skutečně poběží. To se píše také do spec bloku, ale spec bloku uvnitř bloku template
      spec {
        // zde definujeme seznam kontejnerů. I když budeme mít jen jeden.
        // můžeme toto opakovat vícekrát pokud bychom chtěli více kontejnerů
        container {
          name = "hello-container"
          // Jaký obraz se má stáhnout
          // To je ten software, který distriibuujeme (distrbuovaný systém).
          // Tento konkrétní obraz od Nginx nám v prohlížeči ukáže, který z našich dvou agentů právě běží
          image = "nginxdemos/hello"
          port {
            // říkáme kubernetes: "Tato aplikace uvnitř kontejneru poslouchá na portu 80"
            container_port = 80
          }
        }
      }
    }
  }
}

// Nyní potřebujeme service, bez něj by kontejnery sice běžely, ale byly by uzavřené uvnitř clusteru.
// Nikdy zvenčí by k nim nemohl přistupovat či s nimi komunikovat.
resource "kubernetes_service" "hello_world_service" {
  depends_on = [k3d_cluster.tracking_system_k3s_cluster]
  metadata {
    // stejně jako u deploymentu, i Service musí mít jméno
    name = "hello-world-service"
    namespace = kubernetes_namespace.apps.metadata[0].name
  }

  // zde definujeme jak má service fungovat
  spec {
    selector = {
        app = "hello-world"
    }

    // tady definujeme mapování portů uvnitř clusteru
    port {
      // na tomto portu bude služba dostupná v rámci sítě Kubernetes
      port = 80
      // toto je port, na kterém poslouchá náš kontejner, jak jsme definovali v deploymentu
      target_port = 80
    }

    // v k3d cluster jsme definovali host_port = 8080 ... node_filters = ["loadbalancer"]
    // takže když teď vytvoříme službu typu LoadBalancer, k3d pochopí a automaticky ji připojí na ten připravený port 8080
    type = "LoadBalancer"
  }
}

// po dopsání tohoto terraform init a pak terraform apply