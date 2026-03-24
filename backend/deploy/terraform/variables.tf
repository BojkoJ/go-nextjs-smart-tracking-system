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

// -------- Pro deployment: -----------------
variable "k3s_deployment_name" {
  type = string
  description = "Název deploymentu uvnitř kubernetes"
  default = "hello-world-deployment"
}

variable "k3s_deployment_replicas_num" {
  type = number
  description = "Počet replik v hello_world deploymentu"
  default = 2
}

variable "k3s_deployment_container_label" {
  type = string
  description = "Label kontejnerů, které produkuje náš hello world Nginx deployment"
  default = "hello-world"
}

variable "k3s_deployment_container_name" {
  type = string
  description = "Jméno kontejnerů, které produkuje náš hello world Nginx deployment"
  default = "hello-container"
}

variable "k3s_deployment_container_image" {
  type = string
  description = "Image, který se pro kontejnery stáhne a nainstaluje"
  default = "nginxdemos/hello"
}

variable "k3s_deployment_container_port" {
  type = number
  description = "Port, na kterém poslouchá aplikace uvnitř kontejneru produkovaném deploymentem"
  default = 80
}

// -------- Pro service: -----------------
variable "k3s_service_name" {
  type = string
  description = "Název služby (serivce) univtř kubernetes"
  default = "hello-world-service"
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