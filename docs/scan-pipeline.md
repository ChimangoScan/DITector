# Pipeline de varredura — DITector / ChimangoScan

## Visão geral em 3 estágios

```
STAGE I — Crawler (Go)
  └─ Mongo dockerhub_data
       repositories_data  12.716.568 repos indexados
       tags_data           5.603.957 tags
       images_data         6.500.000+ imagens (digest + metadados)
       crawler_keywords    2.000.000+ keywords usadas na busca

STAGE II — Builder (Go)
  └─ Lê repositories_data, faz docker pull, extrai histórico de layers
  └─ Constrói grafo IS_BASE_OF no Neo4j: (Layer)-[:IS_BASE_OF]->(Layer)
  └─ Status (2026-05-14): 4.961.614 repos buildados (39% de 12,7 M)
       ~50 M edges no grafo (relações transitivas incluídas)
       taxa: ~207 repos/min → ETA ~30 dias para 100%

STAGE III — Scanner distribuído (ChimangoScan)
  └─ Fila SQLite (ditector.db) com ~504 k jobs rankeados por exposure
  └─ Workers em gpu1 + a9 + l01..l09 consomem da mesma fila via HTTP
  └─ 6 scanners estáticos por imagem: syft, trivy, grype, osv, dockle, trufflehog
```

---

## Stage I — Como o crawler descobre repos (busca por prefixo)

O Docker Hub Search API (`/v2/search/repositories/?query=<term>`) retorna no máximo **10.000 resultados** por query. O crawler contorna isso com uma travessia de trie de prefixos:

```
Seeds iniciais: a b c d ... z 0 1 ... 9 - _    (38 seeds, um por caractere)

Para cada prefixo:
  1. Chama Search API — retorna até 10.000 repos que contêm o prefixo
  2. Salva repos novos no Mongo (dedup por namespace/name)
  3. Se count >= 10.000 (teto da API) → expande:
       prefixo "py" → enfileira "pya", "pyb", ..., "py0", "py-", "py_"
  4. Se count < 10.000 → prefixo esgotado, marca como done

Prioridade: 255 - len(prefixo) → prefixos curtos primeiro (BFS over trie)
Token plateau: se um prefixo com "-" ou "_" retorna 10k mas 0 repos novos
              → filhos recebem priority=-1 (depriorizados)
```

**Resultado**: a travessia cobre o espaço de nomes do Docker Hub sistematicamente, sem depender de links entre repos. A `crawler_keywords` collection no Mongo armazena o estado de cada prefixo (`pending` / `processing` / `done`), tornando o crawler **retomável** após paradas.

---

## Stage III em detalhe — pipeline de scan de uma imagem

```
Job claim (worker)
    │
    ▼
docker pull <image>@sha256:<digest>       ← pinado por digest; não muda durante o scan
    │
    ▼
docker save → /cache/tars/<slug>.tar      ← produzido UMA vez, compartilhado pelos scanners
    │
    ├──► [syft]       ─── lê tarball ──► .syft.json + .cdx.json + .spdx.json
    ├──► [trivy]      ─── lê tarball ──► .trivy.json + .trivy.cdx.json + .trivy.sarif
    ├──► [grype]      ─── lê tarball ──► .grype.json + .grype.sarif + .grype.cdx.json
    ├──► [osv]        ─── lê tarball ──► .osv.json
    ├──► [dockle]     ─── lê tarball ──► .dockle.json
    └──► [trufflehog] ─── lê tarball ──► .trufflehog.jsonl   (stdout capturado)
         │
         │   todos rodam em containers Docker isolados (DooD via docker.sock)
         │   scan_parallelism = 2–3 scanners simultâneos por worker slot
         ▼
    Adapter por scanner (Python)
         │   lê o arquivo de saída bruto
         │   normaliza para o schema Finding interno
         ▼
    Merge + dedup                         ← findings idênticos entre scanners agrupados
         │   chave de dedup: (cve_id | (package, version, scanner_category))
         │   scanner que detectou fica em Finding.scanners[]
         ▼
    report.json                           ← salvo em out/<slug>/report.json
    invocations[]                         ← metadados de cada run (status, wall_s, n_findings, severity_dist)
         │
         ▼
    Coordinator — reports table           ← INSERT OR REPLACE; n_findings, report_json, finished_at
    jobs table  — status = 'done'
         │
         ▼
    docker rmi <image>                    ← remove_image_after: true (libera disco)
    rm tarball                            ← tarball removido após scan
```

---

## Os 6 scanners — o que produzem e o que salvamos

### 1. Syft

| Campo | Valor |
|-------|-------|
| **Imagem Docker** | `anchore/syft:latest` |
| **O que faz** | Gera SBOM (Software Bill of Materials) — inventário completo de pacotes instalados na imagem |
| **Categoria dos findings** | `sbom-component` |
| **Severidade** | sempre `info` (não é vuln, é inventário) |

**Comando executado:**
```bash
syft docker-archive:/work/image.tar \
  -o cyclonedx-json=/out/<slug>.cdx.json \
  -o spdx-json=/out/<slug>.spdx.json \
  -o syft-json=/out/<slug>.syft.json
```

**Arquivos gerados:**
| Arquivo | Formato | Salvamos? |
|---------|---------|-----------|
| `<slug>.syft.json` | JSON proprietário Anchore (artifacts[]) | ✅ **parseado** pelo adapter |
| `<slug>.cdx.json` | CycloneDX JSON (SBOM padrão OWASP) | ✅ salvo no disco do worker |
| `<slug>.spdx.json` | SPDX JSON (SBOM padrão Linux Foundation) | ✅ salvo no disco do worker |

**Trecho raw (`<slug>.syft.json`):**
```json
{
  "artifacts": [
    {
      "id": "8b7e1a2c3d4f5e6a",
      "name": "@cloudron/pipework",
      "version": "2.1.2",
      "type": "npm",
      "purl": "pkg:npm/%40cloudron/pipework@2.1.2",
      "licenses": ["ISC"],
      "locations": [
        { "path": "/app/code/node_modules/@cloudron/pipework/package.json" }
      ],
      "language": "javascript",
      "cpes": ["cpe:2.3:a:cloudron:pipework:2.1.2:*:*:*:*:*:*:*"]
    }
  ],
  "schema": { "version": "15.0.0", "url": "..." },
  "distro": { "name": "debian", "version": "12" }
}
```

**Finding normalizado:**
```json
{
  "scanner": "syft",
  "category": "sbom-component",
  "severity": "info",
  "id": "pkg:npm/%40cloudron/pipework@2.1.2",
  "title": "@cloudron/pipework",
  "description": "ISC",
  "package": "@cloudron/pipework",
  "version": "2.1.2",
  "ecosystem": "npm",
  "location": "/app/code/node_modules/@cloudron/pipework/package.json"
}
```

---

### 2. Trivy

| Campo | Valor |
|-------|-------|
| **Imagem Docker** | `aquasec/trivy:latest` |
| **O que faz** | CVEs em pacotes SO + libs de linguagem + segredos + misconfigurações |
| **Categoria dos findings** | `pkg-vuln`, `secret`, `image-config` |
| **Severidade** | `critical`, `high`, `medium`, `low`, `unknown` |

**Comando executado:**
```bash
trivy --cache-dir /cache image \
  --input /work/image.tar \
  --scanners vuln,secret,misconfig,license \
  --format json --output /out/<slug>.trivy.json \
  --list-all-pkgs --quiet

# extras (não bloqueiam o parse principal):
trivy ... --format cyclonedx --output /out/<slug>.trivy.cdx.json
trivy ... --format sarif --output /out/<slug>.trivy.sarif
```

**Arquivos gerados:**
| Arquivo | Formato | Salvamos? |
|---------|---------|-----------|
| `<slug>.trivy.json` | JSON Trivy (Results[].Vulnerabilities[]) | ✅ **parseado** |
| `<slug>.trivy.cdx.json` | CycloneDX JSON | ✅ salvo |
| `<slug>.trivy.sarif` | SARIF 2.1.0 | ✅ salvo |

**Trecho raw (`<slug>.trivy.json`):**
```json
{
  "SchemaVersion": 2,
  "ArtifactName": "/work/image.tar",
  "Results": [
    {
      "Target": "ubuntu 24.04",
      "Class": "os-pkgs",
      "Type": "ubuntu",
      "Vulnerabilities": [
        {
          "VulnerabilityID": "CVE-2024-38474",
          "PkgName": "apache2",
          "InstalledVersion": "2.4.58-1ubuntu8.5",
          "FixedVersion": "2.4.58-1ubuntu8.8",
          "Severity": "MEDIUM",
          "Title": "httpd: Substitution encoding issue in mod_rewrite",
          "Description": "...",
          "CVSS": { "nvd": { "V3Score": 9.8 } },
          "References": ["https://..."]
        }
      ],
      "Secrets": [...],
      "Misconfigurations": [...]
    }
  ]
}
```

---

### 3. Grype

| Campo | Valor |
|-------|-------|
| **Imagem Docker** | `anchore/grype:latest` |
| **O que faz** | CVEs via Anchore VulnDB — foco em pacotes SO + ecosystems de linguagem (Go, npm, PyPI, etc.) |
| **Categoria dos findings** | `pkg-vuln` |
| **Severidade** | `critical`, `high`, `medium`, `low`, `info`, `unknown` |

**Comando executado:**
```bash
grype docker-archive:/work/image.tar \
  -o json=/out/<slug>.grype.json \
  -o sarif=/out/<slug>.grype.sarif \
  -o cyclonedx-json=/out/<slug>.grype.cdx.json

# env: GRYPE_DB_CACHE_DIR=/cache  (~100 MB de vuln DB cacheada)
```

**Arquivos gerados:**
| Arquivo | Formato | Salvamos? |
|---------|---------|-----------|
| `<slug>.grype.json` | JSON Grype (matches[]) | ✅ **parseado** |
| `<slug>.grype.sarif` | SARIF 2.1.0 | ✅ salvo |
| `<slug>.grype.cdx.json` | CycloneDX JSON | ✅ salvo |

**Trecho raw (`<slug>.grype.json`):**
```json
{
  "matches": [
    {
      "vulnerability": {
        "id": "CVE-2023-44487",
        "severity": "High",
        "cvss": [{ "version": "3.1", "metrics": { "baseScore": 7.5 } }],
        "fix": { "versions": ["1.20.10", "1.21.3"], "state": "fixed" },
        "description": "The HTTP/2 protocol allows a denial of service...",
        "relatedVulnerabilities": []
      },
      "artifact": {
        "name": "stdlib",
        "version": "go1.18.2",
        "type": "go-module",
        "locations": [{ "path": "/usr/local/bin/gosu" }]
      }
    }
  ],
  "source": { "type": "image", "target": { "imageID": "sha256:..." } }
}
```

**Por que Trivy + Grype juntos?** Bases de dados diferentes (NVD/OSS-Index vs Anchore VulnDB). Um pode ter CVEs que o outro não tem. Os findings duplicados são removidos no merge por `(package, version, cve_id)`.

---

### 4. OSV

| Campo | Valor |
|-------|-------|
| **Imagem Docker** | `ghcr.io/google/osv-scanner:latest` |
| **O que faz** | CVEs via banco OSV (Google) — especializado em libs de linguagem (npm, PyPI, Go, Rust, Maven, etc.) |
| **Categoria dos findings** | `pkg-vuln` |
| **Severidade** | `unknown` na maioria (OSV não inclui CVSS em todos os registros) |

**Comando executado:**
```bash
osv-scanner scan image \
  --archive --format json \
  --output-file /out/<slug>.osv.json \
  --all-packages \
  /work/image.tar

# exit code 1 quando acha vulns → tratado como sucesso se o arquivo existir
```

**Arquivos gerados:**
| Arquivo | Formato | Salvamos? |
|---------|---------|-----------|
| `<slug>.osv.json` | JSON OSV schema (results[].packages[].vulnerabilities[]) | ✅ **parseado** |

**Trecho raw (`<slug>.osv.json`):**
```json
{
  "results": [
    {
      "source": { "path": "/app/package-lock.json", "type": "lockfile" },
      "packages": [
        {
          "package": { "name": "path-to-regexp", "version": "0.1.7", "ecosystem": "npm" },
          "vulnerabilities": [
            {
              "id": "GHSA-9wv6-86v2-598j",
              "aliases": ["CVE-2024-45296"],
              "summary": "path-to-regexp outputs backtracking regular expressions",
              "details": "...",
              "severity": [{ "type": "CVSS_V3", "score": "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H" }],
              "references": [{ "type": "ADVISORY", "url": "https://github.com/..." }]
            }
          ]
        }
      ]
    }
  ]
}
```

---

### 5. Dockle

| Campo | Valor |
|-------|-------|
| **Imagem Docker** | `goodwithtech/dockle:latest` |
| **O que faz** | Boas práticas de configuração de imagem Docker (CIS Docker Benchmark) — NÃO é CVE |
| **Categoria dos findings** | `image-config` |
| **Severidade** | `high` (FATAL), `medium` (WARN), `low` (INFO) — ignora PASS/SKIP |

**Comando executado:**
```bash
dockle \
  --input /work/image.tar \
  --format json \
  --output /out/<slug>.dockle.json \
  --exit-code 0
```

**Arquivos gerados:**
| Arquivo | Formato | Salvamos? |
|---------|---------|-----------|
| `<slug>.dockle.json` | JSON Dockle (details[]) | ✅ **parseado** |

**Trecho raw (`<slug>.dockle.json`):**
```json
{
  "summary": { "fatal": 2, "warn": 1, "info": 3, "pass": 18, "skip": 0 },
  "details": [
    {
      "code": "CIS-DI-0009",
      "title": "Use COPY instead of ADD in Dockerfile",
      "level": "FATAL",
      "alerts": [
        "ADD supervisor/postgresql.conf supervisor/postgresql-service.conf /etc/supervisor/conf.d/"
      ]
    },
    {
      "code": "CIS-DI-0001",
      "title": "Create a user for the container",
      "level": "WARN",
      "alerts": ["Last user should not be root"]
    }
  ]
}
```

**Checks cobertos:** root user, segredos em ENV/ARG, ADD vs COPY, healthcheck ausente, imagem sem tag, conteúdo suspeito em layers.

---

### 6. TruffleHog

| Campo | Valor |
|-------|-------|
| **Imagem Docker** | `trufflesecurity/trufflehog:latest` |
| **O que faz** | Detecta segredos, chaves privadas e tokens hardcoded dentro da imagem |
| **Categoria dos findings** | `secret` |
| **Severidade** | `critical` (se verificado/ativo), `medium` (não verificado) |

**Comando executado:**
```bash
trufflehog docker \
  --image file:///work/image.tar \
  --json \
  --no-update
# stdout capturado como JSONL (um objeto JSON por linha)
```

**Arquivos gerados:**
| Arquivo | Formato | Salvamos? |
|---------|---------|-----------|
| `<slug>.trufflehog.jsonl` | JSONL (stdout, 1 objeto por linha) | ✅ **parseado** |

**Trecho raw (`<slug>.trufflehog.jsonl`, uma linha por finding):**
```json
{
  "DetectorName": "PrivateKey",
  "DecoderName": "BASE64",
  "Verified": false,
  "Raw": "-----BEGIN PRIVATE KEY-----\nMIIEvwIBADANBgkqhki...",
  "Redacted": "-----BEGIN PRIVATE KEY-----\nMIIEvwIBADANBgkqhki...",
  "ExtraData": null,
  "StructuredData": null,
  "SourceMetadata": {
    "Data": {
      "Docker": {
        "file": "/etc/ssl/private/ssl-cert-snakeoil.key",
        "image": "cloudron/postgresql:6.3.1",
        "layer": "sha256:3a2b..."
      }
    }
  },
  "SourceName": "trufflehog - docker",
  "SourceType": 15
}
```

**Detectores ativos:** AWS keys, GCP tokens, GitHub tokens, Slack tokens, SSH private keys, JWT, passwords em config files, certificados privados, e ~700+ outros padrões.

---

## Merge e deduplicação

Depois que todos os scanners terminam, os findings passam por uma etapa de **merge**:

```
Trivy encontrou: CVE-2023-44487 em stdlib go1.18.2
Grype encontrou: CVE-2023-44487 em stdlib go1.18.2
OSV encontrou:   CVE-2023-44487 em stdlib go1.18.2

→ merge produz UM finding com:
  {
    "id": "CVE-2023-44487",
    "scanners": ["trivy", "grype", "osv"],  ← todos que detectaram
    "severity": "high",
    "cvss": 7.5,
    ...
  }
```

**Chave de dedup:**
- Para `pkg-vuln`: `(cve_id, package, version)` se CVE disponível; senão `(package, version, title)`
- Para `secret`: `(detector_name, location, redacted_prefix)`
- Para `image-config`: `(code, location)`
- Para `sbom-component` (syft): nunca deduplica — cada pacote é único

---

## O que fica armazenado onde

```
out/<slug>/
    report.json          ← report completo: target, invocations[], findings[]
    syft/
        <slug>.syft.json         raw syft
        <slug>.cdx.json          CycloneDX
        <slug>.spdx.json         SPDX
    trivy/
        <slug>.trivy.json        raw trivy (parseado)
        <slug>.trivy.cdx.json    CycloneDX
        <slug>.trivy.sarif       SARIF
    grype/
        <slug>.grype.json        raw grype (parseado)
        <slug>.grype.sarif       SARIF
        <slug>.grype.cdx.json    CycloneDX
    osv/
        <slug>.osv.json          raw osv (parseado)
    dockle/
        <slug>.dockle.json       raw dockle (parseado)
    trufflehog/
        <slug>.trufflehog.jsonl  raw trufflehog (parseado)

Coordinator (ditector.db — SQLite):
    jobs.status = 'done'
    reports.report_json = <report.json compactado>
    reports.n_findings  = <total merged>
    reports.finished_at = <epoch>
```

**Nota**: com `remove_image_after: true`, a imagem Docker e o tarball são removidos após o scan. Os arquivos raw em `out/<slug>/` permanecem. O `report_json` na coluna SQLite é a cópia canônica usada pelo dashboard e pela API.

---

---

## Cálculo de exposure — como a fila é priorizada

### Fórmula

$$
\text{exposure}(I) = \text{pull\_count}(\text{repo}(I))
  + \sum_{N \;\in\; \text{desc}(L_I)} \;\sum_{r \;\in\; \text{images}(N)} \text{pull\_count}(\text{repo}(r))
$$

$$
\text{dependency\_weight}(I) = \sum_{N \;\in\; \text{desc}(L_I)} |\text{images}(N)|
$$

Onde:
- $L_I$ = top layer da imagem $I$ no grafo IS_BASE_OF
- $\text{desc}(L_I)$ = conjunto de todos os descendentes **estritos** de $L_I$ (exclui o próprio $L_I$)
- $\text{images}(N)$ = conjunto de refs cujo top layer é o nó $N$
- $\text{pull\_count}(\text{repo})$ = pulls históricos do repositório no Docker Hub

Em linguagem natural:

- **`pull_count`** — pulls históricos do próprio repo (Mongo `repositories_data`)
- **`downstream_pull_sum`** — soma dos `pull_count` de **todos** os repos que herdam este como base, direta ou transitivamente (grafo Neo4j IS_BASE_OF)

**Por quê**: `alpine:latest` tem 10 bi de pulls próprios, mas é a base de milhões de imagens que somam mais 91 bi de pulls downstream → exposure 101 bi. Uma CVE no alpine expõe toda essa cadeia.

### Grafo IS_BASE_OF

O Stage II constrói `(Layer)-[:IS_BASE_OF]->(Layer)` no Neo4j. Cada Layer tem **no máximo um pai** (o ID é `sha256(parent_id + sha256(layer.digest))` — determinístico), então o grafo é uma **floresta de out-trees**. Cada nó carrega `images[]` — lista das refs cujo top layer é aquele nó.

### Algoritmo — subtree sums bottom-up O(n)

```
1. Dump Mongo → repo_pull.tsv.gz (ns, name, pull_count) + tags.tsv.gz
2. Dump Neo4j → edges.tsv.gz (parent_id, child_id, 50 M linhas) + toplayers.jsonl.gz

3. Carrega edges em arrays densos indexados pelo ID interno Neo4j:
   parent[i] = pai do nó i  (-1 se raiz)
   sub_w[i]  = nº imagens no subtree (seed: len(images[i]))
   sub_p[i]  = soma pull_count no subtree (seed: sum pull_count das refs de i)

4. Kahn bottom-up (folhas primeiro):
   sub_w[pai] += sub_w[filho]
   sub_p[pai] += sub_p[filho]

5. Para cada repo:
   downstream_pull_sum = sub_p[top_layer] - self_p[top_layer]
   dependency_weight   = sub_w[top_layer] - self_w[top_layer]
   exposure            = pull_count + downstream_pull_sum
```

### Output e UPSERT

O ranker gera um JSONL com um entry por repo, ordenado por `exposure` desc:

```json
{"repository_namespace":"library","repository_name":"alpine",
 "tag_name":"latest","image_digest":"sha256:1775bebec...",
 "pull_count":10123456789,"dependency_weight":4821033,
 "downstream_pull_sum":91234567890,"exposure":101358024679}
```

O daemon `exposure-updater` (loop de 6 h) faz UPSERT na fila:

```sql
ON CONFLICT(image) DO UPDATE SET weight = excluded.weight
WHERE status = 'pending'   -- nunca sobrescreve done/running
```

Depois trim: mantém top-500k pending por exposure. Ciclo completo: ~2,5 h (dominado pelo dump dos 50 M edges do Neo4j).

---

## Schema normalizado de um Finding

Todos os scanners são normalizados para este schema antes do merge:

```json
{
  "scanner":       "trivy",
  "scanners":      ["trivy", "grype"],
  "category":      "pkg-vuln | sbom-component | secret | image-config",
  "severity":      "critical | high | medium | low | info | unknown",
  "id":            "CVE-2024-38474",
  "title":         "httpd: Substitution encoding issue in mod_rewrite",
  "description":   "...",
  "cvss":          9.8,
  "package":       "apache2",
  "version":       "2.4.58-1ubuntu8.5",
  "fixed_version": "2.4.58-1ubuntu8.8",
  "ecosystem":     "ubuntu",
  "location":      "/work/image.tar (ubuntu 24.04)",
  "cves":          ["CVE-2024-38474"],
  "references":    ["https://..."],
  "target_image":  "budibase/budibase:latest@sha256:ea1293...",
  "target_name":   "budibase_budibase_latest",
  "target_ip":     null,
  "endpoint":      ""
}
```
