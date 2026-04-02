# Changelog - DITector Research Fork

## [1.1.0] - 2024-04-02
### Added
- **Parallel Go Crawler**: New core crawler implemented in Golang for high-performance metadata extraction.
- **DFS Search Strategy**: Implemented Depth-First Search keyword generation to bypass Docker Hub's 10,000 results API limit.
- **Auto-Login System**: Automatic JWT token generation and rotation using Docker Hub credentials from `accounts.json`.
- **Identity Manager**: Support for Proxy Pooling (multi-IP) and Account Rotation to mitigate rate limits (HTTP 429).
- **Meet-in-the-Middle Scaling**: Added `--seed` flag to allow distributed crawling across multiple machines (e.g., Machine A starts at 'a', Machine B at 'n').
- **Dockerized Execution**: Support for running the crawler inside a Go container to avoid local environment pollution.

### Changed
- **CLI Integration**: Updated `docker-scan crawl` command to support new parallel architecture.
- **Data Persistence**: Optimized MongoDB `Upsert` operations for concurrent writes from multiple workers/machines.

### Security
- **Credential Protection**: Implemented dynamic credential loading from `accounts.json` (untracked by git) to ensure no secrets are leaked in the repository.

---
*Este fork transforma o DITector em uma ferramenta de descoberta de containers em escala industrial (~12M+ repos).*
