
// Nyní potřebujeme service, bez něj by kontejnery sice běžely, ale byly by uzavřené uvnitř clusteru.
// Nikdy zvenčí by k nim nemohl přistupovat či s nimi komunikovat.
resource "kubernetes_service" "hello_world_service" {
  depends_on = [k3d_cluster.tracking_system_k3s_cluster]
  metadata {
    // stejně jako u deploymentu, i Service musí mít jméno
    name = var.k3s_service_name
    namespace = kubernetes_namespace.apps.metadata[0].name
  }

  // zde definujeme jak má service fungovat
  spec {
    selector = {
      app = var.k3s_deployment_container_label
    }

    // tady definujeme mapování portů uvnitř clusteru
    port {
      // na tomto portu bude služba dostupná v rámci sítě Kubernetes
      port = var.k3s_deployment_container_port
      // toto je port, na kterém poslouchá náš kontejner, jak jsme definovali v deploymentu
      target_port = var.k3s_deployment_container_port
    }

    // v k3d cluster jsme definovali host_port = 8080 ... node_filters = ["loadbalancer"]
    // takže když teď vytvoříme službu typu LoadBalancer, k3d pochopí a automaticky ji připojí na ten připravený port 8080
    type = "LoadBalancer"
  }
}

