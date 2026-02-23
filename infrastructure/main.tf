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

