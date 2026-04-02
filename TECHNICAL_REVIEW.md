# Relatório de Evolução Arquitetural: DITector Research Fork

## 1. Introdução
Este documento detalha as modificações realizadas no framework DITector para viabilizar a coleta e análise de dados do ecossistema Docker Hub em escala industrial. A reengenharia focou em substituir componentes legados por uma arquitetura concorrente em Go, priorizando performance, resiliência a limites de taxa (rate limiting) e distribuição de carga.

## 2. Registro Cronológico de Modificações (Commit-by-Commit)

### 2.1. Implementação do Motor de Descoberta Paralelo
**Commit:** `feat: implement parallel DFS crawler base`
*   **Descrição:** Introdução de um crawler nativo em Go utilizando o padrão *Worker Pool*.
*   **Racional Técnico:** A utilização de *goroutines* permite a execução de centenas de requisições I/O-bound simultâneas. Foi aplicado o algoritmo de Busca em Profundidade (DFS) sobre o espaço de prefixos alfabéticos para contornar a limitação de 10.000 registros por consulta da API do Docker Hub.

### 2.2. Gestão de Identidades e Rotação de Proxies
**Commit:** `feat: add support for proxy pooling and multiple accounts rotation`
*   **Descrição:** Criação do módulo `IdentityManager` para gerenciar pools de endereços IP e credenciais.
*   **Racional Técnico:** Para mitigar o erro HTTP 429 (Too Many Requests), o sistema rotaciona identidades e proxies, distribuindo a carga de requisições e aumentando o limite efetivo de coleta por unidade de tempo.

### 2.3. Autenticação Automática e Renovação de Sessão (JWT)
**Commit:** `feat: implement auto-login, token rotation and seed flag`
*   **Descrição:** Implementação de lógica para obtenção e renovação automática de tokens JWT do Docker Hub.
*   **Racional Técnico:** Garante a autonomia do sistema em execuções de longa duração, eliminando a necessidade de intervenção manual para atualização de credenciais expiradas.

### 2.4. Resiliência e Sincronização de Concorrência
**Commit:** `fix: synchronize login process and add detailed error logging`
*   **Descrição:** Aplicação de primitivas de sincronização (`sync.Mutex`) no processo de autenticação.
*   **Racional Técnico:** Evita condições de corrida (*race conditions*) onde múltiplos workers tentam realizar login simultaneamente, o que poderia resultar em bloqueios temporários da conta por comportamento anômalo.

### 2.5. Transição para API Oficial V2
**Commit:** `feat: switch to official V2 search API for better crawler reliability`
*   **Descrição:** Migração do endpoint de busca para a API oficial de registro (`/v2/search/repositories`).
*   **Racional Técnico:** A API V2 oferece maior estabilidade e consistência nos metadados retornados em comparação com endpoints de interface web, garantindo a integridade dos campos `pull_count` e identificadores de repositório.

## 3. Análise de Persistência e Idempotência
...

## 4. Análise Empírica de Performance
Durante a fase de testes, foram observadas as seguintes métricas de desempenho:
*   **Taxa de Descoberta (Crawl):** ~15 a 20 repositórios/segundo. O gargalo é puramente o Rate Limit da API de Busca.
*   **Taxa de Extração de Dependências (Build):** ~0.3 repositórios/segundo. Esta fase é significativamente mais onerosa, pois exige múltiplas chamadas por repositório (Tags -> Manifests -> Layers).

### Estimativa de Tempo (ETL)
Para uma amostra de 100.000 repositórios, estima-se um tempo de processamento de aproximadamente 92 horas em uma única instância. Para o ecossistema total (12M repositórios), o tempo projetado excede 1 ano, tornando a extração exaustiva inviável sem uma frota massiva de instâncias.

## 5. Estratégia de Otimização e Priorização (Filtro de Rede)
Para viabilizar o escaneamento via OpenVAS em tempo hábil, a metodologia de extração será alterada de exaustiva para seletiva:

1.  **Priorização por Popularidade:** A fase de `Build` será restrita a repositórios com `pull_count` superior a um limite estatístico (ex: > 10.000 pulls).
2.  **Identificação de Serviços de Rede:** Aplicação de filtros heurísticos no MongoDB para identificar containers que operam como servidores (keywords: *server, api, http, sql*).
3.  **Mapeamento de Layers Críticas:** O foco do Neo4j será identificar imagens base (como `debian`, `alpine`, `node`) que possuem o maior "Dependency Weight", permitindo que o scan dinâmico seja direcionado aos nós mais influentes da cadeia de suprimentos.

## 6. Conclusão
...
