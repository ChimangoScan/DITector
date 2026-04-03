# Changelog — DITector Research Fork

---

## [2.0.0] — 2026-04-03

### Adicionado

**`crawler/` (implementação do Estágio I — stub sem `Run` no upstream):**
- `crawler/crawler.go`: `ParallelCrawler` com DFS recursivo sobre espaço de prefixos do Docker Hub. N workers independentes, cada um executando `crawlDFS` recursivamente a partir de suas seeds. Deduplicação em memória via `seenRepos sync.Map`. Aprofundamento forçado para prefixos de 1 caractere. Goroutine `repoWriter` com `BulkWrite` ao MongoDB a cada 2s ou 1.000 repositórios. Checkpointing post-order por keyword via coleção `crawler_keywords`.
- `crawler/auth_proxy.go`: `IdentityManager` — carrega contas Docker Hub de `accounts.json` e proxies de arquivo texto; auto-login JWT com `sync.Mutex`; rotação round-robin de identidades via `GetNextClient()`.

**`buildgraph/from_mongo.go` (Estágio II reengenhado):**
- Pipeline de três estágios desacoplados por buffered channels: Loader (MongoDB → `repoChan` buf 4.000), repoWorkers (`max(NumCPU×16, 64)`) e buildGraphWorkers (`max(NumCPU×4, 16)`).
- Semáforo `tagConcurrency=4` por repositório para controle de requisições paralelas de manifest.
- Checkpointing `graph_built_at` no MongoDB após conclusão de cada repositório.

**`myutils/mongo.go`:**
- `BulkUpsertRepositories`: bulk write não-ordenado com upsert por `{namespace, name}`.
- `KeywordsColl`, `IsKeywordCrawled`, `MarkKeywordCrawled`: sistema de checkpoint para o Estágio I.
- `MarkRepoGraphBuilt`: checkpoint para o Estágio II.
- Connection pool: `MaxPoolSize=100`, `MinPoolSize=5`, `MaxConnIdleTime=5min`.
- Timeout do ping inicial: `1s → 30s`.

**`myutils/neo4j.go`:**
- `InsertImageToNeo4j` reescrito: IDs de layer pré-computados localmente via SHA256 chain; toda a cadeia de layers inserida em uma única transação `ExecuteWrite` (O(1) round-trips por imagem, em vez de O(N layers)).

**`myutils/urls.go`:**
- `V2SearchURLTemplate` e `GetV2SearchURL` — API V2 do Docker Hub com `ordering=-pull_count`.

**`myutils/config.go`:**
- Override por variáveis de ambiente `MONGO_URI` e `NEO4J_URI`.
- Localização do config por `os.Getwd()` (compatível com `go run`).
- Neo4j opcional na inicialização.

**`myutils/docker_hub_api_requests.go`:**
- Keep-alives habilitados (conexões TCP/TLS reutilizadas entre requisições).
- Connection pool: `MaxIdleConns=300`, `MaxIdleConnsPerHost=50`, `IdleConnTimeout=90s`, `Timeout=30s`.

**`cmd/cmd.go`:**
- Subcomando `crawl` com flags `--workers`, `--seed`, `--shard`, `--shards`, `--accounts`, `--proxies`, `--config`.

**Infraestrutura:**
- `docker-compose.yml`: MongoDB, Neo4j, crawler.
- `docker-compose.node2.yml`: Node 2 apontando para MongoDB do Node 1.
- `automation/pipeline_autopilot.sh`: orquestração sequencial dos 3 estágios.
- `automation/test_e2e.sh`: teste de integração end-to-end.

### Corrigido

- **`myutils/neo4j.go` — `findLayerNodesByRawLayerDigestFunc` (crítico):** query Cypher usava `{id: $digest}` para matchar nó `RawLayer`, mas a propriedade armazenada é `digest`. A propriedade `id` não existe em `RawLayer`. A query nunca retornava resultados, quebrando silenciosamente o rastreamento de imagens upstream. Corrigido para `{digest: $digest}`.

- **`scripts/calculate_node_dependent_weights.go` (médio):** branch `if repoDoc.Namespace == "library"` continha `continue` como primeira instrução, tornando todo o código subsequente inalcançável. Imagens oficiais Docker eram ignoradas no cálculo de dependency weight. `continue` removido.

- **`myutils/docker_hub_api_requests.go` (médio):** `DisableKeepAlives: true` impedia reutilização de conexões TCP, acrescentando ~100–300ms de handshake+TLS por requisição desnecessariamente. Removido.

- **`automation/test_e2e.sh` (baixo):** sintaxe `[ [` inválida em bash substituída por `[ "$(expr)" -gt N ]`.

---

## [1.0.0] — baseline upstream (NSSL-SJTU/DITector)

Pipeline original com subcomando `crawl` declarado em `cmd/cmd.go` sem campo `Run` (stub sem implementação). Estágios II e III funcionais. Implementa: `buildgraph/build.go` (inserção síncrona por layer no Neo4j), `myutils/`, `scripts/`, `analyzer/`, `cmd/`, `main.go`.
