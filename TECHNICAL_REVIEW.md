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
O sistema foi projetado sob o princípio da **Idempotência de Escrita**. Toda descoberta é processada via `UpdateRepository` no MongoDB com a flag `upsert: true`.
*   **Recuperação de Falhas:** O estado do banco de dados permanece íntegro após interrupções. A retomada da varredura é coordenada via flag `--seed`, permitindo que o pesquisador divida o espaço de busca alfabética de forma arbitrária entre instâncias independentes.

## 4. Metodologia de Construção do Grafo (Layers)
Embora a fase de descoberta foque em metadados de catálogo, o framework está preparado para o estágio de extração de dependências:
1.  **Mapeamento de Tags:** Utilização do método `ReqTagsMetadata` para listar todas as versões de uma imagem.
2.  **Extração de Camadas:** O módulo `ReqImagesMetadata` recupera a lista de SHAs das camadas que compõem cada imagem.
3.  **Relacionamento em Grafo:** Estes SHAs são utilizados como nós no **Neo4j**, onde uma aresta de dependência é criada sempre que uma imagem compartilha a mesma pilha de camadas de uma imagem base, permitindo a análise de propagação de vulnerabilidades na cadeia de suprimentos.

## 5. Conclusão
A arquitetura atual do fork `DITector` evoluiu de um script de análise estática para um sistema de coleta de dados distribuído e resiliente. A conformidade com os princípios SOLID e o tratamento rigoroso de casos de borda (Rate Limits, Expiração de Sessão) tornam a ferramenta apta para a condução de pesquisas de segurança em larga escala.
