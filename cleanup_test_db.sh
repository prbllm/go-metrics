#!/bin/bash

# Скрипт для очистки тестовой базы данных от старых прогонов
# Очищает таблицу metrics в базе данных praktikum

# Не используем set -e, чтобы не останавливать весь процесс при ошибках очистки

# DSN из autotests_run.sh
DB_DSN="${DATABASE_DSN:-postgres://postgres:postgres@localhost:5432/praktikum?sslmode=disable}"

# Парсим DSN для получения параметров подключения
# Формат: postgres://user:password@host:port/database?sslmode=disable
parse_dsn() {
    local dsn="$1"
    
    # Убираем префикс postgres://
    dsn="${dsn#postgres://}"
    
    # Извлекаем user:password
    local user_pass="${dsn%%@*}"
    DB_USER="${user_pass%%:*}"
    DB_PASSWORD="${user_pass#*:}"
    
    # Извлекаем host:port/database?params
    local rest="${dsn#*@}"
    local host_port_db="${rest%%\?*}"
    
    # Извлекаем database
    DB_NAME="${host_port_db##*/}"
    
    # Извлекаем host:port
    local host_port="${host_port_db%/*}"
    DB_HOST="${host_port%%:*}"
    DB_PORT="${host_port##*:}"
    
    # Если порт не указан, используем дефолтный
    if [ "$DB_PORT" = "$DB_HOST" ]; then
        DB_PORT="5432"
    fi
}

echo "🧹 Очистка тестовой базы данных..."
echo "================================================"

# Парсим DSN
parse_dsn "$DB_DSN"

# Проверяем, что все параметры извлечены корректно
if [ -z "$DB_HOST" ] || [ -z "$DB_NAME" ] || [ -z "$DB_USER" ] || [ -z "$DB_PASSWORD" ]; then
    echo "⚠️  Предупреждение: Не удалось распарсить DSN. Пропускаем очистку."
    exit 0
fi

echo "  Host: $DB_HOST"
echo "  Port: $DB_PORT"
echo "  Database: $DB_NAME"
echo "  User: $DB_USER"
echo ""

# Проверяем доступность psql
if ! command -v psql &> /dev/null; then
    echo "⚠️  Предупреждение: psql не найден. Пропускаем очистку базы данных."
    exit 0
fi

# Экспортируем пароль для psql
export PGPASSWORD="$DB_PASSWORD"

# Проверяем подключение к БД
if ! psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "SELECT 1;" > /dev/null 2>&1; then
    echo "⚠️  Предупреждение: Не удалось подключиться к базе данных."
    echo "   База данных может быть недоступна или не существует."
    echo "   Пропускаем очистку."
    exit 0
fi

# Очищаем таблицу metrics
echo "  Очистка таблицы metrics..."
if psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "TRUNCATE TABLE metrics;" > /dev/null 2>&1; then
    echo "✅ Таблица metrics очищена успешно"
else
    echo "⚠️  Предупреждение: Не удалось очистить таблицу metrics."
    echo "   Таблица может не существовать (это нормально для первого запуска)."
fi

# Очищаем таблицу schema_migrations (опционально, если нужно сбросить миграции)
# Раскомментируйте следующую строку, если нужно также очистить историю миграций
# psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "TRUNCATE TABLE schema_migrations;" > /dev/null 2>&1 || true

echo ""
echo "✅ Очистка базы данных завершена"

