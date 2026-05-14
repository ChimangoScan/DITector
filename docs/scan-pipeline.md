# Pipeline de varredura — DITector / ChimangoScan

## Visão geral em 3 estágios

```
STAGE I — Crawler (Go)                                       [COMPLETO]
  └─ Descobre repositórios via busca por prefixo na Docker Hub Search API
  └─ Mongo dockerhub_data:
       repositories_data   12.716.568 repos indexados   ← escrito pelo crawler
       crawler_keywords     2.051.801 prefixos buscados  ← estado do crawler

STAGE II — Builder (Go)                                      [EM ANDAMENTO]
  └─ Para cada repo em repositories_data:
       busca tags na Docker Hub API → escreve tags_data  (5.732.556 tags)
       busca manifests por digest   → escreve images_data (6.709.152 imagens)
       puxa a imagem, extrai layers → constrói grafo Neo4j IS_BASE_OF
  └─ Progresso (2026-05-14):  4.964.236 / 12.716.568 repos buildados (39%)
       ~50 M edges no grafo (inclui relações transitivas)
       taxa: ~207 repos/min → ETA ~30 dias para 100%

STAGE III — Scanner distribuído (ChimangoScan)               [EM ANDAMENTO]
  └─ Fila: 504.837 jobs totais rankeados por exposure
       done:    8.382 scans concluídos  (1,66%)
       pending: 488.659 aguardando
       skipped: 7.544 (pull falhou / imagem removida)
       failed:  205
       findings: 16.029.295 acumulados (merged, dedup entre scanners)
  └─ Workers distribuídos em múltiplos hosts consomem da mesma fila via HTTP
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

## Stage II — Builder e grafo IS_BASE_OF

Para cada repo em `repositories_data`, o builder Go:

1. Consulta a Docker Hub API e salva **tags** (`tags_data`) e **manifests/digests** (`images_data`)
2. Faz `docker pull` da imagem e extrai o histórico de layers (cada layer tem um digest)
3. Para cada layer, calcula `id = sha256(parent_id + sha256(layer_digest))` e insere a relação `(Layer)-[:IS_BASE_OF]->(Layer)` no Neo4j
4. Ao terminar, marca `repositories_data.graph_built_at`

O grafo resultante é uma **floresta de out-trees**: cada Layer tem no máximo um pai (o ID é determinístico dado o par parent+digest), com `~50 M edges` quando completo — inclui relações transitivas (se A é base de B que é base de C, tanto `A→B` quanto `B→C` estão no grafo).

---

## Cálculo de exposure — como a fila do Stage III é ordenada

O Stage III não escaneia imagens em ordem aleatória. A fila é rankeada por **exposure**: imagens que são base de muitas outras têm prioridade porque uma vulnerabilidade nelas afeta toda a cadeia downstream.

### Fórmula

$$
E(I) = p(R_I) + \sum_{N \,\in\, D(L_I)} \sum_{r \,\in\, \mathrm{img}(N)} p(R_r)
$$

$$
W(I) = \sum_{N \,\in\, D(L_I)} |\,\mathrm{img}(N)\,|
$$

Onde:
- $E(I)$ — exposure da imagem $I$ (métrica de prioridade)
- $p(R)$ — pulls históricos do repositório $R$ no Docker Hub
- $L_I$ — top layer de $I$ no grafo IS_BASE_OF
- $D(L_I)$ — descendentes **estritos** de $L_I$ (exclui o próprio $L_I$)
- $\mathrm{img}(N)$ — refs de imagens cujo top layer é o nó $N$
- $W(I)$ — dependency weight: nº de imagens downstream distintas

**Exemplo:** `alpine:latest` tem $p = 10$ bi de pulls próprios e 91 bi de pulls acumulados em tudo que herda dele → $E = 101$ bi → primeiro da fila.

### Algoritmo (subtree sums bottom-up, O(n))

```
1. Dump Mongo → repo_pull.tsv.gz  (ns, name, pull_count)
2. Dump Neo4j → edges.tsv.gz      (parent_id, child_id, ~50 M linhas)
               toplayers.jsonl.gz (layer_id, images[])

3. Arrays densos indexados pelo ID interno Neo4j:
   parent[i] = pai do nó i  (-1 se raiz)
   sub_p[i]  = soma pull_count no subtree   (seed: sum p(R) das refs de i)
   sub_w[i]  = nº imagens no subtree        (seed: len(images[i]))

4. Kahn bottom-up — folhas primeiro:
   sub_p[pai] += sub_p[filho]
   sub_w[pai] += sub_w[filho]

5. Para cada repo:
   downstream_pull_sum = sub_p[L] - self_p[L]
   dependency_weight   = sub_w[L] - self_w[L]
   exposure            = pull_count + downstream_pull_sum
```

O ranker gera um JSONL por repo ordenado por exposure desc e o daemon `exposure-updater` faz UPSERT na fila a cada 6 h:

```sql
ON CONFLICT(image) DO UPDATE SET weight = excluded.weight
WHERE status = 'pending'   -- nunca sobrescreve done/running
```

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

## Os 6 scanners — diferenças e complementaridade

| Scanner | O que detecta | Base de dados | Categorias | Overlap |
|---------|--------------|---------------|------------|---------|
| **Syft** | Inventário de pacotes (SBOM) — lista o que está instalado, não detecta vulns | — | `sbom-component` (info) | Único no papel |
| **Trivy** | CVEs em pacotes SO + libs + segredos em arquivos + misconfig de imagem | NVD + OSS-Index + Trivy Advisories | `pkg-vuln`, `secret`, `image-config` | Overlap alto com Grype/OSV |
| **Grype** | CVEs em pacotes SO + libs de linguagem | Anchore VulnDB (NVD + GitHub Advisory + mais) | `pkg-vuln` | Overlap alto com Trivy |
| **OSV** | CVEs em libs de linguagem (npm, PyPI, Go, Rust, Maven…) | Google OSV Database | `pkg-vuln` | Overlap parcial com Trivy/Grype |
| **Dockle** | Má configuração da imagem (CIS Docker Benchmark) — não é CVE | CIS checks internos | `image-config` | Único no papel |
| **TruffleHog** | Segredos e credenciais hardcoded na imagem (chaves, tokens, passwords) | 700+ detectores proprietários | `secret` | Parcial com Trivy |

**Por que rodar Trivy + Grype + OSV juntos se detectam CVEs?**
Cada um cobre CVEs que os outros não têm: Trivy é o mais abrangente (SO + linguagem + secrets + misconfig), Grype usa a Anchore VulnDB com fontes adicionais e às vezes severidade diferente, OSV foca em libs de linguagem com o banco do Google atualizado em tempo real. O merge posterior consolida os overlaps — se os três detectam o mesmo CVE no mesmo pacote, vira um finding único com `"scanners": ["trivy", "grype", "osv"]`.

**Os dois únicos no papel:**
- **Syft** — não detecta vuln, só SBOM. Inventaria exatamente o que está na imagem.
- **Dockle** — o único que olha para a *configuração da imagem* (root user, ADD vs COPY, healthcheck ausente, segredos em ENV) em vez do conteúdo dos pacotes.

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

## Schemas dos bancos de dados

### MongoDB — `dockerhub_data`

#### `repositories_data` (12.716.568 docs)
Populado pelo **Stage I** (crawler).

```json
{
  "_id":         "ObjectId",
  "namespace":   "library",
  "name":        "alpine",
  "pull_count":  { "high": 0, "low": 10123456789, "unsigned": false },
  "star_count":  { "high": 0, "low": 9999, "unsigned": false },
  "description": "...",
  "is_private":  false,
  "last_updated": "2026-04-01T00:00:00Z",
  "graph_built_at": "2026-05-14T03:00:00Z"   // null se Stage II ainda não processou
}
```

Índices: `{namespace,name}` (unique lookup), `{pull_count:-1}` (ranking), `{graph_built_at:1}` (Stage II progress).

---

#### `tags_data` (5.732.556 docs)
Populado pelo **Stage II** (builder), uma entrada por tag de cada repo.

```json
{
  "_id":                   "ObjectId",
  "repositories_namespace": "library",
  "repositories_name":      "alpine",
  "name":                   "latest",
  "digest":                 "sha256:1775bebec...",
  "content_type":           "image",
  "creator":                0,
  "id":                     123456,
  "last_updated":           "2026-04-01T00:00:00Z",
  "tag_status":             "active",
  "full_size":              { "high": 0, "low": 3500000, "unsigned": false },
  "images": [
    {
      "architecture": "amd64",
      "os":           "linux",
      "digest":       "sha256:abc123...",
      "size":         { "high": 0, "low": 3500000, "unsigned": false },
      "status":       "active",
      "last_pulled":  "2026-04-05T18:33:45Z",
      "last_pushed":  "2026-04-05T14:36:41Z"
    }
  ]
}
```

Índices: `{repositories_namespace, repositories_name, name}` (lookup por tag), `{repositories_namespace, repositories_name}`.

---

#### `images_data` (6.709.152 docs)
Populado pelo **Stage II**. Uma entrada por digest de imagem (por arquitetura).

```json
{
  "_id":         "ObjectId",
  "digest":      "sha256:1e70e0ad...",
  "architecture": "amd64",
  "last_pulled": "2026-04-05T18:33:45Z",
  "last_pushed": "2026-04-05T14:36:41Z",
  "layers": [
    {
      "digest":      "sha256:fd582657...",
      "size":        { "high": 0, "low": 4471206, "unsigned": false },
      "instruction": "COPY /image/ / # buildkit"
    },
    {
      "digest":      "sha256:00000000...",   // digest vazio = layer de metadata
      "size":        { "high": 0, "low": 0, "unsigned": false },
      "instruction": "USER 65532:65532"
    }
  ]
}
```

Índice: `{digest:1}` (lookup por digest).

---

#### `crawler_keywords` (2.051.801 docs)
Estado da busca por prefixo do **Stage I**.

```json
{
  "_id":        "alpine",          // o prefixo buscado
  "status":     "done",            // pending | processing | done
  "priority":   251,               // 255 - len(prefixo); maior = processado primeiro
  "crawled_at": "2026-04-03T19:17:21Z",
  "finished_at": "2026-04-03T21:48:48Z"
}
```

---

### Neo4j — grafo IS_BASE_OF

Nó `Layer` — representa uma camada de imagem Docker:

```
(:Layer {
  id:          "sha256:abc123...",   // Layer.id = sha256(parent_id + sha256(digest))
  digest:      "sha256:fd5826...",   // digest da camada em si
  size:        4471206,              // bytes
  instruction: "COPY /image/ / # buildkit",  // instrução do Dockerfile
  images:      ["docker.io/library/alpine:latest@sha256:1775...",  ...]
               // refs de imagens cujo top layer é este nó (presente só no top layer)
})
```

Relação `IS_BASE_OF`:

```
(:Layer)-[:IS_BASE_OF]->(:Layer)
// pai → filho: o filho usa o pai como camada base
// cada Layer tem no máximo UM pai (ID determinístico)
// o grafo é uma floresta de out-trees com ~50 M edges
```

---

### SQLite — `ditector.db` (fila de scan)

#### `jobs` (504.837 linhas)

```sql
CREATE TABLE jobs (
  id           INTEGER PRIMARY KEY,
  image        TEXT NOT NULL UNIQUE,   -- "ns/repo:tag@sha256:digest"
  name         TEXT NOT NULL,          -- slug filesystem-safe
  target_json  TEXT NOT NULL,          -- JSON com meta de exposure (ver abaixo)
  weight       REAL NOT NULL DEFAULT 0,-- = exposure; define a ordem da fila
  status       TEXT NOT NULL DEFAULT 'pending',  -- pending|running|done|skipped|failed
  worker_id    TEXT,                   -- "hostname/pid#slot" do worker atual
  attempts     INTEGER NOT NULL DEFAULT 0,
  error        TEXT,
  created_at   REAL NOT NULL,          -- epoch float
  started_at   REAL,
  heartbeat_at REAL,                   -- atualizado a cada ~30s pelo worker
  finished_at  REAL
);
```

`target_json` expandido:

```json
{
  "image":  "browsers/chrome:latest@sha256:0a362...",
  "name":   "browsers_chrome_latest",
  "weight": 192453.0,
  "meta": {
    "repository_namespace": "browsers",
    "repository_name":      "chrome",
    "tag_name":             "latest",
    "image_digest":         "sha256:0a362...",
    "pull_count":           189696,
    "dependency_weight":    9,
    "downstream_pull_sum":  2757,
    "exposure":             192453
  }
}
```

Índices: `idx_jobs_claim (status, weight DESC, id)`, `idx_jobs_status (status)`, `idx_jobs_heartbeat (status, heartbeat_at)`.

---

#### `reports` (8.382 linhas)

```sql
CREATE TABLE reports (
  image        TEXT PRIMARY KEY,   -- mesmo valor de jobs.image
  report_json  TEXT NOT NULL,      -- JSON completo do relatório (ver abaixo)
  n_findings   INTEGER NOT NULL DEFAULT 0,  -- total de findings merged
  finished_at  REAL NOT NULL       -- epoch float
);
```

Índices: `idx_reports_finished_at (finished_at DESC)`, `idx_reports_n_findings (n_findings)`.

`report_json` expandido:

```json
{
  "target": {
    "image":  "alpine:latest@sha256:1775...",
    "name":   "alpine_latest",
    "weight": 101239064764.0,
    "meta":   { "pull_count": 10123456789, "exposure": 101239064764, ... }
  },
  "started_at":    "2026-05-12T21:15:50Z",
  "finished_at":   "2026-05-12T21:16:03Z",
  "container_ip":  null,
  "open_ports":    [],
  "http_endpoints": [],
  "invocations": [
    {
      "scanner":      "syft",
      "status":       "ok-cached",    // ok | ok-cached | error | skipped | pull-failed | timeout
      "findings":     16,
      "findings_by_severity": { "info": 16 },
      "wall_seconds": 0.0,
      "exit_code":    0,
      "error":        "",
      "image_ref":    "anchore/syft:latest",
      "mode":         "static",
      "started_at":   "2026-05-12T21:15:53Z"
    }
    // ... um por scanner
  ],
  "findings": [ /* lista de Finding merged — schema abaixo */ ]
}
```

---

#### `exposure_state` (1 linha)
Watermark do daemon `exposure-updater`.

```sql
CREATE TABLE exposure_state (
  key        TEXT PRIMARY KEY,   -- ex: "last_run_at"
  value      TEXT NOT NULL,
  updated_at REAL NOT NULL
);
```

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
