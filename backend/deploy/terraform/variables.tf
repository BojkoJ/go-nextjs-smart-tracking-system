// Proměnné:

// -------- Pro cluster: -----------------
variable "cluster_name" {
  type = string
  description = "Jméno k3d clusteru"
  default = "tracking-system-k3s-cluster"   // Nesmí používat podtržítka - kubernetes má radši pomlčky.
}

variable "cluster_servers_count" {
  type = number
  description = "Počet \"mozků clusteru\" - control planes"
  default = 1 // Pro naše účely stačí jeden
}

variable "cluster_agents_count" {
  type = number
  description = "Počet \"dělníků\" - worker nodes"
  default = 2 // Pro začátek 2, ať si můžeme zkusit odolnost.
}

variable "cluster_image_name" {
  type = string
  description = "Jméno image, který budeme používat pro k3d nody"
  default = "rancher/k3s:v1.31.5-k3s1"
}

variable "cluster_host_port" {
  type = number
  description = "Host port z OS, který je propojený s portem clusteru"
  default = 8080
}

variable "cluster_container_port" {
  type = number
  description = "Port uvnitř clusteru, na kterém poslouchá loadbalancer"
  default = 80
}

// -------- Pro kubeapi od clusteru: -----------------
variable "kubeapi_host_port" {
  type = number
  description = "Port, na kterém bude poslochat control plane na lokálním PC (localhostu)"
  default = 6445
}


// -------- Pro namespaces: -----------------
variable "k3s_namespace_infra_name" {
  type = string
  description = "Název namespacu pro infrastrukturu uvnitř kubernetes"
  default = "infrastructure"
}

variable "k3s_namespace_apps_name" {
  type = string
  description = "Název namespacu pro mikroslužby/back-end uvnitř kubernetes"
  default = "tracking-system"
}

// -------- Pro infrastrukturu: -----------------
variable "helm_release_nats_name" {
  type = string
  description = "Jméno pro nats helm release, které uvidíme uvnitř Helmu"
  default = "nats-server"
}

variable "helm_release_nats_repo" {
  type = string
  description = "Adresa repozitáře, kde chceme lokalizovat daný Helm Chart"
  default = "https://nats-io.github.io/k8s/helm/charts/"
}

variable "helm_release_nats_chart" {
  type = string
  description = "Jméno chartu, který hledáme v daném repozitáři"
  default = "nats"
}

// -------- Pro PostgreSQL: -----------------
variable "helm_release_postgres_name" {
  type = string
  description = "Jméno PostgreSQL Helm Releasu"
  default = "postgresql"
}

variable "helm_release_postgres_repo" {
  type = string
  description = "Odkaz na repozitář, který obsahuje daný Helm Chart pro PostgreSQL"
  default = "oci://registry-1.docker.io/bitnamicharts"
}

variable "helm_release_postgres_chart" {
  type = string
  description = "Jméno PostgreSQL Helm Chartu pro tento Helm Release"
  default = "postgresql"
}

variable "postgres_db_name" {
  type = string
  description = "Jméno PostgreSQL databáze, kterou projekt používá"
  default = "tracking_db"
}

variable "postgres_user" {
  type = string
  description = "Jméno PostgreSQL usera, který bude databázi používat"
  default = "tracking_user"
}

// Heslo: nemá default, hodnotu pak bude brát z terraform.tfvars
variable "postgres_password" {
  type = string
  description = "Heslo k PostgreSQL databázi, které zde není skutečně uloženo"
  sensitive = true // Terraform toto pole skryje v logu, nikdy tuto hodnotu nebude vypisovat do terminálu.
}

// -------- Pro ArgoCD: -----------------
variable "helm_release_argocd_name" {
  type = string
  description = "Jméno ArgoCD Helm Releasu"
  default = "argocd"
}

variable "helm_release_argocd_repo" {
  type = string
  description = "Odkaz na repozitář, který obsahuje daný Helm Chart pro ArgoCD"
  default = "https://argoproj.github.io/argo-helm"
}

variable "helm_release_argocd_chart" {
  type = string
  description = "Jméno ArgoCD Helm Chartu pro tento Helm Release"
  default = "argo-cd"
}

variable "k3s_namespace_argocd_name" {
  type = string
  description = "Jméno pro ArgoCD, které bude znát k3s namespace zvaný \"infra\""
  default = "argocd"
}