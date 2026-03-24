
// tímto řekneme terraformu: "Použij ten ovladač kubernetes a vytvoř v něm objekt typu deployment"
// "hello_world" je vnitřní název pro Terraform. Pokud bychom později na tento deployment chtěli odkazovat, použijeme toto jméno.
resource "kubernetes_deployment" "hello_world" {
  // Tímto explicitně říkáme: "Nesahej na Kubernetes objekty, dokud není zdroj k3d_cluster v pořádku a plně načtený."
  // Terraform díky tomu nejdříve načte ty správné certifikáty do providera a až pak se pokusí o připojení.
  depends_on = [k3d_cluster.tracking_system_k3s_cluster]

  metadata {
    // toto je název přímo uvnitř kubernetes. když později napíšeme v terminálu kubectl get deployments, uvidíme tam toto jméno
    name = var.k3s_deployment_name
    namespace = kubernetes_namespace.apps.metadata[0].name
  }

  spec {
    // v tomto je síla kubernetes, říkáme: "chci aby mi na těch mých dvou agentech běžely vždy 2 kopie této aplikace".
    // Pokud jedna spadne, kubernetes ji sám restartuje"
    replicas = var.k3s_deployment_replicas_num

    // Toto je jako "lepidlo". Deployment musí vědět, které kontejnery pod něj patří.
    // říkáme mu: "Sleduj a ovládej všechny kontejnery, které mají na sobě label app="hello-world"
    selector {
      match_labels = {
        app = var.k3s_deployment_container_label
      }
    }

    // v spec, pod selector se píše template - to je návod, podle kterého kubernetes vyrobí jednotlivé kopie (pods) mojí aplikace
    template {
      metadata {
        // zde se to přesně musí shodovat s tím co je v v selector -> match_labels
        // Protože tím říkáme: "Tento deployment vyrábí kontejnery s tímto labelem, aby je mohl později poznat a ovládat"
        labels = {
          app = var.k3s_deployment_container_label
        }
      }
      // teď musíme říct, co v tom Podu skutečně poběží. To se píše také do spec bloku, ale spec bloku uvnitř bloku template
      spec {
        // zde definujeme seznam kontejnerů. I když budeme mít jen jeden.
        // můžeme toto opakovat vícekrát pokud bychom chtěli více kontejnerů
        container {
          name = var.k3s_deployment_container_name
          // Jaký obraz se má stáhnout
          // To je ten software, který distriibuujeme (distrbuovaný systém).
          // Tento konkrétní obraz od Nginx nám v prohlížeči ukáže, který z našich dvou agentů právě běží
          image = var.k3s_deployment_container_image
          port {
            // říkáme kubernetes: "Tato aplikace uvnitř kontejneru poslouchá na portu 80"
            container_port = var.k3s_deployment_container_port
          }
        }
      }
    }
  }
}
