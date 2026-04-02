# DITector - Large-Scale Security Measurement of Docker Image Ecosystem

## Project Goal
Este projeto visa realizar um **escaneamento dinâmico em larga escala** (~100.000 containers) do Docker Hub para identificar vulnerabilidades de rede usando o **OpenVAS**.

A estratégia de seleção e priorização baseia-se no framework **DITector** e no artigo *"Dr. Docker: A Large-Scale Security Measurement of Docker Image Ecosystem"*.

### Metodologia de Pesquisa
1.  **Crawling Distribuído:** Implementação de um crawler em Go altamente paralelo, capaz de utilizar múltiplos IPs (Proxy Pooling) e múltiplas contas do Docker Hub para contornar Rate Limits.
2.  **Construção do Grafo de Dependências (IDEA):** Mapeamento de camadas de imagens no Neo4j para identificar imagens "mãe" críticas (High-Dependency Weight).
3.  **Priorização:** Seleção de containers baseada em:
    *   **Pull Count:** Popularidade no Docker Hub.
    *   **Dependency Weight:** Impacto na cadeia de suprimentos (quantas imagens dependem deste container).
    *   **Network Exposure:** Filtro de containers que possuem diretivas `EXPOSE` ou configurações de rede vulneráveis.
4.  **Scan Dinâmico:** Automação do setup dos containers de rede e escaneamento via OpenVAS.

---

## 🚀 Como Executar o Crawler (Modo Pesquisa)

### 1. Preparação (Docker Hub Accounts)
Crie um arquivo `accounts.json` na raiz do projeto (não commitado):
```json
[
  { "username": "seu_user", "password": "sua_password" }
]
```

### 2. Execução Distribuída (Meet-in-the-Middle)
Divida o alfabeto entre duas ou mais máquinas para acelerar a descoberta:

**Máquina 1 (GPU1 - Letra A):**
```bash
docker run -d --name ditector_gpu1 -v $(pwd):/app -w /app --network host golang:1.22 \
go run main.go crawl --workers 50 --seed 'a' --accounts accounts.json
```

**Máquina 2 (A9 - Letra N):**
```bash
docker run -d --name ditector_a9 -v $(pwd):/app -w /app --network host golang:1.22 \
go run main.go crawl --workers 50 --seed 'n' --accounts accounts.json
```

### 3. Monitoramento dos Logs
Para ver o crawler logando e descobrindo repositórios em tempo real:
```bash
docker logs -f ditector_gpu1
```

---

## Roadmap de Desenvolvimento
- [x] Fork do repositório original.
- [x] Implementação do Parallel DFS Crawler em Go.
- [x] Integração com Auto-Login e Rotação de Contas.
- [ ] Otimização do cálculo de pesos de dependência no Neo4j.
- [ ] Script de exportação para dataset de scan do OpenVAS.

---
## Framework DITector original
O framework original foi desenvolvido pela Shanghai Jiao Tong University para detectar cinco tipos de ameaças. Este fork foca na expansão das capacidades de crawling e no escaneamento de rede dinâmico.

*Veja o [CHANGELOG.md](./CHANGELOG.md) para detalhes técnicos das mudanças.*
