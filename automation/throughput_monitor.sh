#!/bin/bash
# throughput_monitor.sh
# Rationale: Log real-time discovery metrics to a CSV for performance analysis.

LOG_FILE="throughput_metrics.csv"
DB_NAME="dockerhub_data"

# Header
if [ ! -f "$LOG_FILE" ]; then
    echo "timestamp,repos_total,diff_minute,tasks_pending,tasks_processing" > "$LOG_FILE"
fi

echo ">>> Monitoring started. Logging to $LOG_FILE"

# Initial count
PREV=$(docker exec ditector_mongo mongosh --quiet --eval "db.getSiblingDB('$DB_NAME').repositories_data.countDocuments()")

while true; do
    sleep 60
    
    # Collect Metrics
    STATS=$(docker exec ditector_mongo mongosh --quiet --eval "
        var db = db.getSiblingDB('$DB_NAME');
        var curr = db.repositories_data.countDocuments();
        var pend = db.crawler_keywords.countDocuments({status:'pending'});
        var proc = db.crawler_keywords.countDocuments({status:'processing'});
        print(curr + ',' + pend + ',' + proc);
    ")
    
    CURR=$(echo $STATS | cut -d',' -f1)
    PEND=$(echo $STATS | cut -d',' -f2)
    PROC=$(echo $STATS | cut -d',' -f3)
    
    DIFF=$((CURR - PREV))
    TIME=$(date '+%Y-%m-%d %H:%M:%S')
    
    # Log to CSV
    echo "$TIME,$CURR,$DIFF,$PEND,$PROC" >> "$LOG_FILE"
    
    # Log to Console
    echo "[$TIME] +$DIFF repos | Queue: $PEND pending | Workers: $PROC"
    
    PREV=$CURR
done
