#!/bin/bash

set -e

PROFILE_NAME=${1:-base}
PROFILES_DIR="profiles"
SERVER_ADDR="localhost:8080"
PPROF_PORT="8080"
PPROF_URL="http://localhost:$PPROF_PORT/debug/pprof"

mkdir -p "$PROFILES_DIR"

echo "=== Профилирование сервера ==="
echo "Профиль будет сохранен в: $PROFILES_DIR/$PROFILE_NAME.pprof"
echo "pprof URL: $PPROF_URL"

if ! command -v go &> /dev/null; then
    echo "Ошибка: Go не установлен"
    exit 1
fi

generate_load() {
    echo "Генерация нагрузки..."
    
    if ! command -v curl &> /dev/null; then
        echo "Ошибка: curl не установлен"
        exit 1
    fi
    
    echo "Запуск нагрузки в фоне..."
    LOAD_PIDS=""
    TOTAL_REQUESTS=2000
    BATCH_SIZE=50
    BATCHES=$((TOTAL_REQUESTS / BATCH_SIZE))
    
    for batch in $(seq 1 $BATCHES); do
        for i in $(seq $(( (batch - 1) * BATCH_SIZE + 1 )) $(( batch * BATCH_SIZE ))); do
            curl -s -X POST "http://$SERVER_ADDR/update/gauge/test_metric_$i/123.45" > /dev/null 2>&1 &
            LOAD_PIDS="$LOAD_PIDS $!"
        done
        sleep 0.1
    done
    
    echo "Нагрузка запущена ($TOTAL_REQUESTS запросов)..."
}

capture_profile() {
    echo "Снятие профиля памяти..."
    
    curl -s "$PPROF_URL/heap" > "$PROFILES_DIR/$PROFILE_NAME.pprof" || {
        echo "Ошибка: не удалось снять профиль. Убедитесь, что:"
        echo "1. Сервер запущен с флагом -pprof"
        echo "2. pprof endpoint доступен по адресу $PPROF_URL"
        exit 1
    }
    
    echo "Профиль сохранен: $PROFILES_DIR/$PROFILE_NAME.pprof"
}

echo ""
echo "Начинаем генерацию нагрузки..."
generate_load

echo "Снимаем профиль во время нагрузки..."
capture_profile

echo "Останавливаем нагрузку..."
if [ -n "$LOAD_PIDS" ]; then
    for pid in $LOAD_PIDS; do
        kill $pid 2>/dev/null || true
    done
    wait 2>/dev/null || true
fi
echo "Нагрузка остановлена"

echo ""
echo "=== Профиль успешно снят ==="
