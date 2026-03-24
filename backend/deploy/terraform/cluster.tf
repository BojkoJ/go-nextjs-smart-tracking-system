// Po spuštění příkazu init máme nastavený Terraform, máme stažený ovladač.
// Teď musíme napsat to nejdůležitější – co chceme vytvořit? - k3s cluster!
// v Terraformu se k tomu používá blok "resource":
// resource "TYP_ZDROJE" "MOJE_JMÉNO_ZDROJE" {.....}
// TYP_ZDROJE: To definuje ten ovladač (provider). Pro k3d je to vždy k3d_cluster.
// MOJE_JMÉNO_ZDROJE: náš custom název (třeba CLUSTER_PRO_SEM_PROJEKT atd.)
resource "k3d_cluster" "tracking_system_k3s_cluster" {
  // V rámci tohoto bloku teď budeme definovat, jak má ten cluster vypadat "fyzicky".
  name = var.cluster_name // To je jméno, které uvidíme když dáme příkaz "k3d cluster ls".
  servers = var.cluster_servers_count // Počet "mozků" (control plane).
  agents = var.cluster_agents_count // Počet dělníků (worker nodes).
  image   = var.cluster_image_name

  // Toto  jsou "zadní vrátka" pro administrátory.
  kube_api {
    // Říkáme, že mozek Kubernetes (Control Plane) má poslouchat na počítači (localhostu) na portu 6445.
    host_ip   = "127.0.0.1"
    host_port = var.kubeapi_host_port
  }

  port { // Důležité pro frontend a backend. Musíme říct: "Vezmi port 8080 na Ubuntu a propoj ho s portem 80 uvnitř clusteru".
    host_port = var.cluster_host_port
    container_port = var.cluster_container_port
    // Tímto řekneme k3d, že tento port má otevřít na té vstupní bráně clusteru.
    node_filters = ["loadbalancer"]
  }
}