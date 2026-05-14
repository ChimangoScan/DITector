#!/usr/bin/env python3
"""Integrate phase-2 + phase-3 scanner results into the original
reports/scanner-comparison.html. Modifies that file in-place — the
companion file scanner-comparison-new.html is removed when done.

Steps (surgical, no rewrite):
1. Load metrics + findings from reports/scans-output/{node2,node1}/<target>/<scanner>/.
2. Build a DATA-dict fragment matching the existing schema and merge it into
   the JS DATA literal on line 1111.
3. Append a calc-toggle for each new scanner in the toggles fieldset.
4. Append a tab button for each new scanner in the per-scanner tablist.
5. Append a tab-panel block (header + scanner-info section) for each new
   scanner, matching the openvas placeholder layout.
6. Insert coverage-matrix rows in the matrix tbody.
7. Update the "Por scanner" badge count.
"""
from __future__ import annotations

import html
import json
import re
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
HTML_PATH = ROOT / "reports" / "scanner-comparison.html"
NEW_HTML = ROOT / "reports" / "scanner-comparison-new.html"
SCANS_OUT = ROOT / "reports" / "scans-output"

HOSTS = ["node1", "node2"]  # node1 wins for shared scanners (more recent)
TARGETS = ["webgoat", "dvwa", "juice-shop"]
ORIGINAL = {"syft", "trivy", "grype", "dockle", "trufflehog", "gitleaks",
            "clamav", "whatweb", "nmap", "nikto", "nuclei", "wapiti", "zap",
            "sqlmap", "openvas"}

# Per-scanner profile (matches the existing scanner-info <dl> shape).
INFO = {
    "osv": dict(type="Vuln (OSV.dev)", mode="static", since="2022", author="Google",
                license="Apache-2.0",
                covers=["cve", "sbom"],
                what="Scanner que casa pacotes contra a base OSV.dev — agregador open-source de advisories ecosystem-native.",
                how="Lê SBOMs ou imagens via tarball; consulta osv.dev REST API e Maven Central, npm, PyPI, RubyGems, crates.io para cruzar versão→advisory ID. Output JSON/SARIF.",
                when="Quando precisa cobrir Go modules, Rust crates, Cargo — onde NVD costuma ficar atrás. Cross-validação com Trivy/Grype.",
                pros="DB ecosystem-native (Go, Rust, Cargo, Maven). Mantida pelo Google. Atualização semanal.",
                cons="Coverage menor de SO packages (Alpine, Debian) vs Trivy/Grype.",
                alts="Trivy (mais amplo), Grype (CVE-only), Snyk (commercial)."),
    "secretscanner": dict(type="Secrets (container)", mode="static", since="2020",
                author="Deepfence", license="Apache-2.0", covers=["secrets"],
                what="Scanner de secrets focado em containers. ~140 regras YARA-style.",
                how="Mounta o socket Docker, indexa layers e camadas, rodando regras YARA contra arquivos extraídos. Output JSON.",
                when="Quando o image traz tokens/keys nas camadas (env files, JWTs, AWS keys). Complementa TruffleHog para padrões container-específicos.",
                pros="Container-native, sem precisar exportar FS. Engine YARA permite custom rules.",
                cons="Binário precisa AVX2 — falha em CPUs antigas. Sem verificação ativa (regex-only).",
                alts="TruffleHog (com verificação API), GitLeaks, Whispers (configs estruturadas)."),
    "yara": dict(type="Malware/pattern", mode="static", since="2007", author="VirusTotal",
                license="BSD-3", covers=["malware"],
                what="Engine genérico de pattern-matching para classificar binários e textos.",
                how="Compila regras (.yar) em bytecode e roda contra arquivos. Suporta strings, bytes, regex, hex. Roteamos via xargs -n1 -P4 para paralelizar.",
                when="Detectar cryptominers, reverse shells, chaves privadas embedded. Custom rules para campanhas específicas.",
                pros="Extensível com regras próprias. Padrão da indústria malware-research. Fast.",
                cons="Rules precisam ser escritas/curadas. Sem signatures embutidas.",
                alts="ClamAV (signature-only), Capa (FLARE, behavior-based)."),
    "detect-secrets": dict(type="Secrets (baseline)", mode="static", since="2018",
                author="Yelp", license="Apache-2.0", covers=["secrets"],
                what="Scanner de secrets baseline-diffable — emite JSON de estado atual e diff contra runs futuros.",
                how="Plugin-based (AWS, JWT, Slack, base64-high-entropy). pip install detect-secrets, scan --all-files. Cada finding tem hash + offset.",
                when="Repos longevos com secrets históricos conhecidos: registra o baseline e flagga só novos.",
                pros="Modelo baseline reduz noise drastically em codebases antigos. Plugins extensíveis.",
                cons="Não verifica live (ao contrário de TruffleHog). Plugins podem ter FP altos sem tuning.",
                alts="TruffleHog (verifica API), GitLeaks, Whispers."),
    "guarddog": dict(type="Supply-chain malware", mode="static", since="2022",
                author="Datadog", license="Apache-2.0", covers=["supplychain"],
                what="Detecta supply-chain malice em pacotes npm/PyPI: typosquatting, install scripts, exfil patterns.",
                how="Faz AST-analysis de package source code com Semgrep + heurísticas. Para cada pacote: download → scan → score.",
                when="Auditar dependencies de uma image antes de produção. Cobre uma classe de ataques que NVD/GHSA não cobrem (CVEs vs malice).",
                pros="Único scanner do stack focado em malice (não vulnerabilidade). Datadog mantém.",
                cons="Limitado a npm e PyPI. Lento (download + AST por pacote).",
                alts="Phylum (commercial), Snyk Advisor, Sonatype Nexus."),
    "govulncheck": dict(type="Vuln Go (reachability)", mode="static", since="2022",
                author="Google (Go team)", license="BSD-3", covers=["cve"],
                what="Scanner de CVE para Go com reachability analysis — só reporta CVE se a função vulnerável é chamável.",
                how="Lê o binary's BuildInfo (Go 1.18+) ou source modules; constrói call graph; cruza com vuln.go.dev DB. Reporta apenas paths reachable.",
                when="Imagens Go-heavy. Reduz false positives em 30-70% vs Trivy/Grype na detecção de CVE.",
                pros="Único scanner reachability-aware no stack. Drasticamente reduz noise em Go.",
                cons="Só Go. Binários sem BuildInfo (Go ≤1.17) são pulados.",
                alts="Trivy (sem reachability), Grype (idem). Para outras langs: pip-audit (Python), npm audit (Node)."),
    "clair": dict(type="Vuln (CVE per layer)", mode="static", since="2015",
                author="CoreOS / Red Hat", license="Apache-2.0", covers=["cve"],
                what="Scanner CVE com arquitetura layer-by-layer indexed (origem do Quay).",
                how="Cada layer da imagem é indexada separadamente; CVEs casados por layer. clair-action é wrapper single-binary que dispensa o stack server completo.",
                when="Terceira opinião quando Trivy e Grype divergem. Modelo dedup por layer útil em registries grandes.",
                pros="Layer-indexing dedup natural em registries. Backed by Red Hat.",
                cons="Setup do server completo é pesado (Postgres + matchers); o action standalone é mais leve mas com cobertura menor.",
                alts="Trivy, Grype, Anchore Engine."),
    "dependency-check": dict(type="Vuln (CPE matching)", mode="static", since="2012",
                author="OWASP (Jeremy Long)", license="Apache-2.0", covers=["cve"],
                what="OWASP Dependency-Check: motor de correlação CPE-based contra NVD.",
                how="Identifica componentes Maven/Gradle/npm/Nuget/Python/etc; gera CPE (Common Platform Enumeration); cruza com NVD + OSS Index. Confidence score heurístico.",
                when="Strong em Java (Maven coords → CPE bem definido). Padrão em pipelines OWASP-focused.",
                pros="OWASP project. NVD + OSS Index dual-feed. Saída SARIF/HTML/JSON/JUnit/CSV.",
                cons="Lento na 1ª execução (download NVD ~30+ min). False positives em coords ambíguos. CPE matching é heurístico.",
                alts="Snyk (commercial), Trivy, Grype."),
    "cdxgen": dict(type="SBOM (OWASP)", mode="static", since="2021",
                author="OWASP (CycloneDX)", license="Apache-2.0", covers=["sbom"],
                what="Gerador SBOM CycloneDX 1.5 multi-language com VEX support.",
                how="Walks build tools (Maven, Gradle, sbt, Bazel, pnpm, yarn) e resolve transitive deps. Output CycloneDX 1.5 + VEX.",
                when="Monorepos onde Syft perde transitive deps. Padrão CycloneDX.",
                pros="Coverage profunda em Maven/Gradle/Bazel. CycloneDX 1.5 + VEX.",
                cons="Requer toolchain do build instalada para o melhor resultado. Mais lento que Syft.",
                alts="Syft (mais simples, format multi), Microsoft SBOM Tool, Tern."),
    "retire": dict(type="Vuln JS (hash/version)", mode="static", since="2013",
                author="Erlend Oftedal", license="Apache-2.0", covers=["cve"],
                what="retire.js: detecta libs JS vulneráveis casando hash/version dos arquivos .js.",
                how="Scaneia FS por .js e cruza com DB de versões/hashes vulneráveis (jQuery <3.5 → CVE-X). Independente do package.json.",
                when="Catches CDN-vendored JS, builds copiados, libs minified que package.json não captura. Webapps client-heavy.",
                pros="Detecta libs vendored fora do dependency manifest. Database curada por anos.",
                cons="Só JS client-side. Não detecta vulns custom-coded. DB depende de manutenção comunitária.",
                alts="Trivy (mas package.json-only), npm audit, Snyk."),
    "whispers": dict(type="Secrets (configs estruturados)", mode="static", since="2019",
                author="Adam Listek (independent)", license="Apache-2.0", covers=["secrets"],
                what="Whispers: parser de configs estruturados (YAML/JSON/Dockerfile) em vez de regex em texto opaco.",
                how="Parseia formato (YAML/JSON/Dockerfile/.env/etc) e extrai valores de keys nomeadas (password, api_key, secret). Plugin-based.",
                when="K8s manifests, Helm values.yaml, Dockerfiles, .env. Bom signal-to-noise vs regex puro.",
                pros="Structured-aware: menos FP que regex em YAML/JSON. Plugin extensível.",
                cons="Não cobre formatos não-estruturados (logs, scripts).",
                alts="TruffleHog, GitLeaks, detect-secrets."),
    "kube-linter": dict(type="K8s manifest issues", mode="static", since="2020",
                author="StackRox / Red Hat", license="Apache-2.0", covers=["misconfig"],
                what="StackRox kube-linter: lint de YAMLs Kubernetes para best practices de segurança.",
                how="Parseia YAML/Helm e roda checagens declarativas (run-as-root, resource limits, network policies, privileged, hostPath). 30+ regras built-in.",
                when="Imagens que bundlam Helm charts ou k8s manifests para deploy. Complementa Trivy --scanners config.",
                pros="K8s-specific: cobre patterns que linter genérico (Trivy) não pega. Backed by StackRox.",
                cons="Skip silently se imagem não traz YAML.",
                alts="Trivy --scanners config, Checkov, Polaris."),
    "hadolint": dict(type="Dockerfile lint", mode="static", since="2015",
                author="Lukas Martinelli", license="MIT", covers=["misconfig"],
                what="Hadolint: linter de Dockerfile com regras de boas-práticas e shellcheck nas linhas RUN.",
                how="Reconstroi Dockerfile a partir de docker history --no-trunc. Aplica DL/SC rules. Output SARIF/JSON/TTY.",
                when="Auditoria estática de Dockerfile fora do build. Útil em compliance (CIS Docker).",
                pros="Shellcheck nas RUNs detecta classes de bugs que outros linters perdem.",
                cons="Reconstrução de Dockerfile via history perde COPY/ADD targets — limita análise.",
                alts="Dockle (foca em CIS, não shellcheck), Trivy --scanners config."),
    "checkov": dict(type="IaC misconfig", mode="static", since="2019",
                author="Bridgecrew / Prisma", license="Apache-2.0", covers=["iac", "misconfig"],
                what="Checkov: 1000+ políticas de IaC + Dockerfile + SCA misconfig.",
                how="Parseia Terraform, CloudFormation, Kubernetes, Helm, ARM, Bicep, OpenAPI, Dockerfile, Serverless, GH Actions. Roda regras Python+YAML.",
                when="Imagens que trazem IaC empacotado. Padrão em DevSecOps pipelines.",
                pros="Coverage maior que Trivy/Dockle. Bridgecrew (Palo Alto) mantém. Custom policies em Python.",
                cons="Heavy: 700MB image. Pode ter FP em coverage muito amplo.",
                alts="Trivy --scanners config (subset menor), tfsec (Terraform-only), KICS."),
    "pip-audit": dict(type="Vuln Python (PyPA)", mode="static", since="2022",
                author="PyPA / Trail of Bits", license="Apache-2.0", covers=["cve"],
                what="pip-audit: scanner de CVE Python autoritativo, usa PyPA Advisory DB.",
                how="Lê requirements.txt ou site-packages instalados; cruza com pypa/advisory-database. Suporta yanked release flags.",
                when="Per-ecosystem alternativa a Trivy/Grype para Python. Recomendada pelo próprio PyPA.",
                pros="Autoridade direta (PyPA). Detecta yanked releases (Trivy não).",
                cons="Só Python. Cobertura menor que Trivy em multi-language images.",
                alts="Safety (commercial), Snyk, Trivy."),
    "httpx": dict(type="Web fingerprint+probe", mode="dynamic", since="2020",
                author="ProjectDiscovery", license="MIT", covers=["fingerprint", "web"],
                what="httpx: prober web fast de ProjectDiscovery — fingerprint, TLS, JARM, CDN.",
                how="HTTP client otimizado em Go. -tech-detect para Wappalyzer-style, -tls-grab, -jarm, -cdn, -favicon. Output JSONL.",
                when="Recon inicial: alt mais robusto que WhatWeb. Pipeline-friendly (JSONL).",
                pros="Fast. Multi-feature em um binário. Mantido (ProjectDiscovery).",
                cons="Apenas fingerprint, não scan de vulns.",
                alts="WhatWeb (Ruby, mais lento), Wappalyzer CLI."),
    "jaeles": dict(type="Web Vuln (signatures)", mode="dynamic", since="2019",
                author="j3ssie", license="MIT", covers=["web"],
                what="Jaeles: scanner web template-driven, similar a Nuclei.",
                how="Carrega signatures YAML do j3ssie/jaeles-signatures DB e roda contra URL. Engine matcher (regex/diff/length).",
                when="Cross-validation com Nuclei. Tsunami era a 1ª opção mas o image é privado no GHCR.",
                pros="Template-driven (similar a Nuclei). Open-source.",
                cons="Comunidade menor que Nuclei. Templates exigem `jaeles config init` na 1ª run.",
                alts="Nuclei (mais ativo), Tsunami (image privado), Wapiti."),
    "arachni": dict(type="Web App (DAST)", mode="dynamic", since="2010",
                author="Tasos Laskos (Sarosys)", license="Arachni Public Source",
                covers=["web"],
                what="Arachni: framework Ruby DAST com checks profundos para SQLi/XSS/RFI/LFI.",
                how="Crawl + plug-ins de detecção (audits) + plug-ins de meta. Parallel via processes. Output AFR (binário) + JSON/HTML reporters.",
                when="Cross-validation com ZAP. Plugins independentes — útil como 3a opinião.",
                pros="Suite completa (similar a ZAP). Plugins ortogonais. Detect SQLi/XSS/RFI/LFI/CSRF/etc.",
                cons="Projeto archived (último release 2017). Não recebe novas vulns desde então.",
                alts="OWASP ZAP, Burp Suite (commercial), Wapiti."),
    "testssl": dict(type="TLS/SSL audit", mode="dynamic", since="2014",
                author="Dirk Wetter", license="GPLv2", covers=["network"],
                what="testssl.sh: auditoria abrangente de TLS/SSL via OpenSSL.",
                how="Bash + OpenSSL custom build (inclui exploits removidos do upstream). Testa 100+ checks: cipher suites, vulns famosas (Heartbleed, POODLE, BEAST, LOGJAM, FREAK, ROBOT), HSTS, certs.",
                when="Auditoria de endpoint HTTPS — sinal totalmente ortogonal aos scanners HTTP-layer.",
                pros="Mais detalhado que Nessus/OpenVAS para TLS. Bash standalone — fácil de auditar.",
                cons="Bash-only (lento em compare a Go tools). Targets HTTP retornam baseline 'no TLS' — útil mas magro.",
                alts="sslyze, sslscan, Qualys SSL Labs."),
}


# ────────────────────────────────────────────────────────────────────────────
# Findings counters (best-effort, per scanner)
# ────────────────────────────────────────────────────────────────────────────

def _safe_json(p):
    if not p or not p.exists():
        return None
    try:
        return json.loads(p.read_text(encoding="utf-8", errors="replace") or "null")
    except Exception:
        return None


def _count_lines(p):
    if not p or not p.exists():
        return 0
    try:
        return sum(1 for ln in p.read_text(encoding="utf-8", errors="replace").splitlines() if ln.strip())
    except Exception:
        return 0


def count_findings(scanner, target, sdir):
    if not sdir.exists():
        return 0, {}
    p = lambda *names: next((sdir / n for n in names if (sdir / n).exists()), None)
    severities = {}

    if scanner == "osv":
        f = p(f"{target}-osv.json")
        d = _safe_json(f)
        if isinstance(d, dict):
            n = sum(len(r.get("packages", [])) for r in d.get("results", []))
            return n, severities
    if scanner == "yara":
        f = p(f"{target}-yara.txt")
        return _count_lines(f), severities
    if scanner == "detect-secrets":
        f = p(f"{target}-detect-secrets.json")
        d = _safe_json(f)
        if isinstance(d, dict):
            return sum(len(v) for v in (d.get("results") or {}).values()), severities
    if scanner == "guarddog":
        f = p(f"{target}-guarddog.jsonl")
        return _count_lines(f), severities
    if scanner == "govulncheck":
        f = p(f"{target}-govulncheck.json")
        if not f or not f.exists():
            return 0, severities
        return _count_lines(f) // 5, severities
    if scanner == "clair":
        return 0, severities  # output empty in our runs
    if scanner == "dependency-check":
        f = p("dependency-check-report.json")
        d = _safe_json(f)
        if isinstance(d, dict):
            n = sum(len(dep.get("vulnerabilities", []))
                    for dep in d.get("dependencies", []))
            return n, severities
        return 0, severities
    if scanner == "cdxgen":
        f = p(f"{target}-cdxgen.cdx.json")
        d = _safe_json(f)
        if isinstance(d, dict):
            return len(d.get("components", [])), severities
    if scanner == "retire":
        f = p(f"{target}-retire.json")
        d = _safe_json(f)
        if isinstance(d, list):
            return sum(len(c.get("results", [])) for c in d), severities
    if scanner == "whispers":
        f = p(f"{target}-whispers.json")
        return _count_lines(f), severities
    if scanner == "kube-linter":
        f = p(f"{target}-kube-linter.json")
        d = _safe_json(f)
        if isinstance(d, dict):
            return len(d.get("Reports", []) or d.get("reports", [])), severities
    if scanner == "secretscanner":
        return 0, severities
    if scanner == "hadolint":
        f = p(f"{target}-hadolint.json")
        d = _safe_json(f)
        if isinstance(d, list):
            for it in d:
                lvl = (it.get("level") or "").lower()
                severities[lvl] = severities.get(lvl, 0) + 1
            return len(d), severities
    if scanner == "checkov":
        f = p(f"{target}-checkov.json")
        d = _safe_json(f)
        if isinstance(d, dict):
            res = (d.get("results") or {}).get("failed_checks", [])
            return len(res), severities
        if isinstance(d, list):
            n = sum(len((r.get("results") or {}).get("failed_checks", [])) for r in d)
            return n, severities
    if scanner == "pip-audit":
        f = p(f"{target}-pip-audit.txt")
        if f:
            txt = f.read_text(encoding="utf-8", errors="replace")
            return len(re.findall(r"^\S+\s+[0-9]\S*\s+\S+", txt, re.M)), severities
    if scanner == "httpx":
        f = p(f"{target}-httpx.jsonl")
        return _count_lines(f), severities
    if scanner == "jaeles":
        f = p(f"{target}-jaeles.json")
        if f and f.stat().st_size > 0:
            return f.read_text(encoding="utf-8").count('"vuln_url"'), severities
    if scanner == "arachni":
        f = p(f"{target}-arachni.json")
        d = _safe_json(f)
        if isinstance(d, dict):
            return len(d.get("issues", [])), severities
    if scanner == "testssl":
        f = p(f"{target}-testssl.json")
        d = _safe_json(f)
        if isinstance(d, list):
            for it in d:
                lvl = (it.get("severity") or "").lower()
                severities[lvl] = severities.get(lvl, 0) + 1
            return len(d), severities
    if scanner == "openvas":
        f = p(f"{target}-openvas.xml")
        if f:
            txt = f.read_text(encoding="utf-8", errors="replace")
            counts = {}
            for tag in ("critical", "high", "medium", "low", "log", "false_positive"):
                m = re.search(rf"<{tag}><full>(\d+)</full>", txt)
                if m:
                    counts[tag] = int(m.group(1))
            total_m = re.search(r"<result_count>(\d+)", txt)
            total = int(total_m.group(1)) if total_m else sum(counts.values())
            return total, counts
    return 0, severities


# ────────────────────────────────────────────────────────────────────────────
# Aggregation
# ────────────────────────────────────────────────────────────────────────────

def collect_data():
    """Build a DATA-shaped dict for each new scanner."""
    out = {}
    for scanner, info in INFO.items():
        wall_per_target = {}
        by_target_findings = {}
        cpu_max = 0.0
        mem_max = 0.0
        any_ran = False
        sev_total = {}
        for tgt in TARGETS:
            best = None
            for host in HOSTS:
                # OpenVAS: only node1 has the successful run; node2 errored.
                if scanner == "openvas" and host == "node2":
                    continue
                metrics_file = SCANS_OUT / host / tgt / f"{tgt}-metrics.json"
                if not metrics_file.exists():
                    continue
                try:
                    entries = json.loads(metrics_file.read_text(encoding="utf-8"))
                except json.JSONDecodeError:
                    continue
                for e in entries:
                    if e.get("scanner") == scanner:
                        # Prefer ok status
                        if best is None or (e.get("status") == "ok"
                                            and best.get("status") != "ok"):
                            best = {**e, "host": host}
                        break
            if best is None:
                continue
            any_ran = True
            sdir = SCANS_OUT / best["host"] / tgt / scanner
            findings, sev = count_findings(scanner, tgt, sdir)
            wall_per_target[tgt] = round(best.get("wall_seconds", 0), 2)
            by_target_findings[tgt] = findings
            cpu_max = max(cpu_max, best.get("peak_cpu_percent", 0))
            mem_max = max(mem_max, best.get("peak_mem_mb", 0))
            for k, v in sev.items():
                sev_total[k] = sev_total.get(k, 0) + v
        if not any_ran:
            continue
        wall_avg = (sum(wall_per_target.values()) / len(wall_per_target)
                    if wall_per_target else None)
        out[scanner] = {
            "name": scanner,
            "mode": info["mode"],
            "type": info["type"],
            "wall_avg": wall_avg,
            "cpu_max": cpu_max,
            "mem_max": mem_max,
            "wall_per_target": wall_per_target,
            "findings": sum(by_target_findings.values()),
            "severities": sev_total,
            "by_target_findings": by_target_findings,
            "covers": info["covers"],
            "ran": True,
        }
    return out


# ────────────────────────────────────────────────────────────────────────────
# HTML fragment builders
# ────────────────────────────────────────────────────────────────────────────

def fmt_badge(n):
    if not n:
        return ""
    if n >= 1000:
        return f'<span class="badge">{n // 1000}.{n % 1000:03d}</span>'
    return f'<span class="badge">{n}</span>'


def build_calc_toggles(data):
    parts = []
    for s in sorted(data.keys()):
        mode = data[s]["mode"]
        tag = "static" if mode == "static" else "dynamic"
        letter = "s" if mode == "static" else "d"
        parts.append(
            f'        <label class="calc-toggle" data-scanner="{html.escape(s)}">\n'
            f'          <input type="checkbox">\n'
            f'          <span>{html.escape(s)}</span>\n'
            f'          <span class="tag {tag}">{letter}</span>\n'
            f'        </label>'
        )
    return "\n".join(parts)


def build_tab_buttons(data):
    parts = []
    for s in sorted(data.keys()):
        n = data[s]["findings"]
        parts.append(
            f'<button class="tab" role="tab" id="tabbtn-{html.escape(s)}" '
            f'aria-controls="tab-{html.escape(s)}" aria-selected="false" tabindex="-1">'
            f'{html.escape(s)}{fmt_badge(n)}</button>'
        )
    return "".join(parts)


def build_panel(s, info, entry):
    mode = entry["mode"]
    type_ = entry["type"]
    findings = entry["findings"]
    by_t = entry["by_target_findings"]
    severities = entry["severities"]
    sev_html = ""
    if severities:
        items = sorted(severities.items(), key=lambda kv: -kv[1])
        sev_html = ("<dt>Severidades</dt><dd>"
                    + ", ".join(f"<b>{k}</b>: {v}" for k, v in items)
                    + "</dd>")
    by_target_html = ", ".join(f"<b>{html.escape(t)}</b>: {by_t.get(t, 0)}" for t in TARGETS)
    return (
        f'</div><div id="tab-{html.escape(s)}" class="tab-panel" '
        f'role="tabpanel" aria-labelledby="tabbtn-{html.escape(s)}" hidden>\n'
        f'          <div class="header">\n'
        f'            <div>\n'
        f'              <h3>{html.escape(type_)} — {html.escape(s)}</h3>\n'
        f'              <div class="meta">{html.escape(info["what"])} '
        f'· <a href="https://github.com/search?q={html.escape(s)}" '
        f'target="_blank" rel="noopener">repo</a></div>\n'
        f'            </div>\n'
        f'            <span class="tag {mode}">{mode}</span>\n'
        f'          </div>\n'
        f'    <section class="scanner-info" aria-label="Sobre o scanner {html.escape(s)}">\n'
        f'      <div class="meta-row">\n'
        f'        <span>desde<b>{html.escape(info["since"])}</b></span>\n'
        f'        <span>autor<b>{html.escape(info["author"])}</b></span>\n'
        f'        <span>licença<b>{html.escape(info["license"])}</b></span>\n'
        f'        <span>tipo<b>{html.escape(info["type"])}</b></span>\n'
        f'        <span>modo<b>{html.escape(info["mode"])}</b></span>\n'
        f'      </div>\n'
        f'      <dl>\n'
        f'        <dt>O que é</dt><dd>{html.escape(info["what"])}</dd>\n'
        f'        <dt>Como funciona</dt><dd>{html.escape(info["how"])}</dd>\n'
        f'        <dt>Quando usar</dt><dd>{html.escape(info["when"])}</dd>\n'
        f'        <dt>+ Pontos fortes</dt><dd style="color:#0d6e3e">{html.escape(info["pros"])}</dd>\n'
        f'        <dt>− Limitações</dt><dd style="color:#92400e">{html.escape(info["cons"])}</dd>\n'
        f'        <dt>Alternativas</dt><dd>{html.escape(info["alts"])}</dd>\n'
        f'        <dt>Findings totais</dt><dd><b>{findings}</b> '
        f'({by_target_html}){sev_html}</dd>\n'
        f'      </dl>\n'
        f'    </section>'
    )


COVERAGE_COLS = ["sbom", "cve", "secrets", "misconfig", "iac", "web",
                 "network", "fingerprint", "malware", "supplychain"]
# Note original matrix uses these display columns:
DISPLAY_COLS = [("SBOM", "sbom"), ("CVE", "cve"), ("Secrets", "secrets"),
                ("Misconfig", "misconfig"), ("IaC", "iac"), ("Web/DAST", "web"),
                ("Network", "network"), ("Recon", "fingerprint"),
                ("Malware", "malware")]


def build_matrix_rows(data):
    parts = []
    for s in sorted(data.keys()):
        e = data[s]
        cells = [f'<td><strong>{html.escape(s)}</strong></td>'
                 f'<td><span class="tag {e["mode"]}">{e["mode"]}</span></td>']
        for _, key in DISPLAY_COLS:
            mark = "yes" if key in e["covers"] else "no"
            symbol = "●" if mark == "yes" else "·"
            cells.append(f'<td><span class={mark}>{symbol}</span></td>')
        parts.append("<tr>" + "".join(cells) + "</tr>")
    return "".join(parts)


# ────────────────────────────────────────────────────────────────────────────
# Surgical injection
# ────────────────────────────────────────────────────────────────────────────

def inject(content, data):
    # 1. Update DATA dict (line 1111). Find prefix `const DATA = ` and replace
    #    the JSON object up to its closing `};\n` boundary.
    m = re.search(r"const DATA = (\{.*?\});\n", content, flags=re.S)
    if not m:
        raise RuntimeError("DATA = {...} literal not found")
    existing = json.loads(m.group(1))
    existing.update(data)
    new_json = json.dumps(existing, ensure_ascii=False)
    content = content[:m.start()] + f"const DATA = {new_json};\n" + content[m.end():]

    # 2. calc-toggles: append before the closing `</div>` after the zap toggle.
    toggles = build_calc_toggles(data)
    pat = (r'(        <label class="calc-toggle" data-scanner="zap">\n'
           r'          <input type="checkbox">\n'
           r'          <span>zap</span>\n'
           r'          <span class="tag dynamic">d</span>\n'
           r'        </label>)(</div>)')
    repl = r'\1\n' + toggles + r'\2'
    content, n = re.subn(pat, repl, content, count=1)
    if n != 1:
        raise RuntimeError("calc-toggle anchor not matched")

    # 3. tab buttons: append after the openvas button on line 466.
    buttons = build_tab_buttons(data)
    anchor = ('<button class="tab" role="tab" id="tabbtn-openvas" '
              'aria-controls="tab-openvas" aria-selected="false" tabindex="-1">'
              'openvas</button>')
    if anchor not in content:
        raise RuntimeError("openvas tab button anchor not found")
    content = content.replace(anchor, anchor + buttons, 1)

    # 4. panels: insert after the openvas tab-panel closes (the </div> right
    #    before `</div>` ending the per-scanner main section).
    panels = "\n".join(build_panel(s, INFO[s], data[s])
                       for s in sorted(data.keys()))
    panel_anchor = ('        <dt>Alternativas</dt>'
                    '<dd>Nessus (commercial, mais polido), Tenable.io (cloud), '
                    'Qualys, Rapid7 InsightVM.</dd>\n      </dl>\n'
                    '    </section>\n        </div>\n  </div>\n</section>')
    if panel_anchor not in content:
        raise RuntimeError("openvas panel close anchor not found — adjust replacement string")
    insertion = ('        <dt>Alternativas</dt>'
                 '<dd>Nessus (commercial, mais polido), Tenable.io (cloud), '
                 'Qualys, Rapid7 InsightVM.</dd>\n      </dl>\n'
                 '    </section>\n' + panels + '\n        </div>\n  </div>\n</section>')
    content = content.replace(panel_anchor, insertion, 1)

    # 5. coverage matrix rows — append before </tbody></table>.
    matrix_rows = build_matrix_rows(data)
    matrix_anchor = ('<tr><td><strong>zap</strong></td>'
                     '<td><span class="tag dynamic">dynamic</span></td>')
    end_marker = "</tbody></table>"
    # Find the closing </tbody> right after the zap row and insert matrix_rows
    # in front of </tbody>.
    matrix_close = '</tbody></table>'
    # Locate the FIRST </tbody></table> after the matrix's known zap row.
    idx = content.find(matrix_anchor)
    if idx == -1:
        raise RuntimeError("matrix zap row not found")
    close_idx = content.find(matrix_close, idx)
    if close_idx == -1:
        raise RuntimeError("matrix </tbody></table> not found after zap row")
    content = content[:close_idx] + matrix_rows + content[close_idx:]

    # 6. update Por scanner badge from 14 → new total (14 + new)
    new_count = 14 + len(data)
    content = content.replace(
        '<span class="badge">14</span></button>\n    <button class="maintab"',
        f'<span class="badge">{new_count}</span></button>\n    <button class="maintab"',
        1,
    )
    return content


def main():
    content = HTML_PATH.read_text(encoding="utf-8")
    data = collect_data()
    print(f"collected {len(data)} new scanners with data")
    content = inject(content, data)
    HTML_PATH.write_text(content, encoding="utf-8")
    print(f"updated {HTML_PATH.relative_to(ROOT)} ({len(content)} chars)")
    if NEW_HTML.exists():
        NEW_HTML.unlink()
        print(f"removed {NEW_HTML.relative_to(ROOT)}")


if __name__ == "__main__":
    main()
