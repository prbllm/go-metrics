#!/bin/bash

# Скрипт для запуска автотестов локально
# Использование: ./autotests_run <номер_итерации>
# Пример: ./autotests_run 5
# Запускает все итерации до указанной (как в CI)

set -e

# Проверяем аргумент
if [ $# -eq 0 ]; then
    echo "Использование: $0 <номер_итерации>"
    echo "Пример: $0 5"
    echo "Запускает все итерации до указанной (как в CI)"
    exit 1
fi

ITERATION=$1

# Проверяем, что номер итерации валидный
if ! [[ "$ITERATION" =~ ^[0-9]+$ ]] || [ "$ITERATION" -lt 1 ] || [ "$ITERATION" -gt 14 ]; then
    echo "Ошибка: номер итерации должен быть от 1 до 14"
    exit 1
fi

echo "🚀 Запуск автотестов для итераций 1-$ITERATION (как в CI)"
echo "================================================"

# Очищаем тестовую базу данных
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [ -f "$SCRIPT_DIR/cleanup_test_db.sh" ]; then
    echo ""
    # Используем || true, чтобы не останавливать процесс при ошибках очистки
    "$SCRIPT_DIR/cleanup_test_db.sh" || true
    echo ""
fi

# Собираем бинарники
echo "📦 Сборка бинарников..."

echo "  - Сборка сервера..."
go build -buildvcs=false -o cmd/server/server ./cmd/server
if [ $? -ne 0 ]; then
    echo "❌ Ошибка сборки сервера"
    exit 1
fi

echo "  - Сборка агента..."
go build -buildvcs=false -o cmd/agent/agent ./cmd/agent
if [ $? -ne 0 ]; then
    echo "❌ Ошибка сборки агента"
    exit 1
fi

echo "✅ Бинарники собраны успешно"
echo ""

# Функция для получения случайного порта
get_random_port() {
    # Используем встроенную команду для получения случайного порта
    # В macOS можно использовать lsof для проверки занятых портов
    while true; do
        port=$((RANDOM % 10000 + 10000))
        if ! lsof -i :$port > /dev/null 2>&1; then
            echo $port
            return
        fi
    done
}

# Функция для создания временного файла
get_temp_file() {
    mktemp /tmp/metricstest_XXXXXX
}

# Запускаем тесты точно как в CI
echo "🧪 Запуск тестов..."
echo ""

# Code increment #1
if [ "$ITERATION" -ge 1 ]; then
    echo "📋 Code increment #1..."
    ./metricstest_v2-darwin-amd64 -test.v -test.run=^TestIteration1$ \
        -binary-path=cmd/server/server
    echo "✅ Code increment #1 завершен"
    echo ""
fi

# Code increment #2
if [ "$ITERATION" -ge 2 ]; then
    echo "📋 Code increment #2..."
    ./metricstest_v2-darwin-amd64 -test.v -test.run=^TestIteration2[AB]*$ \
        -source-path=. \
        -agent-binary-path=cmd/agent/agent
    echo "✅ Code increment #2 завершен"
    echo ""
fi

# Code increment #3
if [ "$ITERATION" -ge 3 ]; then
    echo "📋 Code increment #3..."
    ./metricstest_v2-darwin-amd64 -test.v -test.run=^TestIteration3[AB]*$ \
        -source-path=. \
        -agent-binary-path=cmd/agent/agent \
        -binary-path=cmd/server/server
    echo "✅ Code increment #3 завершен"
    echo ""
fi

# Code increment #4
if [ "$ITERATION" -ge 4 ]; then
    echo "📋 Code increment #4..."
    SERVER_PORT=$(get_random_port)
    ADDRESS="localhost:${SERVER_PORT}"
    TEMP_FILE=$(get_temp_file)
    ./metricstest_v2-darwin-amd64 -test.v -test.run=^TestIteration4$ \
        -agent-binary-path=cmd/agent/agent \
        -binary-path=cmd/server/server \
        -server-port=$SERVER_PORT \
        -source-path=.
    rm -f "$TEMP_FILE"
    echo "✅ Code increment #4 завершен"
    echo ""
fi

# Code increment #5
if [ "$ITERATION" -ge 5 ]; then
    echo "📋 Code increment #5..."
    SERVER_PORT=$(get_random_port)
    ADDRESS="localhost:${SERVER_PORT}"
    TEMP_FILE=$(get_temp_file)
    ./metricstest_v2-darwin-amd64 -test.v -test.run=^TestIteration5$ \
        -agent-binary-path=cmd/agent/agent \
        -binary-path=cmd/server/server \
        -server-port=$SERVER_PORT \
        -source-path=.
    rm -f "$TEMP_FILE"
    echo "✅ Code increment #5 завершен"
    echo ""
fi

# Code increment #6
if [ "$ITERATION" -ge 6 ]; then
    echo "📋 Code increment #6..."
    SERVER_PORT=$(get_random_port)
    ADDRESS="localhost:${SERVER_PORT}"
    TEMP_FILE=$(get_temp_file)
    ./metricstest_v2-darwin-amd64 -test.v -test.run=^TestIteration6$ \
        -agent-binary-path=cmd/agent/agent \
        -binary-path=cmd/server/server \
        -server-port=$SERVER_PORT \
        -source-path=.
    rm -f "$TEMP_FILE"
    echo "✅ Code increment #6 завершен"
    echo ""
fi

# Code increment #7
if [ "$ITERATION" -ge 7 ]; then
    echo "📋 Code increment #7..."
    SERVER_PORT=$(get_random_port)
    ADDRESS="localhost:${SERVER_PORT}"
    TEMP_FILE=$(get_temp_file)
    ./metricstest_v2-darwin-amd64 -test.v -test.run=^TestIteration7$ \
        -agent-binary-path=cmd/agent/agent \
        -binary-path=cmd/server/server \
        -server-port=$SERVER_PORT \
        -source-path=.
    rm -f "$TEMP_FILE"
    echo "✅ Code increment #7 завершен"
    echo ""
fi

# Code increment #8
if [ "$ITERATION" -ge 8 ]; then
    echo "📋 Code increment #8..."
    SERVER_PORT=$(get_random_port)
    ADDRESS="localhost:${SERVER_PORT}"
    TEMP_FILE=$(get_temp_file)
    ./metricstest_v2-darwin-amd64 -test.v -test.run=^TestIteration8$ \
        -agent-binary-path=cmd/agent/agent \
        -binary-path=cmd/server/server \
        -server-port=$SERVER_PORT \
        -source-path=.
    rm -f "$TEMP_FILE"
    echo "✅ Code increment #8 завершен"
    echo ""
fi

# Code increment #9
if [ "$ITERATION" -ge 9 ]; then
    echo "📋 Code increment #9..."
    SERVER_PORT=$(get_random_port)
    ADDRESS="localhost:${SERVER_PORT}"
    TEMP_FILE=$(get_temp_file)
    ./metricstest_v2-darwin-amd64 -test.v -test.run=^TestIteration9$ \
        -agent-binary-path=cmd/agent/agent \
        -binary-path=cmd/server/server \
        -file-storage-path=$TEMP_FILE \
        -server-port=$SERVER_PORT \
        -source-path=.
    rm -f "$TEMP_FILE"
    echo "✅ Code increment #9 завершен"
    echo ""
fi

# Code increment #10
if [ "$ITERATION" -ge 10 ]; then
    echo "📋 Code increment #10..."
    SERVER_PORT=$(get_random_port)
    ADDRESS="localhost:${SERVER_PORT}"
    TEMP_FILE=$(get_temp_file)
    ./metricstest_v2-darwin-amd64 -test.v -test.run=^TestIteration10[AB]$ \
        -agent-binary-path=cmd/agent/agent \
        -binary-path=cmd/server/server \
        -database-dsn='postgres://postgres:postgres@localhost:5432/praktikum?sslmode=disable' \
        -server-port=$SERVER_PORT \
        -source-path=.
    rm -f "$TEMP_FILE"
    echo "✅ Code increment #10 завершен"
    echo ""
fi

# Code increment #11
if [ "$ITERATION" -ge 11 ]; then
    echo "📋 Code increment #11..."
    SERVER_PORT=$(get_random_port)
    ADDRESS="localhost:${SERVER_PORT}"
    TEMP_FILE=$(get_temp_file)
    ./metricstest_v2-darwin-amd64 -test.v -test.run=^TestIteration11$ \
        -agent-binary-path=cmd/agent/agent \
        -binary-path=cmd/server/server \
        -database-dsn='postgres://postgres:postgres@localhost:5432/praktikum?sslmode=disable' \
        -server-port=$SERVER_PORT \
        -source-path=.
    rm -f "$TEMP_FILE"
    echo "✅ Code increment #11 завершен"
    echo ""
fi

# Code increment #12
if [ "$ITERATION" -ge 12 ]; then
    echo "📋 Code increment #12..."
    SERVER_PORT=$(get_random_port)
    ADDRESS="localhost:${SERVER_PORT}"
    TEMP_FILE=$(get_temp_file)
    ./metricstest_v2-darwin-amd64 -test.v -test.run=^TestIteration12$ \
        -agent-binary-path=cmd/agent/agent \
        -binary-path=cmd/server/server \
        -database-dsn='postgres://postgres:postgres@localhost:5432/praktikum?sslmode=disable' \
        -server-port=$SERVER_PORT \
        -source-path=.
    rm -f "$TEMP_FILE"
    echo "✅ Code increment #12 завершен"
    echo ""
fi

# Code increment #13
if [ "$ITERATION" -ge 13 ]; then
    echo "📋 Code increment #13..."
    SERVER_PORT=$(get_random_port)
    ADDRESS="localhost:${SERVER_PORT}"
    TEMP_FILE=$(get_temp_file)
    ./metricstest_v2-darwin-amd64 -test.v -test.run=^TestIteration13$ \
        -agent-binary-path=cmd/agent/agent \
        -binary-path=cmd/server/server \
        -database-dsn='postgres://postgres:postgres@localhost:5432/praktikum?sslmode=disable' \
        -server-port=$SERVER_PORT \
        -source-path=.
    rm -f "$TEMP_FILE"
    echo "✅ Code increment #13 завершен"
    echo ""
fi

# Code increment #14
if [ "$ITERATION" -ge 14 ]; then
    echo "📋 Code increment #14..."
    SERVER_PORT=$(get_random_port)
    ADDRESS="localhost:${SERVER_PORT}"
    TEMP_FILE=$(get_temp_file)
    ./metricstest_v2-darwin-amd64 -test.v -test.run=^TestIteration14$ \
        -agent-binary-path=cmd/agent/agent \
        -binary-path=cmd/server/server \
        -database-dsn='postgres://postgres:postgres@localhost:5432/praktikum?sslmode=disable' \
        -key="${TEMP_FILE}" \
        -server-port=$SERVER_PORT \
        -source-path=.
    rm -f "$TEMP_FILE"
    echo "✅ Code increment #14 завершен"
    echo ""
fi

# Code increment #14 (race detection)
if [ "$ITERATION" -ge 14 ]; then
    echo "📋 Code increment #14 (race detection)..."
    go test -v -race ./...
    echo "✅ Code increment #14 (race detection) завершен"
    echo ""
fi

echo "🎉 Все тесты для итераций 1-$ITERATION завершены успешно!"
