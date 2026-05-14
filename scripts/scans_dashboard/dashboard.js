"use strict";
// scans.anonshield.org dashboard — fetches /api/v1/* and renders KPIs, hosts,
// scans-over-time bar chart, severity pie, and the top containers table.

const API = (location.pathname.endsWith("/") ? "" : "/") + "api/v1";
const REFRESH_MS = 30_000;
const PAGE_SIZE = 100;

const fmt = (n) => n == null ? "–" : Number(n).toLocaleString("pt-BR");
const fmtBig = (n) => {
  if (n == null) return "–";
  const v = Number(n);
  if (v >= 1e9) return (v/1e9).toFixed(1) + " bi";
  if (v >= 1e6) return (v/1e6).toFixed(1) + " mi";
  if (v >= 1e3) return (v/1e3).toFixed(1) + " k";
  return v.toString();
};
const fmtPct = (a, b) => (b > 0 ? ((a/b)*100).toFixed(2) + "%" : "–");
const fmtTs = (iso) => {
  if (!iso) return "–";
  const d = new Date(iso);
  return d.toLocaleString("pt-BR", { hour12: false });
};
const ago = (iso) => {
  if (!iso) return "";
  const s = (Date.now() - new Date(iso).getTime()) / 1000;
  if (s < 60) return Math.floor(s) + " s atrás";
  if (s < 3600) return Math.floor(s/60) + " min atrás";
  if (s < 86400) return Math.floor(s/3600) + " h atrás";
  return Math.floor(s/86400) + " d atrás";
};

let containersState = { all: [], filtered: [], offset: 0, n_total: 0, generated_at: null };
let charts = { timeline: null, severity: null, scannerSev: null, scannerCov: null, hist: null, topVuln: null };

const SCANNERS = ["syft", "trivy", "grype", "osv", "dockle", "trufflehog"];
const SEV_COLORS = { c: "#eb5757", h: "#f28b30", m: "#f2c94c", l: "#56a8ff", i: "#8b95a8", u: "#555" };
const SEV_LABELS = { c: "crítica", h: "alta", m: "média", l: "baixa", i: "info", u: "desc." };

async function api(path) {
  const r = await fetch(API + path, { cache: "no-store" });
  if (!r.ok) throw new Error(`${path} → HTTP ${r.status}`);
  return r.json();
}

// ── KPIs ────────────────────────────────────────────────────────────────────
async function renderKpis() {
  const live = await api("/dit-live").catch(() => null);
  const q = live?.queue || {};
  const total = q.total || 0;
  const done = q.done || 0;
  const pending = q.pending || 0;
  const skipped = q.skipped || 0;
  const failed = q.failed || 0;
  const findings = q.findings || 0;
  const rate = live?.rate_per_min;

  const cards = [
    { lbl: "containers escaneados", val: fmt(done), sub: fmtPct(done, total) + " de " + fmt(total), klass: "pct" },
    { lbl: "pendentes na fila", val: fmt(pending), sub: skipped > 0 ? `${fmt(skipped)} já feitos · ${fmt(failed)} falhas` : `${fmt(failed)} falhas` },
    { lbl: "findings consolidados", val: fmtBig(findings), sub: "soma bruta dos relatórios" },
    { lbl: "ritmo atual", val: (rate == null ? "–" : rate.toFixed(2)) + "/min", sub: "últimas amostras de 15 min", klass: "rate" },
  ];
  document.getElementById("kpis").innerHTML = cards.map(c =>
    `<div class="kpi"><div class="lbl">${c.lbl}</div><div class="val ${c.klass||''}">${c.val}</div><div class="sub">${c.sub||''}</div></div>`
  ).join("");
}

// ── máquinas ────────────────────────────────────────────────────────────────
async function renderHosts() {
  const data = await api("/machines").catch(() => ({ hosts: [] }));
  const hosts = (data.hosts || []).slice().sort((a, b) => (a.host||'').localeCompare(b.host||''));
  document.getElementById("hosts").innerHTML = hosts.map(h => {
    const onlineCls = h.online ? "online" : "";
    const role = h.role || "worker";
    const wk = h.workers_alive ?? "–";
    const load = h.load1 == null ? "–" : Number(h.load1).toFixed(2);
    const ramGb = h.ram_avail_mb == null ? "–" : (h.ram_avail_mb/1024).toFixed(1) + " GB";
    return `<div class="host ${onlineCls}">
      <div class="dot"></div>
      <h3>${h.host || '?'}</h3>
      <div class="role">${role}</div>
      <div class="stats">
        <div><span>workers</span><b>${wk}</b></div>
        <div><span>load 1m</span><b>${load}</b></div>
        <div><span>ram livre</span><b>${ramGb}</b></div>
        <div><span>status</span><b style="color:${h.online ? 'var(--ok)' : 'var(--err)'}">${h.online ? 'online' : 'offline'}</b></div>
      </div>
    </div>`;
  }).join("");
}

// ── gráfico de timeline ─────────────────────────────────────────────────────
async function renderTimeline() {
  const data = await api("/queue/timeline?bucket_minutes=60&hours=48").catch(() => null);
  if (!data) return;
  const labels = data.buckets.map(b => {
    const d = new Date(b.ts);
    return d.getUTCHours() + "h " + (d.getUTCMonth()+1) + "/" + d.getUTCDate();
  });
  const done = data.buckets.map(b => b.done);
  const ctx = document.getElementById("chart-timeline").getContext("2d");
  if (charts.timeline) charts.timeline.destroy();
  charts.timeline = new Chart(ctx, {
    type: "bar",
    data: { labels, datasets: [{ label: "scans/h", data: done, backgroundColor: "rgba(93,200,255,.7)", borderColor: "rgba(93,200,255,1)", borderWidth: 1 }] },
    options: {
      responsive: true, maintainAspectRatio: false,
      plugins: { legend: { display: false } },
      scales: {
        x: { ticks: { color: "#8b95a8", maxRotation: 0, autoSkipPadding: 24 }, grid: { display: false } },
        y: { ticks: { color: "#8b95a8" }, grid: { color: "#222a3a" } }
      }
    }
  });
}

// ── gráficos baseados em containersState.all ────────────────────────────────
function renderScannerCharts() {
  // findings por scanner empilhados por severidade
  const perScanner = {}; // {scanner: {c,h,m,l,i,u, total, ok, runs}}
  for (const sc of SCANNERS) perScanner[sc] = { c:0,h:0,m:0,l:0,i:0,u:0, total:0, ok:0, runs:0 };
  for (const ct of containersState.all) {
    for (const sc of SCANNERS) {
      const v = ct.by_scanner?.[sc];
      if (!v) continue;
      perScanner[sc].runs += 1;
      if ((v.status||"").startsWith("ok")) perScanner[sc].ok += 1;
      for (const k of ["c","h","m","l","i","u"]) perScanner[sc][k] += v[k]||0;
      perScanner[sc].total += v.n||0;
    }
  }
  const ctx = document.getElementById("chart-scanner-sev")?.getContext("2d");
  if (!ctx) return;
  if (charts.scannerSev) charts.scannerSev.destroy();
  charts.scannerSev = new Chart(ctx, {
    type: "bar",
    data: {
      labels: SCANNERS,
      datasets: ["c","h","m","l","i","u"].map(k => ({
        label: SEV_LABELS[k],
        data: SCANNERS.map(s => perScanner[s][k]),
        backgroundColor: SEV_COLORS[k],
      })),
    },
    options: {
      responsive: true, maintainAspectRatio: false,
      plugins: { legend: { position: "bottom", labels: { color: "#e6e8ee", boxWidth: 11, font:{size:11} } } },
      scales: {
        x: { stacked: true, ticks: { color: "#8b95a8" }, grid: { display:false } },
        y: { stacked: true, ticks: { color: "#8b95a8" }, grid: { color: "#222a3a" } },
      },
    },
  });

  // cobertura: % de runs com status ok* para cada scanner
  const ctx2 = document.getElementById("chart-scanner-cov")?.getContext("2d");
  if (!ctx2) return;
  if (charts.scannerCov) charts.scannerCov.destroy();
  const pct = SCANNERS.map(s => perScanner[s].runs ? (perScanner[s].ok/perScanner[s].runs)*100 : 0);
  charts.scannerCov = new Chart(ctx2, {
    type: "bar",
    data: { labels: SCANNERS, datasets: [{ label: "% ok", data: pct, backgroundColor: "rgba(111,207,151,.65)", borderColor: "#6fcf97", borderWidth: 1 }] },
    options: {
      indexAxis: "y",
      responsive: true, maintainAspectRatio: false,
      plugins: { legend: { display:false }, tooltip:{ callbacks:{ label:(c)=>`${c.parsed.x.toFixed(1)}%` } } },
      scales: {
        x: { min:0, max:100, ticks: { color:"#8b95a8", callback:(v)=>v+"%" }, grid:{ color:"#222a3a" } },
        y: { ticks: { color: "#8b95a8" }, grid: { display:false } },
      },
    },
  });
}

function renderDistributions() {
  // histograma de findings (merged) por container
  const bins = [
    { l: "0", min: 0, max: 0 },
    { l: "1–10", min: 1, max: 10 },
    { l: "11–100", min: 11, max: 100 },
    { l: "101–500", min: 101, max: 500 },
    { l: "501–2k", min: 501, max: 2000 },
    { l: "2k–10k", min: 2001, max: 10000 },
    { l: ">10k", min: 10001, max: Infinity },
  ];
  const counts = bins.map(()=>0);
  for (const ct of containersState.all) {
    const n = ct.merged || 0;
    for (let i = 0; i < bins.length; i++) if (n >= bins[i].min && n <= bins[i].max) { counts[i]++; break; }
  }
  const ctx = document.getElementById("chart-hist")?.getContext("2d");
  if (ctx) {
    if (charts.hist) charts.hist.destroy();
    charts.hist = new Chart(ctx, {
      type: "bar",
      data: { labels: bins.map(b=>b.l), datasets: [{ data: counts, backgroundColor: "rgba(93,200,255,.65)", borderColor: "#5dc8ff", borderWidth: 1 }] },
      options: { responsive:true, maintainAspectRatio:false, plugins:{ legend:{display:false} },
        scales:{ x:{ticks:{color:"#8b95a8"}, grid:{display:false}}, y:{ticks:{color:"#8b95a8"}, grid:{color:"#222a3a"}} } },
    });
  }

  // top 15 mais vulneráveis (críticos + altos)
  const ranked = containersState.all.map(ct => {
    let ch = 0;
    for (const v of Object.values(ct.by_scanner||{})) ch += (v.c||0) + (v.h||0);
    return { image: ct.image, ch };
  }).filter(x => x.ch > 0).sort((a,b)=>b.ch-a.ch).slice(0,15);
  const ctx2 = document.getElementById("chart-top-vuln")?.getContext("2d");
  if (ctx2) {
    if (charts.topVuln) charts.topVuln.destroy();
    const labels = ranked.map(r => r.image.length > 50 ? "…" + r.image.slice(-48) : r.image);
    charts.topVuln = new Chart(ctx2, {
      type: "bar",
      data: { labels, datasets: [{ data: ranked.map(r=>r.ch), backgroundColor: "rgba(235,87,87,.65)", borderColor: "#eb5757", borderWidth: 1 }] },
      options: { indexAxis:"y", responsive:true, maintainAspectRatio:false, plugins:{ legend:{display:false} },
        scales:{ x:{ticks:{color:"#8b95a8"}, grid:{color:"#222a3a"}}, y:{ticks:{color:"#8b95a8", font:{size:10, family:"ui-monospace,monospace"}}, grid:{display:false}} } },
    });
  }
}

// ── pie de severidade ───────────────────────────────────────────────────────
function renderSeverity() {
  const sums = { c: 0, h: 0, m: 0, l: 0, i: 0, u: 0 };
  for (const ct of containersState.all) {
    for (const sc of Object.values(ct.by_scanner || {})) {
      sums.c += sc.c||0; sums.h += sc.h||0; sums.m += sc.m||0;
      sums.l += sc.l||0; sums.i += sc.i||0; sums.u += sc.u||0;
    }
  }
  const ctx = document.getElementById("chart-severity").getContext("2d");
  if (charts.severity) charts.severity.destroy();
  charts.severity = new Chart(ctx, {
    type: "doughnut",
    data: {
      labels: ["crítica","alta","média","baixa","info","desconhecida"],
      datasets: [{ data: [sums.c, sums.h, sums.m, sums.l, sums.i, sums.u],
                   backgroundColor: ["#eb5757","#f28b30","#f2c94c","#56a8ff","#8b95a8","#555"],
                   borderColor: "#161a22", borderWidth: 2 }]
    },
    options: {
      responsive: true, maintainAspectRatio: false,
      plugins: { legend: { position: "right", labels: { color: "#e6e8ee", boxWidth: 12, font: { size: 11 } } } },
      cutout: "55%"
    }
  });
}

// ── tabela ──────────────────────────────────────────────────────────────────
function applyFilter(qstr) {
  const q = (qstr || "").trim().toLowerCase();
  containersState.filtered = q
    ? containersState.all.filter(c => c.image.toLowerCase().includes(q))
    : containersState.all;
  containersState.offset = 0;
  renderTable();
}

function renderTable() {
  const items = containersState.filtered.slice(containersState.offset, containersState.offset + PAGE_SIZE);
  const tbody = document.querySelector("#tbl tbody");
  tbody.innerHTML = items.map(ct => {
    const sc = (name) => {
      const v = ct.by_scanner?.[name];
      if (!v) return `<td class="num">–</td>`;
      const sev = `<span class="sev">${["c","h","m","l","i","u"].map(k => v[k] > 0 ? `<span class="${k}" title="${k}:${v[k]}"></span>` : "").join("")}</span>`;
      return `<td class="num"><b>${fmt(v.n)}</b> ${sev}</td>`;
    };
    return `<tr>
      <td class="num">${ct.rank}</td>
      <td class="num">${fmtBig(ct.exposure)}</td>
      <td class="img" title="${ct.image}">${ct.image}</td>
      <td class="num"><b>${fmt(ct.merged)}</b></td>
      ${sc("syft")}${sc("trivy")}${sc("grype")}${sc("osv")}${sc("dockle")}${sc("trufflehog")}
    </tr>`;
  }).join("");
  document.getElementById("count").textContent =
    `${fmt(containersState.filtered.length)} de ${fmt(containersState.all.length)} containers`;
  const start = containersState.offset + 1;
  const end = Math.min(containersState.offset + PAGE_SIZE, containersState.filtered.length);
  document.getElementById("page-info").textContent =
    containersState.filtered.length ? `mostrando ${fmt(start)}–${fmt(end)}` : "sem resultados";
  document.getElementById("prev").disabled = containersState.offset === 0;
  document.getElementById("next").disabled = end >= containersState.filtered.length;
}

async function loadContainers() {
  // pega o snapshot inteiro de uma vez (cache no servidor faz isso rapido)
  const data = await api("/containers?limit=100000").catch(() => null);
  if (!data) return;
  containersState.all = data.containers;
  containersState.n_total = data.n_total_scanned;
  containersState.generated_at = data.generated_at;
  applyFilter(document.getElementById("q").value);
  renderSeverity();
  renderScannerCharts();
  renderDistributions();
}

// ── eventos ─────────────────────────────────────────────────────────────────
document.getElementById("q").addEventListener("input", (e) => applyFilter(e.target.value));
document.getElementById("prev").addEventListener("click", () => { containersState.offset = Math.max(0, containersState.offset - PAGE_SIZE); renderTable(); });
document.getElementById("next").addEventListener("click", () => { containersState.offset += PAGE_SIZE; renderTable(); });

async function renderRecent() {
  const recent = await api("/queue/recent?limit=30").catch(() => []);
  document.getElementById("recent-list").innerHTML = recent.map(r => {
    const when = r.finished_at ? ago(new Date(r.finished_at * 1000).toISOString()) : "—";
    return `<div class="item">
      <div class="img" title="${r.image}">${r.image}</div>
      <div class="meta"><span class="findings">${fmt(r.findings)}</span> findings · <span>${when}</span></div>
    </div>`;
  }).join("");
}

async function refreshAll() {
  document.getElementById("last-update").textContent = "atualizando…";
  await Promise.allSettled([renderKpis(), renderHosts(), renderTimeline(), renderRecent(), loadContainers()]);
  document.getElementById("last-update").textContent = "última atualização: agora · F5 recarrega";
}

refreshAll();
setInterval(refreshAll, REFRESH_MS);
