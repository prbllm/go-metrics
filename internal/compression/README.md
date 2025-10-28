# Compression Package

Пакет `compression` предоставляет общие функции для работы со сжатием данных в проекте go-metrics.

## Функции

### CompressData
Сжимает данные с помощью алгоритма gzip.

```go
compressed, err := compression.CompressData([]byte("Hello, World!"))
if err != nil {
    log.Fatal(err)
}
```

### DecompressData
Распаковывает gzip данные.

```go
decompressed, err := compression.DecompressData(compressedData)
if err != nil {
    log.Fatal(err)
}
```

### SupportsGzip
Проверяет, поддерживает ли клиент gzip сжатие по заголовку Accept-Encoding.

```go
if compression.SupportsGzip(r.Header.Get("Accept-Encoding")) {
    // Клиент поддерживает gzip
}
```

### GetCompressionStats
Возвращает статистику сжатия.

```go
stats := compression.GetCompressionStats(originalData, compressedData)
fmt.Printf("Compression ratio: %.2f\n", stats.CompressionRatio)
```

## Использование в проекте

### Агент (Agent)
- Сжатие JSON данных перед отправкой на сервер
- Логирование статистики сжатия

### Сервер (Server)
- Проверка поддержки gzip клиентом
- Middleware для декомпрессии входящих запросов
- Сжатие исходящих ответов

## Преимущества

1. **Устранение дублирования кода** - общая логика сжатия вынесена в отдельный пакет
2. **Единообразие** - одинаковое поведение сжатия во всех частях приложения
3. **Тестируемость** - легко тестировать функции сжатия изолированно
4. **Расширяемость** - легко добавить поддержку других алгоритмов сжатия
5. **Статистика** - встроенная поддержка метрик сжатия

## Тестирование

```bash
go test ./pkg/compression/... -v
```

Пакет покрыт комплексными тестами, включающими:
- Тестирование сжатия и декомпрессии
- Проверку корректности round-trip операций
- Тестирование различных типов данных
- Обработку ошибок
- Статистику сжатия
