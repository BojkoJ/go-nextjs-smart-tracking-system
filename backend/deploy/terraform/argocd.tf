resource "helm_release" "argocd" {
  // jméno, které uvidíme v Helmu
  name = var.helm_release_argocd_name
  // Repozitář, kde chceme lokalizovat příslušný Helm chart
  repository = var.helm_release_argocd_repo
  // jméno toho příslušného Helm chartu, který si chceme vytáhnout z repozitáře
  chart = var.helm_release_argocd_chart
  // tímhle terraformu řekneme: "instaluj to tam, kde jsi vytvořil namespace argocd"
  namespace = kubernetes_namespace.argocd.metadata[0].name

  // ArgoCD je velká aplikace. Spouští mnoho podů.
  // wait = true říká Terraform Helm provideru: "nekonči, dokud nejsou všechny pody Read"
  // timeout = 300 říká: "čekej max 300 sekund".
  // Bez těctho polí by Terraform řekl "hotovo" hned po odeslání manifestů do kubernetes, i když ArgoCD ještě startuje.
  timeout = 300
  wait = true

  // ArgoCD nepotřebuje žádné set hodnoty, výchozí konfigurace je pro nás dostačující
}