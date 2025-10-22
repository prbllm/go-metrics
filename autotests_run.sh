#!/bin/bash

# Скрипт для запуска автотестов локально
# Использование: ./autotests_run <номер_итерации>
# Пример: ./autotests_run 5

set -e

# Проверяем аргумент
if [ $# -eq 0 ]; then
    echo "Использование: $0 <номер_итерации>"
    echo "Пример: $0 5"
    exit 1
fi

ITERATION=$1

# Проверяем, что номер итерации валидный
if ! [[ "$ITERATION" =~ ^[0-9]+$ ]] || [ "$ITERATION" -lt 1 ] || [ "$ITERATION" -gt 14 ]; then
    echo "Ошибка: номер итерации должен быть от 1 до 14"
    exit 1
fi

echo "🚀 Запуск автотестов для итерации $ITERATION"
echo "================================================"

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

# Запускаем тесты в зависимости от итерации
echo "🧪 Запуск тестов..."
echo ""

# Итерация 1
if [ "$ITERATION" -ge 1 ]; then
    echo "📋 Тест итерации 1..."
    ./metricstest_v2-darwin-amd64 -test.v -test.run=^TestIteration1$ \
        -binary-path=cmd/server/server
    echo "✅ Итерация 1 завершена"
    echo ""
fi

# Итерация 2
if [ "$ITERATION" -ge 2 ]; then
    echo "📋 Тест итерации 2..."
    ./metricstest_v2-darwin-amd64 -test.v -test.run=^TestIteration2[AB]*$ \
        -source-path=. \
        -agent-binary-path=cmd/agent/agent
    echo "✅ Итерация 2 завершена"
    echo ""
fi

# Итерация 3
if [ "$ITERATION" -ge 3 ]; then
    echo "📋 Тест итерации 3..."
    ./metricstest_v2-darwin-amd64 -test.v -test.run=^TestIteration3[AB]*$ \
        -source-path=. \
        -agent-binary-path=cmd/agent/agent \
        -binary-path=cmd/server/server
    echo "✅ Итерация 3 завершена"
    echo ""
fi

# Итерации 4-9 (с портом)
if [ "$ITERATION" -ge 4 ]; then
    for i in $(seq 4 $((ITERATION < 10 ? ITERATION : 9))); do
        echo "📋 Тест итерации $i..."
        SERVER_PORT=$(get_random_port)
        ADDRESS="localhost:${SERVER_PORT}"
        TEMP_FILE=$(get_temp_file)
        
        ./metricstest_v2-darwin-amd64 -test.v -test.run=^TestIteration${i}$ \
            -agent-binary-path=cmd/agent/agent \
            -binary-path=cmd/server/server \
            -server-port=$SERVER_PORT \
            -source-path=.
        
        # Удаляем временный файл
        rm -f "$TEMP_FILE"
        echo "✅ Итерация $i завершена"
        echo ""
    done
fi

# Итерация 9 (с файловым хранилищем)
if [ "$ITERATION" -ge 9 ]; then
    echo "📋 Тест итерации 9 (с файловым хранилищем)..."
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
    echo "✅ Итерация 9 завершена"
    echo ""
fi

# Итерации 10-14 (с базой данных)
if [ "$ITERATION" -ge 10 ]; then
    echo "⚠️  Внимание: Итерации 10-14 требуют PostgreSQL"
    echo "Убедитесь, что PostgreSQL запущен и доступен по адресу localhost:5432"
    echo "База данных: praktikum, пользователь: postgres, пароль: postgres"
    echo ""
    
    for i in $(seq 10 $ITERATION); do
        echo "📋 Тест итерации $i (с базой данных)..."
        SERVER_PORT=$(get_random_port)
        ADDRESS="localhost:${SERVER_PORT}"
        TEMP_FILE=$(get_temp_file)
        
        # Специальная обработка для итерации 10
        if [ "$i" -eq 10 ]; then
            ./metricstest_v2-darwin-amd64 -test.v -test.run=^TestIteration10[AB]$ \
                -agent-binary-path=cmd/agent/agent \
                -binary-path=cmd/server/server \
                -database-dsn='postgres://postgres:postgres@localhost:5432/praktikum?sslmode=disable' \
                -server-port=$SERVER_PORT \
                -source-path=.
        else
            ./metricstest_v2-darwin-amd64 -test.v -test.run=^TestIteration${i}$ \
                -agent-binary-path=cmd/agent/agent \
                -binary-path=cmd/server/server \
                -database-dsn='postgres://postgres:postgres@localhost:5432/praktikum?sslmode=disable' \
                -server-port=$SERVER_PORT \
                -source-path=.
        fi
        
        # Специальная обработка для итерации 14 (с ключом)
        if [ "$i" -eq 14 ]; then
            echo "📋 Тест итерации 14 (с ключом)..."
            ./metricstest_v2-darwin-amd64 -test.v -test.run=^TestIteration14$ \
                -agent-binary-path=cmd/agent/agent \
                -binary-path=cmd/server/server \
                -database-dsn='postgres://postgres:postgres@localhost:5432/praktikum?sslmode=disable' \
                -key="${TEMP_FILE}" \
                -server-port=$SERVER_PORT \
                -source-path=.
        fi
        
        rm -f "$TEMP_FILE"
        echo "✅ Итерация $i завершена"
        echo ""
    done
fi

# Race detection для итерации 14
if [ "$ITERATION" -ge 14 ]; then
    echo "📋 Тест race detection для итерации 14..."
    go test -v -race ./...
    echo "✅ Race detection завершен"
    echo ""
fi

echo "🎉 Все тесты для итерации $ITERATION завершены успешно!"
