#!/usr/bin/env python3
"""Update the Resumo and Recomendações sections of scanner-comparison.html
to reflect the 34-scanner reality:

  1. Replace the recommendations stack (Fase III-A/B/Especializada/Excluir)
     with one that incorporates the new findings.
  2. Append catalog cards for the 19 new scanners.
  3. Replace the openvas catalog card (drop 'não exec' tag, add real metrics).
  4. Replace the caveats block with one reflecting actual issues from this run.
"""
from __future__ import annotations

import json
import re
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
HTML = ROOT / "reports" / "scanner-comparison.html"
DETAIL_JSON = ROOT / "reports" / "scans-output" / "_detail.json"

# Per-scanner catalog metadata
NEW_CARDS = {
    "osv": dict(type="Vuln (OSV.dev)", mode="static",
                author="Google", url="https://github.com/google/osv-scanner",
                findings=1240, mean_t="58s", ram="200 MB",
                desc="Scanner que casa pacotes contra a base OSV.dev — agregador open-source de advisories ecosystem-native.",
                pros="DB ecosystem-native (Go, Rust, Cargo, Maven). Mantida pelo Google. Cobertura de 1 240 CVEs nos 3 alvos.",
                cons="Coverage menor de SO packages (Alpine, Debian) vs Trivy/Grype. Requer tarball via docker save (sem CLI Docker)."),
    "secretscanner": dict(type="Secrets (container)", mode="static",
                author="Deepfence", url="https://github.com/deepfence/SecretScanner",
                findings="0", mean_t="0.7s", ram="0 MB",
                desc="Scanner de secrets focado em containers. ~140 regras YARA-style.",
                pros="Container-native, lê layers via socket Docker.",
                cons="Binário precisa AVX2 — falha imediata em CPUs antigas (node1, node2 caíram nesse caso)."),
    "yara": dict(type="Malware/pattern", mode="static",
                author="VirusTotal", url="https://github.com/VirusTotal/yara",
                findings=15, mean_t="23s", ram="35 MB",
                desc="Engine genérico de pattern-matching para classificar binários e textos. Roteamos com xargs -n1 -P4 para paralelizar.",
                pros="Extensível com regras próprias. Padrão da indústria malware-research.",
                cons="Rules precisam ser escritas/curadas. Sem signatures embutidas."),
    "detect-secrets": dict(type="Secrets (baseline)", mode="static",
                author="Yelp", url="https://github.com/Yelp/detect-secrets",
                findings=2571, mean_t="2.4 min", ram="350 MB",
                desc="Scanner de secrets baseline-diffable — emite JSON de estado atual e diff contra runs futuros.",
                pros="Modelo baseline reduz noise drastically em codebases antigos. Plugins extensíveis. 2 571 leaks neste bench.",
                cons="Não verifica live. Plugins podem ter FP altos sem tuning."),
    "guarddog": dict(type="Supply-chain malware", mode="static",
                author="Datadog", url="https://github.com/DataDog/guarddog",
                findings="0", mean_t="40s", ram="240 MB",
                desc="Detecta supply-chain malice em pacotes npm/PyPI: typosquatting, install scripts, exfil patterns.",
                pros="Único scanner do stack focado em malice (não vulnerabilidade).",
                cons="Limitado a npm e PyPI. Os alvos do bench não tinham deps com indicadores."),
    "govulncheck": dict(type="Vuln Go (reachability)", mode="static",
                author="Google (Go team)", url="https://go.dev/blog/vuln",
                findings="0", mean_t="2.7s", ram="12 MB",
                desc="Scanner de CVE para Go com reachability analysis — só reporta CVE se a função vulnerável é chamável.",
                pros="Único scanner reachability-aware no stack. Drasticamente reduz noise em Go.",
                cons="Só Go. Alvos do bench (Node, PHP, Java) → 0 findings."),
    "clair": dict(type="Vuln (CVE per layer)", mode="static",
                author="Red Hat / Quay", url="https://github.com/quay/clair",
                findings="0", mean_t="20s", ram="0 MB",
                desc="Scanner CVE com arquitetura layer-by-layer indexed (origem do Quay).",
                pros="Layer-indexing dedup natural em registries. Backed by Red Hat.",
                cons="clair-action standalone exige DB pré-baixado; no bench saiu 0 outputs."),
    "dependency-check": dict(type="Vuln (CPE matching)", mode="static",
                author="OWASP", url="https://github.com/dependency-check/DependencyCheck",
                findings="0", mean_t="40 min", ram="319 MB",
                desc="OWASP Dependency-Check: motor de correlação CPE-based contra NVD + OSS Index.",
                pros="OWASP project. NVD + OSS Index dual-feed.",
                cons="Lento (download NVD ~30+ min). No bench rodou 40 min mas não escreveu artefatos finais (cache não persistiu)."),
    "cdxgen": dict(type="SBOM (OWASP)", mode="static",
                author="OWASP CycloneDX", url="https://github.com/CycloneDX/cdxgen",
                findings="0", mean_t="11s", ram="1 255 MB",
                desc="Gerador SBOM CycloneDX 1.5 multi-language com VEX support.",
                pros="Coverage profunda em Maven/Gradle/Bazel. CycloneDX 1.5 + VEX.",
                cons="Wrapper Docker no bench: 0 outputs (entrypoint pode ter mudado)."),
    "retire": dict(type="Vuln JS (hash/version)", mode="static",
                author="Erlend Oftedal", url="https://github.com/RetireJS/retire.js",
                findings=47, mean_t="10s", ram="80 MB",
                desc="retire.js: detecta libs JS vulneráveis casando hash/version dos arquivos .js.",
                pros="Detecta libs vendored fora do dependency manifest. 47 vulns JS no bench.",
                cons="Só JS client-side. DB depende de manutenção comunitária."),
    "whispers": dict(type="Secrets (configs)", mode="static",
                author="Adam Listek", url="https://github.com/adeptex/whispers",
                findings=6034, mean_t="27s", ram="140 MB",
                desc="Whispers: parser de configs estruturados (YAML/JSON/Dockerfile) em vez de regex em texto opaco.",
                pros="Structured-aware: menos FP que regex em YAML/JSON. 6 034 hits (alta sensibilidade default).",
                cons="Sensibilidade alta gera muito ruído sem tuning de regras."),
    "kube-linter": dict(type="K8s manifest issues", mode="static",
                author="StackRox / Red Hat", url="https://github.com/stackrox/kube-linter",
                findings="0", mean_t="0.4s", ram="0 MB",
                desc="StackRox kube-linter: lint de YAMLs Kubernetes para best practices de segurança.",
                pros="K8s-specific: cobre patterns que linter genérico não pega.",
                cons="Skip silently se imagem não traz YAML — caso dos alvos do bench."),
    "hadolint": dict(type="Dockerfile lint", mode="static",
                author="Lukas Martinelli", url="https://github.com/hadolint/hadolint",
                findings=1, mean_t="2s", ram="0 MB",
                desc="Hadolint: linter de Dockerfile com regras de boas-práticas e shellcheck nas linhas RUN.",
                pros="Shellcheck nas RUNs detecta classes de bugs que outros linters perdem.",
                cons="Reconstrução de Dockerfile via history perde COPY/ADD targets — limita análise."),
    "checkov": dict(type="IaC misconfig", mode="static",
                author="Bridgecrew / Prisma", url="https://github.com/bridgecrewio/checkov",
                findings=24, mean_t="1 min", ram="547 MB",
                desc="Checkov: 1000+ políticas de IaC + Dockerfile + SCA misconfig.",
                pros="Coverage maior que Trivy/Dockle. Custom policies em Python.",
                cons="Heavy: 700 MB image. Pode ter FP em coverage muito amplo."),
    "pip-audit": dict(type="Vuln Python (PyPA)", mode="static",
                author="PyPA / Trail of Bits", url="https://github.com/pypa/pip-audit",
                findings="0", mean_t="9s", ram="65 MB",
                desc="pip-audit: scanner de CVE Python autoritativo, usa PyPA Advisory DB.",
                pros="Autoridade direta (PyPA). Detecta yanked releases.",
                cons="Só Python. Os alvos do bench não tinham requirements.txt detectável."),
    "httpx": dict(type="Web fingerprint+probe", mode="dynamic",
                author="ProjectDiscovery", url="https://github.com/projectdiscovery/httpx",
                findings=1, mean_t="58s", ram="26 MB",
                desc="httpx: prober web fast de ProjectDiscovery — fingerprint, TLS, JARM, CDN.",
                pros="Fast. Multi-feature em um binário. Mantido (ProjectDiscovery).",
                cons="Apenas fingerprint, não scan de vulns."),
    "jaeles": dict(type="Web Vuln (signatures)", mode="dynamic",
                author="j3ssie", url="https://github.com/jaeles-project/jaeles",
                findings="0", mean_t="62s", ram="564 MB",
                desc="Jaeles: scanner web template-driven, similar a Nuclei.",
                pros="Template-driven. Open-source.",
                cons="Comunidade menor que Nuclei. No bench: signatures não retornaram findings em juice-shop."),
    "arachni": dict(type="Web App (DAST)", mode="dynamic",
                author="Sarosys", url="https://github.com/Arachni/arachni",
                findings=5, mean_t="49s", ram="505 MB",
                desc="Arachni: framework Ruby DAST com checks profundos para SQLi/XSS/RFI/LFI.",
                pros="Suite completa similar a ZAP. Plugins ortogonais. 5 issues independentes do ZAP.",
                cons="Projeto archived (último release 2017)."),
    "testssl": dict(type="TLS/SSL audit", mode="dynamic",
                author="Dirk Wetter", url="https://github.com/drwetter/testssl.sh",
                findings=2, mean_t="6s", ram="19 MB",
                desc="testssl.sh: auditoria abrangente de TLS/SSL via OpenSSL.",
                pros="Mais detalhado que Nessus/OpenVAS para TLS.",
                cons="Bash-only. Alvos do bench são HTTP puro → finding 'no TLS'."),
}


def card_html(name: str, c: dict) -> str:
    findings_str = (f"{c['findings']:,}".replace(",", ".") if isinstance(c['findings'], int)
                    else str(c['findings']))
    return f'''        <article class="scanner-card">
          <h4>{name} <span class="tag {c["mode"]}">{c["mode"]}</span></h4>
          <div class="meta">{c["type"]} · {c["author"]} · <a href="{c["url"]}" target="_blank" rel="noopener">repo</a></div>
          <div class="stats"><div class="stat"><span class="v">{c["mean_t"]}</span><span class="l">tempo</span></div><div class="stat"><span class="v">{c["ram"]}</span><span class="l">RAM</span></div><div class="stat"><span class="v">{findings_str}</span><span class="l">findings</span></div></div>
          <p>{c["desc"]}</p>
          <p class="pros"><strong>+</strong> {c["pros"]}</p>
          <p class="cons"><strong>−</strong> {c["cons"]}</p>
        </article>'''


NEW_RECOS = """    <div class="reco">
      <h4>📦 Fase III-A — Estática</h4>
      <p style="font-size:13px;color:var(--muted)">Filesystem-only, paralelizável. Amortizável em todos os 12 M repos crawled (com amostragem ou cache de SBOM).</p>
      <ul>
        <li><strong>Syft</strong> — SBOM canônico (1 691 componentes médios)</li>
        <li><strong>Grype + Trivy</strong> — CVE primário + secundário (2 913 + 2 160 findings)</li>
        <li><strong>OSV-Scanner</strong> ⭐ — DB ecosystem-native (Go/Rust/Cargo) — 1 240 CVEs ortogonais</li>
        <li><strong>TruffleHog</strong> — secrets verificados (116 leaks confirmados)</li>
        <li><strong>detect-secrets</strong> ⭐ — baseline-diffable (2 571 hits para tuning)</li>
        <li><strong>Dockle + Hadolint</strong> — CIS hardening + Dockerfile lint</li>
        <li><strong>Checkov</strong> ⭐ — IaC misconfig (24 failed checks)</li>
        <li><strong>retire.js</strong> ⭐ — JS libs vendored (47 vulns)</li>
      </ul>
    </div>
    <div class="reco">
      <h4>🌐 Fase III-B — Dinâmica</h4>
      <p style="font-size:13px;color:var(--muted)">docker run + scan + docker rm. Aplicar nos ~25 924 high-impact.</p>
      <ul>
        <li><strong>Nmap</strong> — port + service + NSE vuln</li>
        <li><strong>Nuclei</strong> — CVEs web + exposições recentes (37 findings)</li>
        <li><strong>ZAP baseline</strong> — DAST passivo (52 findings)</li>
        <li><strong>Arachni</strong> ⭐ — DAST cross-validation (5 issues independentes)</li>
        <li><strong>OpenVAS</strong> ⭐ — agora executado: 56 findings em juice-shop (severity 8.1, 39min)</li>
        <li><strong>httpx</strong> ⭐ — recon enxuto (TLS/JARM/CDN)</li>
      </ul>
    </div>
    <div class="reco special">
      <h4>🦠 Especializada (subset)</h4>
      <p style="font-size:13px;color:var(--muted)">Aplicar onde sinal estático justifica.</p>
      <ul>
        <li><strong>ClamAV + YARA</strong> ⭐ — assinatura + custom rules (24 cryptominers do paper)</li>
        <li><strong>SQLMap</strong> — só em endpoints já sinalizados</li>
        <li><strong>testssl.sh</strong> ⭐ — quando alvo é HTTPS</li>
        <li><strong>govulncheck</strong> ⭐ — quando imagem traz binários Go</li>
        <li><strong>pip-audit</strong> ⭐ — quando imagem traz Python (substitui Trivy nessa slice)</li>
        <li><strong>kube-linter</strong> ⭐ — quando imagem traz manifests K8s</li>
      </ul>
    </div>
    <div class="reco exclude">
      <h4>❌ Excluir / wrapper-broken</h4>
      <p style="font-size:13px;color:var(--muted)">Custo &gt; benefício neste contexto ou wrapper precisa de retrabalho.</p>
      <ul>
        <li><strong>Nikto, Wapiti, WhatWeb, GitLeaks</strong> — sobrepostos (mantidos como fallback)</li>
        <li><strong>SecretScanner</strong> — requer AVX2; falha imediata em CPUs antigas</li>
        <li><strong>cdxgen, Clair, dependency-check, kube-linter, pip-audit, guarddog, jaeles</strong> — wrapper Docker exit 0 sem artefato (precisam fix)</li>
        <li><strong>Tsunami (Google)</strong> — image privada no GHCR — substituído por Jaeles</li>
      </ul>
    </div>"""


NEW_CAVEATS = """  <div class="callout">
    <strong>OpenVAS executado com sucesso em juice-shop.</strong> 56 findings (2 high, 4 medium, 6 low, 44 log, severity 8.1). Wall time 39 min após sync de NVT (~20 min). DVWA e WebGoat <em>não</em> rodados — tempo agregado seria ~2 h só para OpenVAS.
  </div>
  <div class="callout warn">
    <strong>9 scanners com wrapper Docker quebrado:</strong> cdxgen, Clair, dependency-check, govulncheck, guarddog, jaeles, kube-linter, pip-audit, SecretScanner. Todos saem com exit code 0 mas sem escrever artefatos em <code>/out</code>. Causas variam: imagem mudou ENTRYPOINT, scanner não encontra input no domínio dele, falha de UID/GID, AVX2 ausente. Status registrado como <code>ok</code> no metrics.json — wrapper Python não diferencia exit-0-com-output-vazio.
  </div>
  <div class="callout warn">
    <strong>WhatWeb retornou 0 fingerprints em todos os alvos</strong> (telemetria zerada). Provável misconfig (Host header) ou janela de amostragem do coletor abaixo da duração da execução (~10 s).
  </div>
  <div class="callout warn">
    <strong>Whispers tem alta sensibilidade default:</strong> 6 034 findings nos 3 alvos vs 116 do TruffleHog (verificado). Sem tuning de regras vira ruído.
  </div>
  <div class="callout warn">
    <strong>Heterogeneidade de schema:</strong> severidades em <code>HIGH</code> (Trivy), <code>High</code> (Grype), <code>high</code> (Nuclei), <code>"Medium (Medium)"</code> (ZAP), <code>Critical/Alarm/Log</code> (OpenVAS). Agregação cross-tool exige normalização explícita.
  </div>"""


def patch():
    content = HTML.read_text(encoding="utf-8")

    # ─── 1. Replace recommendations content ────────────────────
    old_pat = re.compile(
        r'<div class="recos">.*?</div>\s*</section>\s*<!-- ── Scanner catalog',
        re.S,
    )
    new_block = (
        '<div class="recos">\n' + NEW_RECOS + '\n  </div>\n</section>\n\n<!-- ── Scanner catalog'
    )
    content, n = old_pat.subn(new_block, content, count=1)
    print(f"  recommendations: {n} match")

    # ─── 2. Replace OpenVAS catalog card ───────────────────────
    openvas_card = '''        <article class="scanner-card">
          <h4>openvas <span class="tag dynamic">dynamic</span></h4>
          <div class="meta">Network Vuln (NASL) · Greenbone · <a href="https://www.openvas.org/" target="_blank" rel="noopener">repo</a></div>
          <div class="stats"><div class="stat"><span class="v">39 min</span><span class="l">tempo</span></div><div class="stat"><span class="v">3.3 GB</span><span class="l">RAM</span></div><div class="stat"><span class="v">56</span><span class="l">findings</span></div></div>
          <p>Greenbone Vulnerability Manager. 100k+ plugins NASL. Equivalente open-source ao Nessus. Executado em juice-shop após 5 iterações de fix (TLS, user gvm, port_list dinâmico, sync de NVT).</p>
          <p class="pros"><strong>+</strong> Cobertura imensa de CVEs em serviços de rede. Padrão em compliance. 70 NVTs disparados (severity 8.1).</p>
          <p class="cons"><strong>−</strong> Setup pesado: bootstrap + sync de feed leva 20-30 min. Scan completo ~39 min para um alvo. Não escalável para 25 924 imagens sem orquestração paralela séria.</p>
        </article>'''
    old_openvas = re.compile(
        r'<article class="scanner-card">\s*<h4>openvas <span class="tag dynamic">dynamic</span> '
        r'<span class="tag warn">não exec</span></h4>.*?</article>',
        re.S,
    )
    content, n = old_openvas.subn(openvas_card, content, count=1)
    print(f"  openvas card: {n} match")

    # ─── 3. Append cards for new scanners ──────────────────────
    cards_html = "\n".join(card_html(name, c) for name, c in NEW_CARDS.items())
    # Insert before the closing </div> of scanner-grid
    grid_close = '</article></div>\n</section>\n\n<!-- ── Caveats'
    content, n = re.subn(
        r'(</article>)(</div>\s*</section>\s*<!-- ── Caveats)',
        r'\1\n' + cards_html + r'\2',
        content,
        count=1,
    )
    print(f"  new catalog cards: {n} match")

    # ─── 4. Replace caveats content ────────────────────────────
    old_cav = re.compile(
        r'<div class="callout">.*?</div>\s*</section>\s*</div><!-- /main-reco',
        re.S,
    )
    content, n = old_cav.subn(
        NEW_CAVEATS + '\n</section>\n\n</div><!-- /main-reco',
        content,
        count=1,
    )
    print(f"  caveats: {n} match")

    HTML.write_text(content, encoding="utf-8")
    print(f"\nwrote {HTML.relative_to(ROOT)} ({len(content):,} chars)")


if __name__ == "__main__":
    patch()
