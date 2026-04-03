#!/bin/bash
# benchmark_crawler.sh — testa empiricamente workers e PAGE_CONCURRENCY
# Mede repos/min adicionados ao MongoDB para cada configuração.
#
# Uso: bash automation/benchmark_crawler.sh
# Duração total: ~30 min (5 configs × ~6 min cada)

set -e
cd "$(dirname "$0")/.."

MEASURE_SECS=180   # 3 min de medição por config
COMPILE_WAIT=90    # tempo de espera para compilação do Go na primeira rodada
RESULTS_FILE="benchmark_results_$(date +%Y%m%d_%H%M%S).txt"

mongo_count() {
    docker exec ditector_mongo mongosh --quiet \
        --eval 'db.getSiblingDB("dockerhub_data").repositories_data.countDocuments()' 2>/dev/null || echo 0
}

wait_for_crawler() {
    local max=120
    local i=0
    echo -n "    Aguardando crawler conectar ao MongoDB..."
    while ! docker logs ditector_crawler 2>&1 | grep -q "Connect to MongoDB"; do
        sleep 3; i=$((i+3))
        if [ $i -ge $max ]; then echo " timeout"; return 1; fi
        echo -n "."
    done
    echo " OK"
}

run_config() {
    local label="$1"
    local workers="$2"
    local page_conc="$3"

    echo ""
    echo "========================================"
    echo "CONFIG: $label  (workers=$workers, PAGE_CONCURRENCY=$page_conc)"
    echo "========================================"

    # Reinicia o container com a nova config
    docker-compose stop crawler 2>/dev/null
    docker-compose rm -f crawler 2>/dev/null
    WORKERS=$workers PAGE_CONCURRENCY=$page_conc docker-compose up -d crawler 2>/dev/null

    # Aguarda compilação + conexão
    echo "    Compilando e iniciando (aguardando ${COMPILE_WAIT}s)..."
    sleep $COMPILE_WAIT
    wait_for_crawler || { echo "    ERRO: crawler não iniciou"; return; }

    # Deixa estabilizar 15s após conectar
    sleep 15

    # Mede por MEASURE_SECS
    local t0=$(date +%s)
    local c0=$(mongo_count)
    echo "    [t=0s] repos no MongoDB: $c0"

    # Amostras a cada 30s
    local samples=()
    for i in 30 60 90 120 150 180; do
        sleep 30
        local c=$(mongo_count)
        local elapsed=$(( $(date +%s) - t0 ))
        local added=$((c - c0))
        local rate=$(echo "scale=1; $added * 60 / $elapsed" | bc 2>/dev/null || echo "?")
        echo "    [t=${elapsed}s] repos: $c  (+${added})  taxa: ${rate} repos/min"
        samples+=("$rate")
    done

    local c1=$(mongo_count)
    local total_added=$((c1 - c0))
    local rate_avg=$(echo "scale=1; $total_added * 60 / $MEASURE_SECS" | bc 2>/dev/null || echo "?")

    echo "    RESULTADO: +${total_added} repos em ${MEASURE_SECS}s = ${rate_avg} repos/min"
    echo "$label | workers=$workers | page_conc=$page_conc | repos/min=$rate_avg | total_added=$total_added" >> "$RESULTS_FILE"
}

echo "Benchmark Crawler DITector" > "$RESULTS_FILE"
echo "Data: $(date)" >> "$RESULTS_FILE"
echo "GPU1 — $(docker exec ditector_mongo mongosh --quiet --eval 'db.version()' 2>/dev/null)" >> "$RESULTS_FILE"
echo "" >> "$RESULTS_FILE"

# Primeira rodada compila o binário — as seguintes reutilizam se volume for o mesmo.
# O binário fica em /tmp/ditector (dentro do container) — se o container for removido, recompila.
# Para evitar recompile: usamos um volume extra ou deixamos o container parado (não removido).
# Aqui: compilamos em /app/ditector_bin que é montado no volume .:/app

# Configs a testar: workers × page_concurrency
run_config "W25_PC8"   25  8
run_config "W50_PC8"   50  8
run_config "W100_PC8" 100  8
run_config "W50_PC4"   50  4
run_config "W50_PC16"  50 16

echo ""
echo "========================================"
echo "RESULTADOS FINAIS"
echo "========================================"
cat "$RESULTS_FILE"
echo ""
echo "Arquivo salvo em: $RESULTS_FILE"

# Restaura config default
WORKERS=50 PAGE_CONCURRENCY=8 docker-compose up -d crawler 2>/dev/null
echo "Crawler restaurado com config padrão (workers=50, PAGE_CONCURRENCY=8)"
