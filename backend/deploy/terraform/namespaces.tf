// nadefinujeme si namespace infrastructure, ve kterém poběží věci které "slouží" (redis, postrgesql, message broker atd)
resource "kubernetes_namespace" "infra" {
  metadata {
    // toto uvidíme když dáme kubectl get ns
    name = var.k3s_namespace_infra_name
  }
}

resource "kubernetes_namespace" "apps" {
  metadata {
    name = var.k3s_namespace_apps_name
  }
}
