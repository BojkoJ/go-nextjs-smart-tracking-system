# ---------------------------------------------------------------------------------------------------------
#                                         MULTI-STAGE DOCKERFILE
# ---------------------------------------------------------------------------------------------------------
# Co tento soubor dělá:
# Definuje jak sestavit Docker image pro libovolnou ze 4 mikroslužeb tohoto projektu.
# Pomocí ARG SERVICE určíme kterou službu buildíme - jeden soubor, 4 různé výsledné image.
#
# Co je multi-stage build:
# Dockerfile má dvě fáze (stages). První fáze (builder) kompiluje Go kód a potřebuje
# Go toolchain (~600 MB). Druhá fáze (runner) vezme jen výslednou binárku a spustí ji
# v minimálním Alpine image (~20 MB). Výsledný Docker image neobsahuje Go toolchain - jen binárku.
#
# Jak použít:
#   docker build --build-arg SERVICE=ingest    -t ingest:latest    .
#   docker build --build-arg SERVICE=processor -t processor:latest .
#   docker build --build-arg SERVICE=query     -t query:latest     .
#   docker build --build-arg SERVICE=simulator -t simulator:latest .
#
# Všechny příkazy se spouštějí z adresáře backend/ (kde je go.mod).
# ---------------------------------------------------------------------------------------------------------

# Stage 1: Builder
# golang:1.25-alpine je oficiální Go image postavený na Alpine Linuxu
FROM golang:1.25-alpine AS builder

# apk je package manager Alpine Linuxu (jako apt na Ubuntu nebo yum na CentOS)
# git potřebuje go mod download pro stahování modulů uložených v git repozitářích
# --no-cache: nestahuje index balíčků do cache - menší vrstva image
RUN apk add --no-cache git

# WORKDIR nastaví pracovní adresář uvnitř kontejneru
WORKDIR /app

# Kopírujeme go.mod a go.sum JAKO PRVNÍ - kvůli Docker layer caching.
COPY go.mod go.sum ./

# Stáhne všechny Go závislosti definované v go.mod do cache uvnitř builderu
# Po tomto kroku jsou všechny knihovny dostupné offline - COPY . .
RUN go mod download

# Teprve teď kopírujeme všechen zdrojový kód z build contextu (backend/ adresář) do /app
# Tečka na konci = kopírujeme do aktuálního WORKDIR (/app)
COPY . .

# ARG definuje build-time proměnnou - předává se přes docker build --build-arg SERVICE=ingest
# Dostupná je jen v době buildu, ne za runtime (na rozdíl od ENV)
ARG SERVICE

# Kompilujeme Go binárku:
# CGO_ENABLED=0: zakáže CGO (C bindings) - čistě statická binárka bez C závislostí
#                nutné pro Alpine runner který má jiný libc než builder (musl vs glibc)
# GOOS=linux:    crosscompilujeme pro Linux i když builder běží na Mac nebo Windows
# -ldflags="-w -s": -w odstraní DWARF debug info, -s odstraní symbol table → binárka o 30-40% menší
# -o /app/service: výstupní binárka se jmenuje "service" a leží v /app
# ./cmd/${SERVICE}: vstupní balíček - ${SERVICE} se nahradí hodnotou ARG (např. ingest)
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/service ./cmd/${SERVICE}

# Stage 2: Runner
# Začínáme NOVÝ čistý image - z builderu si vezmeme jen binárku
# alpine:3.19 je minimální Linux bez Go toolchainu (~7 MB celkem)
# Výsledný image neobsahuje zdrojový kód, go.mod, ani Go kompilátor
FROM alpine:3.19

# ca-certificates obsahuje důvěryhodné kořenové certifikáty (CA bundle)
# Potřebné pro TLS spojení: gRPC s TLS, HTTPS requesty, ověření certifikátů
# Bez ca-certificates by TLS handshake selhal s "certificate signed by unknown authority"
RUN apk add --no-cache ca-certificates

# Stejný pracovní adresář jako v builderu
WORKDIR /app

# Hlavní instrukce multi-stage buildu:
# Zkopíruje POUZE soubor /app/service z builder stage do aktuálního image
# Nic jiného z builderu (zdrojáky, toolchain, cache) se do runner image nedostane
COPY --from=builder /app/service ./service

# Bezpečnostní best practice: vytvoříme dedikovaného uživatele bez root oprávnění
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Přepneme na non-root uživatele pro všechny následující instrukce a za runtime
USER appuser

# ENTRYPOINT definuje co se spustí při docker run (nebo v K8s Podu)
# ["./service"] = exec form (ne shell form) - binárka dostane PID 1
ENTRYPOINT ["./service"]