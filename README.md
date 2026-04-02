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

## Roadmap de Desenvolvimento
- [x] Fork do repositório original.
- [ ] Implementação do Parallel/Distributed Crawler em Go.
- [ ] Integração com Proxy Pooling e Rotação de Contas.
- [ ] Otimização do cálculo de pesos de dependência no Neo4j.
- [ ] Script de exportação para dataset de scan do OpenVAS.

## Framework DITector original
O framework original foi desenvolvido pela Shanghai Jiao Tong University para detectar cinco tipos de ameaças:
- Parâmetros de comando sensíveis
- Vazamento de segredos (Secrets)
- Vulnerabilidades de software (SCA)
- Má configuração (Misconfigurations)
- Arquivos maliciosos

---
*Este fork foca na expansão das capacidades de crawling e no escaneamento de rede dinâmico.*
