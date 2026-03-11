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
  // Tímto říkáme: "Všechny příkazy, které se týkají Kubernetes, posílej na tuhle adresu."
  // Je to ta samá adresa, kterou jsme definovali v bloku k3d_cluster jako host_port v rámci kube_api
  host = "https://127.0.0.1:6445"
  // Kubernetes nás dovnitř nepustí jen tak - chce vidět certifikáty - budeme dělat PROPOJOVÁNÍ ZDROJŮ
  // Uvnitř tohoto bloku budeme dále potřebovat tyto tři položky.
  // V k3d clusteru jsou tyto údaje uložené ve vnořené struktuře.
  // V Terraformu se k nim dostaneme přes tuto notaci: TYP_ZDROJE.JMÉNO_ZDROJE.ATRIBUT.
  // Jsou ale uložené v zakódovaném formátu Base64, aby je provider mohl přečíst musíme je rozbalti pomocí base64decode()
  client_certificate = k3d_cluster.tracking_system_k3s_cluster.credentials[0].client_certificate
  client_key = k3d_cluster.tracking_system_k3s_cluster.credentials[0].client_key
  cluster_ca_certificate = k3d_cluster.tracking_system_k3s_cluster.credentials[0].cluster_ca_certificate
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