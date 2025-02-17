# Kafka Connect

## Задание 1

| Эксперимент | batch.size | linger.ms | compression.type | buffer.memory | Source Record Write Rate (кops/sec) |
|-------------|------------|-----------|------------------|---------------|-------------------------------------|
| 1           | 100        | 0         | none             | 32 MB         | 9.4                                 |
| 2           | 65536      | 50        | snappy           | 64 MB         | 140.0                               |
| 3           | 65536      | 200       | lz4              | 64 MB         | 147.0                               |
| 4           | 131072     | 200       | lz4              | 128 MB        | 152.0                               |


## Задание 2

Этот проект демонстрирует кастомный коннектор для извлечения метрик из Kafka и их представления в формате, понятном Prometheus.

## Компоненты проекта

1. **Zookeeper и Kafka** - брокер сообщений для хранения и передачи метрик
2. **Kafka Prometheus Connector** - кастомный коннектор, который читает метрики из Kafka и экспортирует их в формате Prometheus
3. **Prometheus** - система мониторинга для сбора и хранения метрик
4. **Metrics Producer** - генератор тестовых метрик для демонстрации работы коннектора

## Структура проекта

```
/
├── kafka-prometheus-connector/
│   ├── main.go
│   ├── Dockerfile
│   ├── go.mod
│   └── go.sum
├── metrics-producer/
│   ├── main.go
│   ├── Dockerfile
│   ├── go.mod
│   └── go.sum
├── prometheus/
│   └── prometheus.yml
└── docker-compose.yml
```

## Инструкции по запуску

### Предварительные требования

- Docker
- Docker Compose

### Шаги по запуску

1. Клонируйте репозиторий:
   ```bash
   git clone https://github.com/deeramster/kafka-connect.git
   cd kafka-connect
   ```

2. Создайте файлы `go.mod` и `go.sum` в директориях `kafka-prometheus-connector` и `metrics-producer`:

   В директории `kafka-prometheus-connector`:
   ```bash
   go mod init kafka-prometheus-connector
   go get github.com/confluentinc/confluent-kafka-go/kafka
   go get github.com/prometheus/client_golang/prometheus
   go get github.com/prometheus/client_golang/prometheus/promhttp
   ```

   В директории `metrics-producer`:
   ```bash
   go mod init metrics-producer
   go get github.com/confluentinc/confluent-kafka-go/kafka
   ```

3. Запустите все сервисы с помощью Docker Compose:
   ```bash
   docker compose up -d
   ```

4. Проверьте, что все контейнеры успешно запущены:
   ```bash
   docker compose ps
   ```

### Проверка работоспособности

1. **Доступ к Prometheus**: откройте в браузере http://localhost:9090

2. **Проверка работы Kafka**: для проверки, что сообщения успешно отправляются в Kafka, можно использовать инструменты клиента Kafka:
   ```bash
   docker-compose exec kafka kafka-console-consumer --bootstrap-server kafka:29092 --topic metrics --from-beginning
   ```

## Настройки коннектора

Коннектор настраивается через переменные окружения:

Метрики доступны по адресу http://localhost:8080/metrics

- `KAFKA_BOOTSTRAP_SERVERS` - адреса серверов Kafka (по умолчанию "localhost:9092")
- `KAFKA_TOPIC` - топик, из которого читаются метрики (по умолчанию "metrics")
- `KAFKA_GROUP_ID` - идентификатор группы потребителей (по умолчанию "prometheus-exporter")
- `LISTEN_ADDR` - адрес и порт для HTTP-сервера Prometheus (по умолчанию ":8080")

## Формат метрик

Коннектор ожидает JSON-сообщения в Kafka в следующем формате:

```json
{
    "MetricName1": {
        "Type": "gauge",
        "Name": "MetricName1",
        "Description": "Description of Metric1",
        "Value": 123.45
    },
    "MetricName2": {
        "Type": "counter",
        "Name": "MetricName2",
        "Description": "Description of Metric2",
        "Value": 67.89
    }
}
```

Поддерживаемые типы метрик:
- `gauge` - метрика, которая может как увеличиваться, так и уменьшаться
- `counter` - монотонно возрастающий счетчик

## Процесс работы коннектора

1. Коннектор подключается к Kafka и подписывается на указанный топик
2. При получении сообщения из Kafka, оно десериализуется из JSON
3. Для каждой метрики создается соответствующий gauge или counter в Prometheus
4. Метрики экспортируются по HTTP-интерфейсу для сбора Prometheus
5. Prometheus периодически собирает метрики с коннектора



## Задание 3

```bash
curl -X PUT -H 'Content-Type: application/json' --data @connector-truncate.json http://localhost:8083/connectors/pg-connector/config | jq
% Total    % Received % Xferd  Average Speed   Time    Time     Time  Current
Dload  Upload   Total   Spent    Left  Speed
100  1669  100   864  100   805  19971  18607 --:--:-- --:--:-- --:--:-- 38813
```
```json
{
"name": "pg-connector",
"config": {
"connector.class": "io.debezium.connector.postgresql.PostgresConnector",
"database.hostname": "postgres",
"database.port": "5432",
"database.user": "postgres-user",
"database.password": "postgres-pw",
"database.dbname": "customers",
"database.server.name": "customers",
"table.whitelist": "public.customers",
"transforms": "unwrap",
"transforms.unwrap.type": "io.debezium.transforms.ExtractNewRecordState",
"transforms.unwrap.drop.tombstones": "false",
"transforms.unwrap.delete.handling.mode": "rewrite",
"topic.prefix": "customers",
"topic.creation.enable": "true",
"topic.creation.default.replication.factor": "-1",
"topic.creation.default.partitions": "-1",
"signal.data.collection": "true",
"signal.kafka.key": "true",
"signal.kafka.value": "true",
"skipped.operations": "none",
"name": "pg-connector"
},
"tasks": [
{
"connector": "pg-connector",
"task": 0
}
],
"type": "source"
}
```