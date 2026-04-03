# Arquitetura Técnica — DITector Research Fork

**Base científica:** Hequan Shi et al., "Dr. Docker: A Large-Scale Security Measurement of Docker Image Ecosystem", WWW '25, NSSL-SJTU.

---

## 1. Visão Geral da Pipeline

O upstream original (`NSSL-SJTU/DITector`) é integralmente escrito em Go e implementa os estágios II e III (construção do grafo IDEA e ranqueamento). O Estágio I — descoberta de repositórios — não existia. Este fork adiciona o Estágio I e introduz otimizações transversais nos módulos compartilhados.

```
┌─────────────────────────────────────────────────────────────────┐
│                     DITector Research Pipeline                   │
├──────────────┬──────────────────────┬───────────────────────────┤
│  Estágio I   │    Estágio II        │       Estágio III         │
│  CRAWL       │    BUILD             │       RANK                │
│  (novo)      │    (reengenhado)     │       (upstream + fixes)  │
├──────────────┼──────────────────────┼───────────────────────────┤
│ crawler/     │ buildgraph/          │ scripts/                  │
│   crawler.go │   from_mongo.go      │   calculate_node_         │
│   auth_      │ myutils/neo4j.go     │   dependent_weights.go    │
│   proxy.go   │   (reescrito)        │                           │
└──────────────┴──────────────────────┴───────────────────────────┘
         │               │                        │
         ▼               ▼                        ▼
      MongoDB          Neo4j              final_prioritized_
  (repositories_    (grafo IDEA:           dataset.json
      data)          Layer nodes +          (JSONL)
                     IS_BASE_OF edges)
```

---

## 2. Estágio I — Crawler DFS

### 2.1. Restrições da API do Docker Hub

A API de busca (`GET /v2/search/repositories/`) impõe as seguintes restrições relevantes:

- **Limite por query:** 10.000 resultados máximo, independente da cardinalidade real dos repositórios correspondentes.
- **Paginação:** máximo 100 resultados por página (até 100 páginas por keyword).
- **Stopwords do ElasticSearch:** queries de 1 caractere são tratadas como stopwords pelo motor de busca do Docker Hub, que retorna contagens artificialmente baixas (ex.: 500 em vez de >10.000). Em produção, a API aceita queries de 1 caractere mas os contagens retornadas são não confiáveis.
- **Rate limiting:** HTTP 429 para IPs ou tokens JWT com alta frequência de requisições.

O Docker Hub contém mais de 12 milhões de repositórios públicos. A combinação do limite de 10.000 resultados por query com a ausência de listagem pública exaustiva torna necessária a estratégia DFS sobre prefixos.

### 2.2. Algoritmo DFS sobre Espaço de Prefixos

Implementado recursivamente em `crawler/crawler.go`, função `crawlDFS`:

```
crawlDFS(prefix, client, token):
  if prefix ∈ crawledKeys (checkpoint em memória): return

  res = fetchPage(prefix, page=1)   // apenas para obter count

  if len(prefix) == 1 OR res.count >= 10.000:
    // Aprofundar incondicionalmente
    // (len==1 contorna stopwords; count>=10k indica espaço não esgotado)
    for char in [a-z, 0-9, -, _]:
      crawlDFS(prefix + char, client, token)
  else if res.count > 0:
    // Folha da árvore: coletar todas as páginas
    for page in [1 .. ceil(res.count / 100)]:
      results = fetchPage(prefix, page)
      processResults(results)   // enfileirar em RepoChan

  MarkKeywordCrawled(prefix)   // checkpoint post-order no MongoDB
```

**Aprofundamento em prefixos de 1 caractere:** o aprofundamento é forçado independente da contagem reportada pelo servidor. Isto garante que prefixos como `a`, `t`, `p` sempre gerem sub-prefixos `aa`, `ab`, ... até que a contagem real seja capturável pela paginação.

**Checkpointing post-order:** `MarkKeywordCrawled` é gravado somente após todos os filhos DFS terminarem. Em caso de interrupção, o prefixo é retomado do início — sem perda de sub-árvores não concluídas.

**Alfabeto de expansão:** `[a-z, 0-9, -, _]` — 38 caracteres, produzindo fator de ramificação 38 a cada aprofundamento.

### 2.3. Concorrência e Pipeline de Escrita

`Start(seeds)` distribui seeds iniciais entre N workers via channel temporário de tamanho `len(seeds)`. Cada worker executa `crawlDFS` recursivamente de forma independente — sem coordenação de prefixos entre workers durante a recursão:

```
Start(seeds):
  seedChan = make(chan string, len(seeds))
  for s in seeds: seedChan <- s
  close(seedChan)

  for i in [0, WorkerCount):
    go worker(i, seedChan)   // cada worker drena seedChan e executa DFS completo

  WG.Wait()
  close(RepoChan)
  <-writerDone
```

Repositórios descobertos são inseridos em `RepoChan` (buffer 100.000) sem bloqueio. A goroutine `repoWriter` consome o canal de forma assíncrona:

```
repoWriter:
  buffer = []
  ticker = 2s

  loop:
    case repo <- RepoChan:
      buffer.append(repo)
      if len(buffer) >= 1000: BulkUpsert(buffer); buffer = []
    case ticker fires:
      BulkUpsert(buffer); buffer = []
    case RepoChan closed:
      BulkUpsert(buffer); return
```

`BulkUpsert` executa `BulkWrite` não-ordenado com upsert por `{namespace, name}`, garantindo deduplicação persistente entre execuções. Deduplicação intra-execução é feita em memória via `seenRepos sync.Map` — repositórios já vistos nesta execução são descartados em O(1) antes de qualquer I/O.

### 2.4. Tratamento de Rate Limiting (HTTP 429)

```go
// crawler/crawler.go — fetchPage
if resp.StatusCode == 429 {
    time.Sleep(10 * time.Second)
    newClient, newToken := pc.IM.GetNextClient()
    return pc.fetchPage(query, page, newClient, newToken)
}
```

O mecanismo é: aguardar 10s, rotacionar para a próxima identidade disponível via `IdentityManager.GetNextClient()`, e retentar a mesma requisição recursivamente com a nova identidade. A página não é descartada.

O delay de 200ms entre páginas de uma mesma keyword (loop de scrapeAllPages) é aplicado preventivamente para evitar atingir o rate limit com frequência.

### 2.5. Gestão de Identidades — `crawler/auth_proxy.go`

`IdentityManager` centraliza autenticação e proxies:

- Carrega contas de `accounts.json` (`[{username, password}]`)
- Carrega proxies de arquivo texto (uma URL por linha)
- Auto-login JWT via `POST /v2/users/login/` com `sync.Mutex` (evita login paralelo da mesma conta)
- `GetNextClient()` retorna `(*http.Client, token)` com proxy rotacionado round-robin

**Limitação:** não há refresh automático de JWT. Tokens Docker Hub expiram em ~24h. Reiniciar o crawler renova os tokens; o checkpoint de keywords garante retomada sem perda de progresso.

### 2.6. Distribuição Multi-Nó

O comando `crawl` suporta dois modos de particionamento:

| Modo | Flags | Comportamento |
|------|-------|--------------|
| Shard automático | `--shard N --shards M` | Divide o alfabeto igualmente entre M shards; shard N processa a fração correspondente. Implementado em `crawler.ShardSeeds(shard, total)` |
| Seeds manuais | `--seed a,b,c` | Seeds explícitas separadas por vírgula |
| Alfabeto completo | (nenhuma flag) | Semeia todo o alfabeto `[a-z, 0-9, -, _]` |

Variáveis de ambiente `MONGO_URI` e `NEO4J_URI` permitem que nós remotos apontem para o banco do nó principal sem alterar `config.yaml`.

### 2.7. Resultados Empíricos (Produção)

Configuração validada: Node 1 (shard 0/2, 3 workers) + Node 2 (shard 1/2, 4 workers), 7 contas Docker Hub, MongoDB no Node 1, conexão remota Node 2 → Node 1.

| Métrica | Valor |
|---------|-------|
| Repositórios únicos em <10 min | >100.000 |
| Repositórios únicos acumulados | >750.000 (751.149 verificado) |
| Throughput sustentado | ~78.800 repos únicos/minuto |
| Projeção 24h (extrapolação linear) | ~11,3 milhões de repositórios/dia |
| Duplicatas no banco | 0 (índice único MongoDB `{namespace, name}`) |

O throughput é limitado pelo rate limit da API do Docker Hub, não pelo hardware ou pela implementação.

---

## 3. Estágio II — BuildGraph

### 3.1. Pipeline de Três Estágios com Buffered Channels

`buildgraph/from_mongo.go` implementa um pipeline produtor-consumidor que desacopla os três gargalos físicos distintos: leitura de banco de dados (MongoDB), I/O de rede (API Docker Hub) e escrita de banco de dados (Neo4j).

```
MongoDB
  { pull_count >= threshold, graph_built_at: {$exists: false} }
    │
    ▼ goroutine Loader (única, leitura paginada)
    │
repoChan (buffer 4.000)
    │
    ▼ repoWorkers × max(NumCPU × 16, 64)       [I/O bound — espera HTTPS]
    │   1. isNetworkContainer(name) → descartar se falso
    │   2. GET tags da API Docker Hub
    │   3. GET manifests por tag (semáforo tagConcurrency=4 por repo)
    │   4. descartar imagens Windows
    │
jobChan (buffer 20.000)
    │
    ▼ buildGraphWorkers × max(NumCPU × 4, 16)  [DB bound — Bolt/TCP → Neo4j]
    │   1. SHA256 chain de IDs (local, CPU)
    │   2. InsertImageToNeo4j (transação única)
    │   3. MarkRepoGraphBuilt (MongoDB → graph_built_at)
    │
Neo4j (Layer nodes + IS_BASE_OF edges + IS_SAME_AS → RawLayer nodes)
```

**Dimensionamento dos workers:**
- `repoWorkers`: `max(NumCPU × 16, 64)` — fator 16 justificado pelo modelo de I/O: goroutines aguardam respostas HTTPS em estado de sleep sem consumir CPU. Mínimo absoluto de 64 garante paralelismo em máquinas com poucos núcleos.
- `buildGraphWorkers`: `max(NumCPU × 4, 16)` — escrita Neo4j via Bolt é menos paralelizável; excesso de conexões simultâneas degrada o throughput do banco. O fator 4 equilibra paralelismo com estabilidade.

### 3.2. Algoritmo IDEA — Hashing de Layer IDs

O algoritmo definido no paper Dr. Docker (Seção 3.2) é implementado em `myutils/neo4j.go`, função `InsertImageToNeo4j`:

**Content layer** (possui digest SHA256 do arquivo tar):
```
dig_i      = SHA256(layer_i.digest)
Layer_i.id = SHA256(Layer_{i-1}.id || dig_i)
```

**Config layer** (instrução Dockerfile sem conteúdo físico, ex.: `ENV`, `CMD`):
```
dig_i      = SHA256(layer_i.instruction)
Layer_i.id = SHA256(Layer_{i-1}.id || dig_i)
```

**Bottom layer** (i=0): usa `preID = ""` como valor anterior à concatenação.

**Propriedade fundamental:** duas imagens que compartilham as mesmas N primeiras layers na mesma ordem produzem `Layer_N.id` idênticos. Relações de herança são identificáveis por igualdade de ID — sem análise de conteúdo das layers.

### 3.3. Transação Única por Imagem no Neo4j

A implementação original do upstream executava uma transação Neo4j separada por layer (O(N) round-trips por imagem). O fork reescreve `InsertImageToNeo4j`:

```
// Fase 1 — local, sem I/O de rede:
records = []layerRecord{}
preID = ""
for each layer_i in image.Layers:
    dig_i  = SHA256(layer_i.digest or layer_i.instruction)
    currID = SHA256(preID + dig_i)
    records.append({prevID: preID, currID: currID, layer: layer_i})
    preID = currID

// Fase 2 — uma única transação:
session.ExecuteWrite(func(tx):
    for each record in records:
        tx.Run(MERGE (l:Layer {id: record.currID}) ...)
        tx.Run(MERGE (l)-[:IS_BASE_OF]->(next) ...)
        tx.Run(MERGE (rl:RawLayer {digest: ...})-[:IS_SAME_AS]-(l) ...)
    tx.Run(SET last_layer.images += [imgName])
)
```

**Complexidade de rede:**
- Anterior (upstream): O(N) round-trips por imagem, N ∈ [5, 30] tipicamente
- Atual: O(1) round-trips por imagem, independente de N

Com latência típica de Bolt/TCP de ~5–10ms por round-trip, uma imagem com 20 layers passa de ~100–200ms para ~5–10ms de custo de rede de inserção.

### 3.4. Filtro Heurístico de Rede

A função `isNetworkContainer(name string)` verifica presença de keywords de serviços de rede no nome do repositório (comparação case-insensitive):

```
nginx, apache, http, https, server, web, api, rest, grpc,
db, database, mysql, postgres, sql, redis, mongo, elastic,
kafka, rabbitmq, proxy, gateway, lb, balancer, vpn, ssh,
ftp, smtp, imap, ldap, app, service, svc
```

Repositórios que não passam neste filtro são descartados antes de qualquer chamada à API do Docker Hub.

**Limitação:** o campo `EXPOSE` do Dockerfile — indicador mais preciso de exposição de rede — só é acessível após download completo da imagem, o que é inviável como pré-filtro nesta escala. O filtro por nome é uma aproximação conservadora: containers com nomes não descritivos mas que expõem serviços de rede não serão incluídos.

### 3.5. Checkpointing do Estágio II

Após processar todas as tags de um repositório com sucesso, `repoWorker` chama `MarkRepoGraphBuilt`, que grava `graph_built_at: <timestamp RFC3339>` no documento MongoDB. O Loader filtra por `{graph_built_at: {$exists: false}}`.

Em caso de interrupção: repositórios com `graph_built_at` gravado são ignorados; repositórios parcialmente processados são reprocessados integralmente — seguro pela idempotência dos `MERGE` no Neo4j.

### 3.6. Estrutura do Grafo Neo4j

| Tipo | Propriedades | Semântica |
|------|-------------|-----------|
| `Layer` | `id` (SHA256 chain), `digest`, `images[]`, `size`, `instruction` | Posição na cadeia de herança |
| `RawLayer` | `digest` | Conteúdo físico da layer |
| `[:IS_BASE_OF]` | — | Layer antecessora → Layer sucessora |
| `[:IS_SAME_AS]` | — | Layer ↔ RawLayer (posição ao conteúdo) |

Todas as inserções usam `MERGE` (não `CREATE`), garantindo idempotência.

---

## 4. Modificações Transversais no Upstream

### 4.1. `myutils/mongo.go`

| Adição | Descrição |
|--------|-----------|
| `BulkUpsertRepositories(repos)` | Bulk write não-ordenado; ~10–50× mais rápido que upserts individuais para processar uma página de resultados |
| `KeywordsColl` | Coleção `crawler_keywords` para checkpoint do Estágio I |
| `IsKeywordCrawled(kw)` / `MarkKeywordCrawled(kw)` | Leitura/gravação do checkpoint de keywords |
| `MarkRepoGraphBuilt(ns, name)` | Grava `graph_built_at` no repositório (checkpoint Stage II) |
| Connection pool | `MaxPoolSize=100`, `MinPoolSize=5`, `MaxConnIdleTime=5min` |
| Timeout do ping inicial | `1s → 30s` (evita falso-negativo em conexões lentas) |

### 4.2. `myutils/docker_hub_api_requests.go`

| Parâmetro | Antes | Depois | Justificativa |
|-----------|-------|--------|---------------|
| `DisableKeepAlives` | `true` | removido (false) | Reutilização de conexões TCP/TLS; economia de ~100–300ms por requisição |
| `MaxIdleConns` | — | 300 | Pool de conexões para alta concorrência |
| `MaxIdleConnsPerHost` | — | 50 | Limita conexões ociosas por host |
| `IdleConnTimeout` | — | 90s | Descarte de conexões ociosas |
| `Timeout` | — | 30s | Timeout global por requisição |

### 4.3. `myutils/config.go`

| Modificação | Descrição |
|-------------|-----------|
| `MONGO_URI` / `NEO4J_URI` env vars | Sobrescrevem `config.yaml` — permite nós remotos apontarem para o banco central sem alterar configuração local |
| `os.Getwd()` em vez de `filepath.Dir(os.Args[0])` | Config buscado relativo ao CWD (compatível com `go run` e binários compilados) |
| Neo4j opcional na inicialização | Falha de conexão Neo4j não aborta o processo — útil para Estágio I sem Neo4j ativo |

### 4.4. `myutils/neo4j.go` — Correção de Bug Crítico

**`findLayerNodesByRawLayerDigestFunc`:** a query Cypher original usava `{id: $digest}` para matchar um nó `RawLayer`, mas nós `RawLayer` são criados com a propriedade `digest`. A propriedade `id` não existe em `RawLayer`. A query nunca retornava resultados, quebrando silenciosamente toda a funcionalidade de rastreamento de imagens upstream.

```cypher
-- Antes (upstream, incorreto):
MATCH (l:Layer)-[:IS_SAME_AS]-(rl:RawLayer {id: $digest})

-- Depois (correto):
MATCH (l:Layer)-[:IS_SAME_AS]-(rl:RawLayer {digest: $digest})
```

### 4.5. `myutils/urls.go`

Adicionados `V2SearchURLTemplate` e `GetV2SearchURL`:

```go
V2SearchURLTemplate = `https://hub.docker.com/v2/search/repositories/?query=%s&page=%d&page_size=%d&ordering=-pull_count`
```

O parâmetro `ordering=-pull_count` garante resultados determinísticos e ordenados por popularidade — necessário para consistência entre páginas durante o scraping.

### 4.6. `scripts/calculate_node_dependent_weights.go` — Correção de Bug

O branch `if repoDoc.Namespace == "library"` continha `continue` como primeira instrução, tornando todo o código subsequente (`FindAllTagsByRepoName`, etc.) inalcançável. Imagens oficiais Docker (namespace `library`) eram silenciosamente ignoradas no cálculo de dependency weight.

Correção: `continue` removido. Imagens `library` agora passam pelo mesmo processamento das imagens community.

---

## 5. Limitações Conhecidas

1. **Expiração de JWT:** tokens Docker Hub expiram em ~24h; não há refresh automático. O crawler deve ser reiniciado periodicamente. O checkpoint de keywords garante retomada sem perda de progresso.

2. **Build com API live:** se um repositório for deletado entre o Estágio I e o Estágio II, erros são logados mas não interrompem o processamento.

3. **Cobertura do espaço de busca:** o DFS sobre prefixos `[a-z0-9-_]` não garante cobertura de repositórios com nomes compostos exclusivamente por outros caracteres (ex.: Unicode). Cobertura prática para nomenclaturas descritivas é alta, mas não foi quantificada formalmente.

4. **Throughput Neo4j:** uma transação por imagem (O(1) round-trips). Para volumes >1M imagens, o gargalo migra para a memória heap do Neo4j — aumentar `NEO4J_dbms_memory_heap_max__size` é recomendado.
