# Skupina bodů, kterými budeme postupovat při implementaci: učení dané věci, vyzkoušení a až pak produkční kód
# 1. Basic infrastructure
# 2. NATS Jetstream: infra + hello-world (DONE) (viz. /backend/nats-test/main.go)
# 3. Postgresql
# 4. gRPC
# 5. Observability (Prometheus, OTel, Grafana, Loki, Tempo)
# 6. CI/CD + GitOps: ArgoCD + Tekton
# 7. Basics of K3s + K3d + Terraform - test "hello world" deployment Nginx, namespaces, service (DONE)

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

// po dopsání tohoto terraform init a pak terraform apply