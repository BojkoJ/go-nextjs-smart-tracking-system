# Smart Tracking System

Distribuovaný cloudový systém pro sledování aktiv, postavený na Go mikroslužbách, Kubernetes a Next.js dashboardu. Systém simuluje sběr telemetrie v reálném čase od 50 nákladních kontejnerů na lodi plující z Pusanu (Jižní Korea) do Rotterdamu (Nizozemsko), zpracovává data skrze událostmi řízený pipeline a zobrazuje je na živém monitorovacím dashboardu.

---

## Obsah

1. [Přehled systému](#přehled-systému)
2. [Architektura](#architektura)
3. [Předpoklady](#předpoklady)
4. [Zprovoznění — kompletní bootstrap clusteru](#zprovoznění--kompletní-bootstrap-clusteru)
5. [Workflow při vývoji — GitOps s Tekton a ArgoCD](#workflow-při-vývoji--gitops-s-tekton-a-argocd)
6. [Lokální vývoj bez Kubernetes](#lokální-vývoj-bez-kubernetes)
7. [Testování](#testování)
8. [Mikroslužby](#mikroslužby)
   - [Ingest](#ingest-service)
   - [Processor](#processor-service)
   - [Query](#query-service)
   - [Simulator](#simulator-service)
   - [Frontend](#frontend)
9. [Kubernetes manifesty](#kubernetes-manifesty)
10. [Infrastruktura (Terraform)](#infrastruktura-terraform)
11. [k3d / k3s Cluster](#k3d--k3s-cluster)
12. [ArgoCD — GitOps kontinuální nasazování](#argocd--gitops-kontinuální-nasazování)
13. [Tekton — CI pipeline](#tekton--ci-pipeline)
14. [Observabilita](#observabilita)
15. [Přehled užitečných příkazů](#přehled-užitečných-příkazů)

---

## Přehled systému

Systém modeluje životní cyklus průmyslových aktiv (lodních kontejnerů) z pohledu IoT:

- **Simulator** generuje realistická čtení senzorů (GPS poloha, teplota, vlhkost, stav zámku) pro 50 kontejnerů v jednosekundových intervalech a odesílá je přes gRPC do Ingest služby.
- **Ingest** přijímá telemetrii přes gRPC a zveřejňuje ji do fronty NATS JetStream, aniž by blokoval Simulator.
- **Processor** asynchronně konzumuje zprávy z fronty, ukládá je do PostgreSQL, vynucuje obchodní pravidla (limity teploty, vlhkosti, stav zámku), generuje alerty při jejich porušení a řídí životní cyklus kontejnerů (přechody stavů: `new` -> `active` -> `decommissioned` při příjezdu do Rotterdamu).
- **Query** poskytuje REST API podpořené přímo PostgreSQL, podle vzoru CQRS — čtecí provoz je zcela oddělen od zápisového pipeline.
- **Frontend** (Next.js) vykresluje živou mapu pozice lodi, umožňuje uživateli vybrat libovolný ze 50 kontejnerů a zobrazuje telemetrický log v reálném čase spolu s aktuálními hodnotami senzorů.

---

## Architektura

```
Simulator  -> gRPC -> Ingest  -> NATS JetStream  -> Processor  -> PostgreSQL
                                                                         │
Frontend <── HTTP REST ──────────────────── Query <──────────────────────┘

Observabilita:
  Všechny služby  -> OTel SDK  -> Tempo (OTLP gRPC)     -> Grafana
  Všechny služby  -> /metrics    -> Prometheus          -> Grafana
  Všechny pody    -> stdout      -> Promtail  -> Loki   -> Grafana

CI/CD:
  Git push  -> Tekton (build + test + push obrazů)  -> ArgoCD (synchronizace manifestů)
```

**Použité návrhové vzory:**

- **Clean Architecture** — doménová vrstva nemá žádné externí závislosti; adaptéry (gRPC handler, HTTP handler, PostgreSQL repozitář, NATS klient) implementují portová rozhraní definovaná v jádru.
- **Dependency Injection** — každá funkce `main()` je kompozičním kořenem; konkrétní implementace jsou zde sestaveny a předávány jako rozhraní do nižších vrstev.
- **CQRS** — příkazy (příjem telemetrie) putují přes gRPC -> NATS -> Processor; dotazy (čtení dashboardu) jdou přímo do PostgreSQL přes Query.
- **Událostmi řízená architektura** — NATS JetStream zajišťuje doručení alespoň jednou s trvanlivými konzumenty; pokud Processor selže, zprávy čekají ve frontě a jsou znovu doručeny po restartu.
- **12-Factor App** — veškerá konfigurace je poskytována přes proměnné prostředí; do obrazů není vložena žádná konfigurace.

---

## Předpoklady

Na hostitelském počítači musí být před bootstrapem clusteru nainstalované tyto nástroje:

| Nástroj | Verze | Účel |
|---|---|---|
| Docker | 24+ | Container runtime pro uzly k3d |
| k3d | 5+ | Vytváří k3s cluster uvnitř Docker kontejnerů |
| Terraform | 1.0+ | Provisionuje cluster, namespace a Helm releases |
| kubectl | 1.28+ | Kubernetes CLI |
| Helm | 3.14+ | Instaluje Tekton a pracuje s Helm releases |
| Git | libovolná | Správa zdrojového kódu |

---

## Zprovoznění — kompletní bootstrap clusteru

Proveďte tyto kroky v pořadí na čistém počítači. Všechny příkazy se spouštějí z kořene repozitáře, není-li uvedeno jinak.

### 1. Klonování repozitáře

```bash
git clone https://github.com/BojkoJ/go-nextjs-smart-tracking-system.git
cd go-nextjs-smart-tracking-system
```

### 2. Provisionování clusteru a infrastruktury pomocí Terraform

Terraform vytvoří k3d cluster, všechny Kubernetes namespace a nainstaluje NATS, PostgreSQL, ArgoCD, Prometheus + Grafana, Tempo a Loki přes Helm.

```bash
cd backend/deploy/terraform
```

Vytvořte soubor `terraform.tfvars` s přihlašovacími údaji (tento soubor je v gitignore — nikdy ho necommitujte):

```hcl
postgres_password = "vase-postgres-heslo"
grafana_password  = "vase-grafana-heslo"
```

Poté spusťte:

```bash
terraform init
terraform apply
```

Terraform provede:
1. Vytvoření k3d clusteru s názvem `tracking-system-k3s-cluster` s 1 control-plane uzlem a 2 worker uzly.
2. Mapování portu `8080` na hostiteli na port `80` uvnitř clusteru (Traefik ingress).
3. Vytvoření namespace: `tracking-system`, `infrastructure`, `argocd`, `monitoring`.
4. Instalaci NATS JetStream, PostgreSQL (Bitnami), ArgoCD, kube-prometheus-stack, Grafana Tempo a Loki-stack přes Helm.

### 3. Instalace Tekton

Tekton nemá oficiální Helm chart a musí být nainstalován pomocí `kubectl`:

```bash
kubectl apply --filename https://storage.googleapis.com/tekton-releases/pipeline/latest/release.yaml
```

Počkejte, až budou Tekton pody připraveny:

```bash
kubectl wait --for=condition=ready pod --all -n tekton-pipelines --timeout=120s
```

### 4. Aplikování Tekton zdrojů

```bash
kubectl apply -f backend/deploy/tekton/workspace-pvc.yaml
kubectl apply -f backend/deploy/tekton/serviceaccount.yaml
kubectl apply -f backend/deploy/tekton/task-git-clone.yaml
kubectl apply -f backend/deploy/tekton/task-run-tests.yaml
kubectl apply -f backend/deploy/tekton/task-build-push.yaml
kubectl apply -f backend/deploy/tekton/task-update-manifest.yaml
kubectl apply -f backend/deploy/tekton/pipeline.yaml
```

### 5. Vytvoření Kubernetes secrets

Tyto secrets se nikdy necommitují do repozitáře. Vytvořte je ručně na každém novém clusteru.

**GHCR přihlašovací údaje** (Kaniko je používá pro push obrazů):

```bash
kubectl create secret generic ghcr-credentials \
  --from-file=config.json=$HOME/.docker/config.json \
  --namespace tracking-system
```

**Git přihlašovací údaje** (Tekton je používá pro push aktualizací manifestů):

```bash
kubectl create secret generic git-credentials \
  --type=kubernetes.io/basic-auth \
  --from-literal=username=VAS_GITHUB_USERNAME \
  --from-literal=password=VAS_GITHUB_PAT \
  --namespace tracking-system

kubectl annotate secret git-credentials \
  tekton.dev/git-0=https://github.com \
  --namespace tracking-system
```

**Aplikační secrets** (URL databáze a ostatní runtime secrets):

```bash
kubectl create secret generic tracking-secret \
  --from-literal=POSTGRES_URL="postgresql://tracking_user:vase-postgres-heslo@postgresql.infrastructure.svc.cluster.local:5432/tracking_db" \
  --namespace tracking-system
```

### 6. Konfigurace ArgoCD

Načtěte počáteční heslo administrátora ArgoCD:

```bash
kubectl get secret argocd-initial-admin-secret -n argocd \
  -o jsonpath='{.data.password}' | base64 -d
```

Port-forward ArgoCD UI:

```bash
kubectl port-forward svc/argocd-server 8888:80 -n argocd
```

Otevřete `http://localhost:8888`, přihlaste se jako `admin` a vytvořte aplikaci odkazující na váš fork tohoto repozitáře:

- **Název aplikace**: `smart-tracking-system`
- **Projekt**: `default`
- **Sync policy**: `Automatic` se self-heal
- **URL repozitáře**: `https://github.com/VAS_USERNAME/go-nextjs-smart-tracking-system.git`
- **Cesta**: `backend/deploy/k8s`
- **Cluster**: `https://kubernetes.default.svc`
- **Namespace**: `tracking-system`

ArgoCD nyní bude sledovat adresář `backend/deploy/k8s` a automaticky aplikovat veškeré změny.

### 7. Aplikování ConfigMap pro Grafana datasource

Grafana sidecar auto-detekuje ConfigMapy s labellem `grafana_datasource: "1"`:

```bash
kubectl apply -f backend/deploy/k8s/grafana-datasources.yaml
```

### 8. Spuštění počátečního CI pipeline

Spusťte první Tekton PipelineRun pro build všech obrazů a jejich push do GHCR:

```bash
kubectl create -f backend/deploy/tekton/pipeline-run.yaml
```

Sledujte průběh pipeline:

```bash
kubectl get pipelineruns -n tracking-system -w
kubectl get taskruns -n tracking-system --selector=tekton.dev/pipelineRun=<název-runu>
```

Po dokončení pipeline Tekton:
- Sestavil Docker obrazy pro všech pět služeb (`ingest`, `processor`, `query`, `simulator`, `frontend`).
- Pushoval je do `ghcr.io/bojkoj/<sluzba>:<git-sha>`.
- Aktualizoval image tagy v deployment manifestech a pushoval commit zpět do Gitu.

ArgoCD detekuje změnu manifestu a automaticky nasadí aktualizované pody.

### 9. Nastavení GHCR balíčků jako veřejných

Když Kaniko poprvé pushne nový image balíček, GitHub Container Registry ho vytvoří jako soukromý. Nastavte každý balíček jako veřejný na:

`https://github.com/users/VAS_USERNAME/packages/container/<sluzba>/settings`

### 10. Ověření, že vše běží

```bash
kubectl get pods -n tracking-system
kubectl get pods -n infrastructure
kubectl get pods -n monitoring
```

Všechny pody by měly být ve stavu `Running`. Otevřete dashboard na `http://localhost:8080`.

---

## Workflow při vývoji — GitOps s Tekton a ArgoCD

Po dokončení počátečního bootstrapu (viz sekce výše) probíhá iterativní vývoj tímto způsobem. Cluster zůstává spuštěný, ArgoCD průběžně sleduje repozitář.

### Změny v Go kódu nebo frontendu (vyžadují nový Docker obraz)

1. Proveďte změny v kódu a commitněte je na větev `master`.
2. Spusťte Tekton pipeline, který sestaví nové obrazy, pushne je do GHCR a automaticky aktualizuje image tagy v deployment manifestech:

```bash
kubectl create -f backend/deploy/tekton/pipeline-run.yaml
```

3. Sledujte průběh pipeline (trvá přibližně 35 minut kvůli sekvenčním Kaniko buildům):

```bash
kubectl get pipelineruns -n tracking-system -w
```

4. Po úspěšném pipeline runu Tekton pushne commit s novými image tagy do Gitu. ArgoCD tento commit detekuje (polling každé 3 minuty) a automaticky nasadí aktualizované pody.

5. Pro okamžité vynucení synchronizace bez čekání na ArgoCD polling:

```bash
kubectl annotate application smart-tracking-system \
  argocd.argoproj.io/refresh=hard -n argocd
```

### Změny pouze v Kubernetes manifestech (bez změny kódu)

Pokud měníte pouze manifesty v `backend/deploy/k8s/` (ConfigMap, resource limity, počet replik apod.), není nutné spouštět Tekton pipeline. Stačí commitnout a pushnout — ArgoCD změnu detekuje a aplikuje ji automaticky. Pro okamžitou aplikaci bez čekání:

```bash
# Přímá aplikace manifestů (obchází ArgoCD, vhodné pro rychlé ladění)
kubectl apply -f backend/deploy/k8s/

# Nebo vynuťte ArgoCD sync
kubectl annotate application smart-tracking-system \
  argocd.argoproj.io/refresh=hard -n argocd
```

### Vyčištění telemetrických dat (reset simulace)

Pokud chcete restartovat simulaci od začátku (prázdná databáze, nová data):

```bash
# Smazání všech telemetrických záznamů a alertů
kubectl exec -it -n infrastructure postgresql-0 -- psql -U tracking_user -d tracking_db -W \
  -c "TRUNCATE TABLE telemetry; TRUNCATE TABLE alerts;"

# Restart všech aplikačních podů
kubectl rollout restart deployment/ingest deployment/processor \
  deployment/query deployment/simulator deployment/frontend \
  -n tracking-system
```

---

## Lokální vývoj bez Kubernetes

Pro rychlý iterativní vývoj bez nutnosti spouštět k3d cluster. Infrastruktura (NATS, PostgreSQL) běží v Dockeru, služby spouštíme přímo přes `go run` a `npm run dev`.

### 1. Spuštění infrastruktury v Dockeru

```bash
# NATS JetStream
docker run -d --name nats-local -p 4222:4222 nats:latest -js

# PostgreSQL
docker run -d --name postgres-local \
  -e POSTGRES_USER=tracking_user \
  -e POSTGRES_PASSWORD=devpassword \
  -e POSTGRES_DB=tracking_db \
  -p 5432:5432 postgres:16
```

### 2. Inicializace databázového schématu

```bash
docker exec -i postgres-local psql -U tracking_user -d tracking_db <<'EOF'
CREATE TABLE IF NOT EXISTS assets (
    id              TEXT PRIMARY KEY,
    name            TEXT NOT NULL,
    max_temperature DOUBLE PRECISION NOT NULL,
    min_temperature DOUBLE PRECISION NOT NULL,
    max_humidity    DOUBLE PRECISION NOT NULL,
    status          TEXT NOT NULL DEFAULT 'new',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS telemetry (
    id           BIGSERIAL PRIMARY KEY,
    asset_id     TEXT NOT NULL REFERENCES assets(id),
    latitude     DOUBLE PRECISION NOT NULL,
    longitude    DOUBLE PRECISION NOT NULL,
    temperature  DOUBLE PRECISION NOT NULL,
    humidity     DOUBLE PRECISION NOT NULL,
    is_locked    BOOLEAN NOT NULL,
    timestamp_ns BIGINT NOT NULL,
    trace_id     TEXT NOT NULL,
    recorded_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS alerts (
    id         TEXT PRIMARY KEY,
    asset_id   TEXT NOT NULL REFERENCES assets(id),
    type       TEXT NOT NULL,
    message    TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_telemetry_asset_id ON telemetry(asset_id, timestamp_ns DESC);
CREATE INDEX IF NOT EXISTS idx_alerts_asset_id ON alerts(asset_id);
INSERT INTO assets (id, name, max_temperature, min_temperature, max_humidity, status)
SELECT 'asset-' || i, 'IQM Container #' || i, 28.0, 12.0, 45.0, 'active'
FROM generate_series(1, 50) AS i
ON CONFLICT (id) DO NOTHING;
EOF
```

### 3. Nastavení proměnných prostředí

Každá služba čte konfiguraci z proměnných prostředí. Doporučené hodnoty pro lokální vývoj:

| Proměnná | Hodnota pro lokální vývoj |
|---|---|
| `NATS_URL` | `nats://localhost:4222` |
| `GRPC_PORT` | `50051` |
| `HTTP_PORT` | `8080` |
| `INGEST_ADDR` | `localhost:50051` |
| `POSTGRES_URL` | `postgresql://tracking_user:devpassword@localhost:5432/tracking_db` |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | *(nenastavovat — traces se tisknou na stdout)* |

Proměnné lze exportovat v shellu nebo umístit do souboru `.env` a načíst pomocí `export $(cat .env | xargs)`.

### 4. Spuštění mikroslužeb

Každou službu spusťte v samostatném terminálu z adresáře `backend/`:

```bash
# Terminál 1 — Ingest
export NATS_URL=nats://localhost:4222 GRPC_PORT=50051
go run ./cmd/ingest

# Terminál 2 — Processor
export NATS_URL=nats://localhost:4222 POSTGRES_URL=postgresql://tracking_user:devpassword@localhost:5432/tracking_db
go run ./cmd/processor

# Terminál 3 — Query
export POSTGRES_URL=postgresql://tracking_user:devpassword@localhost:5432/tracking_db HTTP_PORT=8080
go run ./cmd/query

# Terminál 4 — Simulator
export INGEST_ADDR=localhost:50051
go run ./cmd/simulator
```

### 5. Spuštění frontendu

```bash
cd frontend
# .env.local už obsahuje NEXT_PUBLIC_API_URL=http://localhost:8080
npm run dev
```

Frontend je dostupný na `http://localhost:3000`. Query API běží na `http://localhost:8080`.

### 6. Spuštění testů

```bash
cd backend
go test ./...
```

### Zastavení a čistění

```bash
docker stop nats-local postgres-local
docker rm nats-local postgres-local
```

---

## Testování

Projekt používá výhradně automatizované unit testy bez externích závislostí. Integrační ani end-to-end testy nejsou součástí sady — infrastruktura (NATS, PostgreSQL) běží v Kubernetes a její chování je pokryto manuálním ověřením po každém nasazení.

### Framework a nástroje

| Nástroj | Účel |
|---|---|
| Go `testing` (stdlib) | Testovací framework — žádná externí závislost |
| `net/http/httptest` | In-process HTTP server pro testování REST handlerů bez síťového spojení |
| Mock structs (ručně psané) | Implementace port interfaců (`AssetRepository`, `TelemetryRepository`, `AlertRepository`, `EventProducer`) s nastavitelnými funkcemi pro simulaci chování repozitáře |

### Testové soubory a pokrytí

| Soubor | Počet testů | Pokrytí balíčku |
|---|---|---|
| `internal/adapters/handler/http_handler_test.go` | 15 | 87,8 % |
| `internal/adapters/handler/grpc_handler_test.go` | 5 | (součást handler balíčku) |
| `internal/services/processing_test.go` | 14 | 89,2 % |
| `internal/services/ingestion_test.go` | 12 | (součást services balíčku) |
| `cmd/simulator/main_test.go` | 18 | 39,8 % |
| `internal/common/config_test.go` | 8 | 40,0 % |
| **Celkem** | **72** | |

Nižší pokrytí `simulator` a `common` odpovídá povaze kódu: hlavní simulační smyčka (`runContainer` goroutina) a inicializace OpenTelemetry jsou těžko testovatelné bez skutečné infrastruktury a nejsou součástí byznys logiky.

### Co se testuje a proč

**HTTP handler** (`http_handler_test.go`, 15 testů) — každý REST endpoint má minimálně tři testovací případy: happy path (správný HTTP status a JSON tělo), not found (404 při prázdném repozitáři) a chyba repozitáře (500). Ověřuje se také správný `Content-Type` header a deserializovatelnost JSON odpovědi.

**gRPC handler** (`grpc_handler_test.go`, 5 testů) — ověřuje mapování všech 8 polí z `TelemetryRequest` na `domain.TelemetryData`, chování při chybě služby (`success=false` v response bez gRPC error kódu, protože chyby jsou součástí aplikačního protokolu) a okrajový případ nulového `timestamp_ns`.

**Processing service** (`processing_test.go`, 14 testů) — pokrývá všechna 4 obchodní pravidla pro generování alertů (teplota nad max, teplota pod min, vlhkost nad max, odemčený zámek), hraniční podmínky (přesně na limitu = žádný alert), souběžná porušení více pravidel najednou, propagaci chyb repozitáře a správnost obsahu vygenerovaného alertu (ID, AssetID).

**Ingestion service** (`ingestion_test.go`, 12 testů) — validace vstupních dat (prázdné `asset_id`, `NaN` teplota, GPS souřadnice mimo fyzikálně platný rozsah), hraniční hodnoty (souřadnice přesně na hranici jsou platné), ověření správného NATS subjektu, platnosti JSON payloadu a propagace chyby publisheru.

**Simulator** (`main_test.go`, 18 testů) — Haversinova vzdálenostní funkce (symetrie, nulový bod, čtvrtina rovníku, trasa Busan–Rotterdam), `computeShipPosition` (progrese v čase, fyzikální meze souřadnic), `computeTemperature` a `computeHumidity` (výstup v fyzikálně smysluplném rozsahu, vliv offsetu per kontejner).

### Strategie testování

Testy jsou navrženy jako **hradlo kvality** v CI pipeline: Tekton spouští `go test ./...` jako krok 2 (po `git-clone`, před jakýmkoliv Docker buildem). Selžou-li testy, pipeline se zastaví a žádný nový obraz není sestaven ani pushnut. To zaručuje, že v GHCR existuje pouze kód, který prošel testy.

Vývojářský workflow lokálně:

```bash
cd backend

# Spustit všechny testy
go test ./...

# Spustit testy s pokrytím kódu
go test -cover ./...

# Spustit konkrétní balíček s verbose výstupem
go test -v ./internal/services/...

# Spustit jeden konkrétní test
go test -v -run TestProcessTelemetry_TemperatureAboveMax_CreatesMaxAlert ./internal/services/
```

---

## Mikroslužby

Všechny Go služby sdílí jediný vícefázový `Dockerfile` v `backend/`. Build argument `SERVICE` vybírá, který vstupní bod `cmd/<sluzba>` se zkompiluje.

### Ingest Service

**Účel:** Vstupní bod pro příjem telemetrie. Přijímá gRPC volání od Simulatoru, serializuje payload do JSON a zveřejňuje ho do NATS JetStream subjektu `telemetry.ingest`.

**Proč gRPC:** Simulator odesílá 50 požadavků za sekundu souběžně. gRPC používá Protocol Buffers (binární kódování, 5–10× menší než JSON přes HTTP). gRPC server runtime automaticky přiděluje každé příchozí volání do vlastní goroutiny — žádný manuální kód pro souběžnost není potřeba.

**Klíčové soubory:**

| Soubor | Popis |
|---|---|
| `backend/cmd/ingest/main.go` | Kompoziční kořen — sestaví NATS klienta, gRPC handler, spustí server a metrics endpoint |
| `backend/internal/adapters/handler/grpc_handler.go` | Překládá `*pb.TelemetryRequest` na `domain.TelemetryData`, volá `IngestionService`, inkrementuje metriku `IngestRequests` |
| `backend/internal/adapters/eventbus/nats_client.go` | NATS JetStream klient — publikuje zprávy, vkládá W3C trace kontext do NATS hlaviček |
| `backend/internal/services/ingestion.go` | `IngestionService` — serializuje doménovou strukturu do JSON a volá `EventProducer.PublishMessage` |
| `backend/proto/telemetry.proto` | Protobuf kontrakt pro `SendTelemetry` RPC |

**Kubernetes zdroje:** `ingest-deployment.yaml`, `ingest-service.yaml`

Deployment vystavuje dva porty: `50051` (gRPC) a `9090` (Prometheus metriky). Service vystavuje oba. Liveness probe používá TCP socket na portu `50051`; readiness probe totéž s kratší počáteční prodlevou.

---

### Processor Service

**Účel:** Konzumuje telemetrické zprávy z NATS, ukládá je do PostgreSQL, vyhodnocuje obchodní pravidla, generuje alerty a řídí životní cyklus kontejnerů.

**Vynucovaná obchodní pravidla:**

| Pravidlo | Spouštěč | Typ alertu |
|---|---|---|
| Teplota překročí `MaxTemperature` (28°C) | `temperature > asset.MaxTemperature` | `temperature_exceeded_max_limit` |
| Teplota klesne pod `MinTemperature` (12°C) | `temperature < asset.MinTemperature` | `temperature_exceeded_min_limit` |
| Vlhkost překročí `MaxHumidity` (45% RH) | `humidity > asset.MaxHumidity` | `humidity_exceeded_max_limit` |
| Kontejner odemčen | `!isLocked` | `container_unlocked` |
| Příjezd do Rotterdamu | Haversinova vzdálenost ≤ 50 km | Status -> `decommissioned` |

**Klíčové soubory:**

| Soubor | Popis |
|---|---|
| `backend/cmd/processor/main.go` | Kompoziční kořen — sestaví PostgreSQL repozitář, `ProcessingService`, NATS subscription |
| `backend/internal/services/processing.go` | Byznys logika — validuje limity, zapisuje telemetrii a alerty, aktualizuje stav assetů |
| `backend/internal/adapters/repository/postgres_repo.go` | PostgreSQL implementace `AssetRepository`, `TelemetryRepository`, `AlertRepository` |
| `backend/internal/common/metrics.go` | Prometheus čítače a histogramy (`tracking_telemetry_processed_total`, `tracking_alerts_generated_total`, `tracking_processing_duration_seconds`) |

**Kubernetes zdroje:** `processor-deployment.yaml`, `processor-service.yaml`

Deployment vystavuje port `9090` pro Prometheus metriky. Service vystavuje pouze port `metrics` (port `9090`); žádný externě přístupný port není potřeba, protože Processor je čistě událostmi řízený.

---

### Query Service

**Účel:** Obsluhuje REST API konzumované Frontendem. Čte přímo z PostgreSQL — bez zapojení NATS — podle čtecí cesty vzoru CQRS.

**API endpointy:**

| Metoda | Cesta | Popis |
|---|---|---|
| `GET` | `/assets` | Výpis všech 50 assetů |
| `GET` | `/assets/{id}` | Načtení jednoho assetu podle ID |
| `GET` | `/assets/{id}/telemetry` | Načtení posledního telemetrického záznamu |
| `GET` | `/assets/{id}/telemetry/history` | Načtení kompletní telemetrické historie |
| `GET` | `/alerts` | Výpis všech alertů |
| `GET` | `/assets/{id}/alerts` | Výpis alertů pro konkrétní asset |
| `GET` | `/metrics` | Prometheus metrics endpoint |

**Klíčové soubory:**

| Soubor | Popis |
|---|---|
| `backend/cmd/query/main.go` | Kompoziční kořen — sestaví PostgreSQL repozitář, HTTP handler, spustí server |
| `backend/internal/adapters/handler/http_handler.go` | chi router, JSON kódování, obsluha chyb |

**Kubernetes zdroje:** `query-deployment.yaml`, `query-service.yaml`

Deployment vystavuje port `8080`. Liveness a readiness proby používají `GET /assets` přes HTTP. HorizontalPodAutoscaler (`hpa.yaml`) škáluje Deployment Query mezi 1 a 3 replikami při 70% využití CPU.

---

### Simulator Service

**Účel:** Simuluje 50 nákladních kontejnerů na jediné lodi plující z Pusanu do Rotterdamu. Každý kontejner běží ve vlastní goroutině a odesílá jeden telemetrický záznam za sekundu přes gRPC do Ingest služby.

**Detaily simulace:**

- **Trasa:** 10 GPS waypointů z Pusanu (35,10°N, 129,04°E) do Rotterdamu (51,98°N, 4,05°E), celkem ~20 000 km. Rychlost je 20× zrychlena, takže celá plavba trvá přibližně 40 minut reálného času.
- **Teplota:** Sinusoidální denní cyklus (základ 14–26°C), plus náhodný šum senzoru (±0,8°C), plus offset pro každý kontejner (±2°C) modelující různé zóny nákladového prostoru.
- **Vlhkost:** 12hodinový sinusoidální cyklus (základ 30–40% RH) plus šum a offset pro každý kontejner.
- **Poloha:** Délky segmentů na základě Haversinovy formule zajišťují fyzikálně správnou rychlost cestování; lineární interpolace v rámci každého segmentu udržuje výpočet levný.

**Klíčové soubory:**

| Soubor | Popis |
|---|---|
| `backend/cmd/simulator/main.go` | Veškerá logika simulace — `computeShipPosition`, `computeTemperature`, `computeHumidity`, goroutina `runContainer` |

**Kubernetes zdroje:** `simulator-deployment.yaml`

Simulator není vystaven externě (žádný Service manifest). Připojuje se k `ingest-service.tracking-system.svc.cluster.local:50051` přes DNS clusteru.

---

### Frontend

**Účel:** Single-page Next.js dashboard zobrazující živou mapu pozice lodi a panel s telemetrickým logem pro každý kontejner.

**Stack:** Next.js 16, React 19, TypeScript, Tailwind CSS, shadcn/ui, Lucide React, Leaflet (přes CDN), SWR, Axios, Zod.

**Tok dat:** SWR polluje `GET /assets` každých 10 sekund. Při prvním načtení aplikace souběžně načte poslední telemetrický záznam pro každý asset, aby získala GPS souřadnice a umístila značku lodi na mapu. Kliknutím na značku se otevře `ContainerPanel`, který polluje `GET /assets/{id}/telemetry` každých 5 sekund a `GET /assets/{id}/telemetry/history` každých 8 sekund.

V clusteru je `NEXT_PUBLIC_API_URL` vložen jako prázdný řetězec během Docker buildu. Požadavky prohlížeče na `/assets/...` a `/alerts/...` jsou relativní ke stejnému originu a jsou routovány Traefikem do Query služby.

**Klíčové soubory:**

| Soubor | Popis |
|---|---|
| `frontend/app/page.tsx` | Kořenová stránka — sestavuje `ShipMap` a `ContainerPanel`, spravuje stav zvoleného kontejneru |
| `frontend/components/ShipMap.tsx` | Leaflet mapa načtená přes script tag; umisťuje čtvercové značky pro každou unikátní pozici lodi; naslouchá stavu `leafletReady` před vykreslením značek, aby předešla race condition |
| `frontend/components/ContainerPanel.tsx` | Panel na pravé straně — výběr kontejneru (shadcn Select), stats bar, scrollovatelný telemetrický log |
| `frontend/components/StatsBar.tsx` | Zobrazuje aktuální teplotu, limity max/min teploty, aktuální vlhkost, limit max vlhkosti, stav zámku |
| `frontend/components/TelemetryLog.tsx` | Telemetrická historie seřazená od nejnovějšího; monospace log formát |
| `frontend/lib/api.ts` | Axios klient se Zod parsováním každé odpovědi |
| `frontend/lib/schemas.ts` | Zod schémata odpovídající Go doménovým typům |
| `frontend/hooks/useAssets.ts` | SWR hook pro seznam assetů a poslední telemetrii |
| `frontend/hooks/useContainerDetail.ts` | SWR hooky pro telemetrickou historii a alerty |
| `frontend/Dockerfile` | Vícefázový: Node 22 builder kompiluje standalone Next.js výstup; Node 22 Alpine runner spouští `server.js` jako non-root uživatel |

**Kubernetes zdroje:** `frontend-deployment.yaml`, `frontend-service.yaml`

---

## Kubernetes manifesty

Všechny manifesty jsou v `backend/deploy/k8s/`. ArgoCD sleduje tento adresář a automaticky aplikuje změny při každém Git push na větev `master`.

| Manifest | Kind | Popis |
|---|---|---|
| `configmap.yaml` | ConfigMap | Sdílená nekonfidenciální konfigurace: `NATS_URL`, `GRPC_PORT`, `HTTP_PORT`, `INGEST_ADDR`, `OTEL_EXPORTER_OTLP_ENDPOINT` |
| `secret.yaml` | Secret | **V gitignore.** Obsahuje `POSTGRES_URL` a ostatní citlivé hodnoty. Musí být vytvořen ručně na každém clusteru. |
| `db-init-job.yaml` | Job | ArgoCD `PreSync` hook. Spouští `psql` proti PostgreSQL před startem aplikačních podů. Vytvoří tabulky `assets`, `telemetry` a `alerts`; vloží 50 počátečních záznamů assetů. Po úspěchu je automaticky smazán (delete policy `HookSucceeded`). |
| `ingest-deployment.yaml` | Deployment | Ingest mikroslužba. Image tag je aktualizován Tektonem při každém úspěšném pipeline runu. |
| `ingest-service.yaml` | Service | Vystavuje gRPC port `50051` a Prometheus metrics port `9090`. |
| `processor-deployment.yaml` | Deployment | Processor mikroslužba. |
| `processor-service.yaml` | Service | Vystavuje pouze Prometheus metrics port `9090`. |
| `query-deployment.yaml` | Deployment | Query mikroslužba. Liveness a readiness proby na `GET /assets` port `8080`. |
| `query-service.yaml` | Service | Vystavuje HTTP port `8080`. |
| `hpa.yaml` | HorizontalPodAutoscaler | Škáluje Deployment `query` mezi 1 a 3 replikami; cílové využití CPU 70%. |
| `simulator-deployment.yaml` | Deployment | Simulator mikroslužba. Žádný přidružený Service; komunikuje odchozím směrem do Ingest. |
| `frontend-deployment.yaml` | Deployment | Next.js dashboard. `imagePullPolicy: Always`, protože image tagy mohou být během vývoje opakovaně použity. |
| `frontend-service.yaml` | Service | Vystavuje HTTP port `3000`. |
| `ingress.yaml` | Ingress | Traefik ingress. Směruje prefixy cest `/assets` a `/alerts` do Query služby; veškerý ostatní provoz jde do Frontendu. |
| `servicemonitor.yaml` | ServiceMonitor (×3) | Prometheus Operator zdroje pro `ingest`, `processor` a `query`. Každý nese label `release: kube-prometheus-stack`, aby ho Prometheus Operator automaticky detekoval. |
| `grafana-datasources.yaml` | ConfigMap | Provisionován do namespace `monitoring` s labelem `grafana_datasource: "1"`. Grafana sidecar ho zachytí a nakonfiguruje datasource Tempo a Loki bez restartu podu. |

---

## Infrastruktura (Terraform)

Veškerá infrastruktura je deklarována v `backend/deploy/terraform/`. Spuštění `terraform apply` je idempotentní — opakované spuštění proti existujícímu clusteru srovná případné odchylky.

**Soubory:**

| Soubor | Obsah |
|---|---|
| `main.tf` | Deklarace providerů (`moio/k3d`, `hashicorp/kubernetes`, `hashicorp/helm`) a jejich konfigurace (přihlašovací údaje clusteru zapojeny přímo ze zdroje k3d) |
| `cluster.tf` | Zdroj `k3d_cluster` — název clusteru, počty serverů/agentů, verze k3s obrazu, mapování hostitelského portu (`8080` -> `80`) |
| `namespaces.tf` | Čtyři zdroje `kubernetes_namespace`: `infrastructure`, `tracking-system`, `argocd`, `monitoring` |
| `infrastructure.tf` | Šest zdrojů `helm_release`: NATS (JetStream povolen), PostgreSQL (Bitnami, persistence vypnuta pro lokální vývoj), kube-prometheus-stack (Prometheus + Grafana + AlertManager, NodePort), Grafana Tempo (OTLP gRPC přijímač na `0.0.0.0:4317`), Loki-stack (Promtail jako DaemonSet, Grafana a Prometheus vypnuty) |
| `argocd.tf` | ArgoCD `helm_release` s `wait: true` a timeoutem 480 sekund |
| `variables.tf` | Všechny deklarace vstupních proměnných s popisy a výchozími hodnotami |
| `terraform.tfvars` | **V gitignore.** Tajné hodnoty: `postgres_password`, `grafana_password` |

**Import existujících releases do stavu** (potřebné pokud byly Helm releases instalovány ručně před nastavením Terraformu):

```bash
cd backend/deploy/terraform
terraform import kubernetes_namespace.monitoring monitoring
terraform import helm_release.kube_prometheus_stack monitoring/kube-prometheus-stack
terraform import helm_release.tempo monitoring/tempo
terraform import helm_release.loki_stack monitoring/loki
```

---

## k3d / k3s Cluster

**k3s** je odlehčená, produkčně připravená distribuce Kubernetes od Rancher. Standardně dodává Traefik jako Ingress controller, používá SQLite místo etcd a je navržen pro prostředí s omezenými zdroji.

**k3d** zabaluje k3s tak, aby každý uzel clusteru běžel uvnitř Docker kontejneru na hostitelském počítači, což ho činí vhodným pro lokální vývoj.

**Topologie clusteru v tomto projektu:**

```
Hostitelský počítač (Ubuntu)
└── Docker síť: 172.19.0.0/16
    ├── k3d-tracking-system-k3s-cluster-server-0   (control plane)
    ├── k3d-tracking-system-k3s-cluster-agent-0    (worker uzel)
    ├── k3d-tracking-system-k3s-cluster-agent-1    (worker uzel)
    └── k3d-tracking-system-k3s-cluster-serverlb   (load balancer, port 8080 -> 80)
```

**Mapování portů:** Hostitelský port `8080` je přesměrován k3d load-balancer kontejnerem do Traefiku uvnitř clusteru na port `80`. Veškerý HTTP provoz do aplikace (dashboard a API) prochází přes `http://localhost:8080`.

**Kubeconfig:** Terraform zapisuje přihlašovací údaje clusteru přímo do konfigurace providera; pro přístup přes `kubectl` zapisuje k3d kontext do `~/.kube/config` automaticky.

**Poznámka ke konfiguraci CoreDNS:** Výchozí nastavení CoreDNS přeposílá DNS dotazy do `/etc/resolv.conf` (brána Docker bridge). Pokud se DNS resoluce externích registrů stane nespolehlivou pod zátěží (např. při paralelních Kaniko buildech), aplikujte následující patch pro explicitní použití brány Docker s vyšší cache TTL:

```bash
kubectl patch configmap coredns -n kube-system --type merge -p '{
  "data": {
    "Corefile": ".:53 {\n    errors\n    health\n    ready\n    kubernetes cluster.local in-addr.arpa ip6.arpa {\n      pods insecure\n      fallthrough in-addr.arpa ip6.arpa\n    }\n    hosts /etc/coredns/NodeHosts {\n      ttl 60\n      reload 15s\n      fallthrough\n    }\n    prometheus :9153\n    forward . 172.19.0.1 8.8.8.8 8.8.4.4\n    cache 300\n    loop\n    reload\n    loadbalance\n    import /etc/coredns/custom/*.override\n}\nimport /etc/coredns/custom/*.server\n"
  }
}'
kubectl rollout restart deployment/coredns -n kube-system
```

---

## ArgoCD — GitOps kontinuální nasazování

ArgoCD průběžně srovnává živý stav clusteru s požadovaným stavem deklarovaným v Gitu. Adresář `backend/deploy/k8s` je jediným zdrojem pravdy pro všechny aplikační manifesty.

**Jak to funguje v tomto projektu:**

1. Tekton CI pipeline sestaví nové Docker obrazy a commitne aktualizované image tagy zpět do `backend/deploy/k8s/*-deployment.yaml` na větvi `master`.
2. ArgoCD detekuje Git změnu (standardně polling každé 3 minuty) a aplikuje aktualizované manifesty.
3. Před aplikováním spustí ArgoCD Job `db-init` definovaný s anotací `argocd.argoproj.io/hook: PreSync`. Tím zajistí existenci databázového schématu a seed dat před startem aplikačních podů.
4. Pokud pod selže při přechodu do stavu Ready, ArgoCD označí aplikaci jako `Degraded` a problém je viditelný v UI.

**Přístup do ArgoCD UI:**

```bash
kubectl port-forward svc/argocd-server 8888:80 -n argocd
# Otevřete http://localhost:8888
# Uživatel: admin
# Heslo: viz krok 6 v sekci Zprovoznění
```

**Vynucení manuální synchronizace:**

```bash
kubectl annotate application smart-tracking-system \
  argocd.argoproj.io/refresh=hard -n argocd
```

---

## Tekton — CI pipeline

Tekton běží uvnitř clusteru (namespace `tracking-system`) a na vyžádání spouští CI pipeline. Každý pipeline run:

1. **git-clone** — Naklonuje repozitář do sdíleného PVC (`tekton-workspace-pvc`). Výstupem je krátký Git SHA jako výsledná hodnota použitá pro tagování obrazů.
2. **run-tests** — Spustí `go test ./...` uvnitř adresáře `backend/`.
3. **build-push-ingest** -> **build-push-processor** -> **build-push-query** -> **build-push-simulator** -> **build-push-frontend** — Sestavuje Docker obrazy sekvenčně pomocí Kaniko. Každý obraz je pushnut do GHCR se dvěma tagy: krátký SHA (neměnný, používaný v manifestech) a `latest` (měnitelný ukazatel).
4. **update-manifest** — Aktualizuje pole `image:` v každém souboru `*-deployment.yaml` pomocí `sed`, commitne změnu a pushne ji do repozitáře.

Build tasky běží sekvenčně, aby se předešlo vyčerpání DNS pod zátěží z více souběžných Kaniko podů.

**Spuštění nového pipeline runu:**

```bash
kubectl create -f backend/deploy/tekton/pipeline-run.yaml
```

**Sledování pipeline runu:**

```bash
# Výpis všech runů
kubectl get pipelineruns -n tracking-system

# Sledování konkrétního runu
kubectl get taskruns -n tracking-system \
  --selector=tekton.dev/pipelineRun=<název-runu> -w

# Stream logů z konkrétního task podu
kubectl logs -n tracking-system <název-taskrun-podu> -f
```

**Aktualizace Tekton zdrojů po změnách manifestů:**

```bash
kubectl apply -f backend/deploy/tekton/task-build-push.yaml
kubectl apply -f backend/deploy/tekton/task-update-manifest.yaml
kubectl apply -f backend/deploy/tekton/pipeline.yaml
```

---

## Observabilita

Tři pilíře observability jsou nasazeny v namespace `monitoring`.

### Metriky (Prometheus + Grafana)

Vlastní aplikační metriky vystavené na `:9090/metrics` pro Ingest a Processor, a na `:8080/metrics` pro Query:

| Metrika | Typ | Popis |
|---|---|---|
| `tracking_ingest_requests_total` | CounterVec | gRPC požadavky přijaté Ingestem, labelované podle `success` |
| `tracking_telemetry_processed_total` | Counter | Telemetrické zprávy zpracované Processorem |
| `tracking_alerts_generated_total` | CounterVec | Vygenerované alerty, labelované podle `type` |
| `tracking_processing_duration_seconds` | Histogram | End-to-end latence zpracování pro každou telemetrickou zprávu |

Zdroje ServiceMonitor v `backend/deploy/k8s/servicemonitor.yaml` konfigurují Prometheus Operator pro scraping všech tří služeb každých 15 sekund.

**Přístup do Grafany:**

```bash
kubectl port-forward svc/kube-prometheus-stack-grafana 3000:80 -n monitoring
# Otevřete http://localhost:3000
# Uživatel: admin
# Heslo: hodnota grafana_password v terraform.tfvars
```

### Traces (OpenTelemetry + Grafana Tempo)

Všechny čtyři Go služby inicializují `TracerProvider` při startu (`backend/internal/common/otel.go`). Pokud je nastavena proměnná `OTEL_EXPORTER_OTLP_ENDPOINT` (přes ConfigMap v Kubernetes), traces jsou exportovány přes OTLP gRPC do Tempa na `tempo.monitoring.svc.cluster.local:4317`. Bez proměnné prostředí jsou traces tištěny na stdout (vhodné pro lokální vývoj).

Trace kontext se propaguje přes hranice služeb pomocí standardu W3C TraceContext:
- Simulator -> Ingest: přes `otelgrpc` stats handler (gRPC metadata)
- Ingest -> NATS -> Processor: přes vlastní `TextMapCarrier` v hlavičkách NATS zpráv

**Zobrazení traces v Grafaně:**

Otevřete Grafanu -> Explore -> vyberte datasource **Tempo** -> použijte záložku Search pro dotaz podle názvu služby.

### Logy (Promtail + Loki)

Promtail běží jako DaemonSet na každém uzlu clusteru a čte soubory v `/var/log/pods`. Všechny strukturované JSON logy zapisované Go službami (`log/slog`) jsou sbírány a indexovány v Loki.

Všechny Go služby používají `log/slog` s JSON výstupem, strukturovanými poli a `service` jako top-level polem pro snadné filtrování.

**Dotazování logů v Grafaně:**

Otevřete Grafanu -> Explore -> vyberte datasource **Loki** -> filtrujte podle `{namespace="tracking-system"}`.

---

## Přehled užitečných příkazů

### Správa clusteru

```bash
# Výpis k3d clusterů
k3d cluster list

# Start / stop clusteru (bez smazání)
k3d cluster start tracking-system-k3s-cluster
k3d cluster stop tracking-system-k3s-cluster

# Úplné smazání clusteru
k3d cluster delete tracking-system-k3s-cluster

# Přepnutí kubectl kontextu (při existenci více clusterů)
kubectl config use-context k3d-tracking-system-k3s-cluster
```

### Aplikační pody

```bash
# Výpis všech podů ve všech namespace
kubectl get pods -A

# Sledování stavu podů v tracking-system
kubectl get pods -n tracking-system -w

# Stream logů ze služby
kubectl logs -n tracking-system deployment/processor -f

# Restart deploymentu (načtení nových hodnot z ConfigMap)
kubectl rollout restart deployment/ingest deployment/processor deployment/query -n tracking-system

# Popis podu (události, využití zdrojů)
kubectl describe pod -n tracking-system <název-podu>
```

### Tekton

```bash
# Spuštění nového pipeline runu
kubectl create -f backend/deploy/tekton/pipeline-run.yaml

# Výpis pipeline runů
kubectl get pipelineruns -n tracking-system

# Výpis task runů pro konkrétní pipeline run
kubectl get taskruns -n tracking-system \
  --selector=tekton.dev/pipelineRun=<název-runu>

# Stream logů z task podu
kubectl logs -n tracking-system <název-taskrun-podu> --all-containers -f

# Smazání všech dokončených pipeline runů (čistění)
kubectl delete pipelineruns -n tracking-system \
  --field-selector=status.conditions[0].reason=Succeeded
```

### ArgoCD

```bash
# Port-forward ArgoCD UI
kubectl port-forward svc/argocd-server 8888:80 -n argocd

# Vynucení hard refresh (přehodnocení Gitu, ignorování cache)
kubectl annotate application smart-tracking-system \
  argocd.argoproj.io/refresh=hard -n argocd

# Načtení hesla administrátora ArgoCD
kubectl get secret argocd-initial-admin-secret -n argocd \
  -o jsonpath='{.data.password}' | base64 -d
```

### Helm

```bash
# Výpis všech Helm releases ve všech namespace
helm list -A

# Kontrola hodnot Helm release
helm get values kube-prometheus-stack -n monitoring

# Upgrade Helm release (např. po změně Terraform proměnné)
cd backend/deploy/terraform && terraform apply
```

### Grafana a observabilita

```bash
# Port-forward Grafany
kubectl port-forward svc/kube-prometheus-stack-grafana 3000:80 -n monitoring

# Kontrola Prometheus targets (měly by zobrazit tracking-ingest, tracking-processor, tracking-query)
kubectl port-forward svc/kube-prometheus-stack-prometheus 9090:9090 -n monitoring
# Otevřete http://localhost:9090/targets
```

### Terraform

```bash
cd backend/deploy/terraform

# Náhled změn bez aplikování
terraform plan

# Aplikování změn
terraform apply

# Import existujícího zdroje do Terraform stavu
terraform import helm_release.loki_stack monitoring/loki

# Zničení všeho (nevratné)
terraform destroy
```
