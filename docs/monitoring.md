# 📊 Мониторинг Volnix Protocol

## Обзор

Volnix Protocol включает встроенный мониторинг с Prometheus метриками и health checks для отслеживания состояния узла.

## Endpoints

### Health Check
```bash
# Проверка здоровья узла
curl http://localhost:9090/health

# Kubernetes-style health check
curl http://localhost:9090/healthz

# Детальный статус
curl http://localhost:9090/status
```

### Prometheus Metrics
```bash
# Получить все метрики в формате Prometheus
curl http://localhost:9090/metrics
```

## Метрики

### Block Metrics
- `volnix_block_height` - Текущая высота блока
- `volnix_block_time_seconds` - Время создания блока (гистограмма)
- `volnix_block_size_bytes` - Размер блока в байтах

### Transaction Metrics
- `volnix_tx_total` - Общее количество транзакций
- `volnix_tx_latency_seconds` - Задержка обработки транзакций (гистограмма)
- `volnix_tx_success_total` - Количество успешных транзакций
- `volnix_tx_failed_total` - Количество неудачных транзакций

### Validator Metrics
- `volnix_validator_count` - Количество активных валидаторов
- `volnix_validator_power_total` - Общая мощность валидаторов

### Network Metrics
- `volnix_peer_count` - Количество подключенных пиров
- `volnix_peer_inbound_count` - Количество входящих пиров
- `volnix_peer_outbound_count` - Количество исходящих пиров

### Sync Metrics
- `volnix_sync_status` - Статус синхронизации (1 = синхронизирован, 0 = синхронизируется)

### Gas Metrics
- `volnix_gas_used_total` - Общее количество использованного газа
- `volnix_gas_limit_total` - Общее количество лимита газа

### Error Metrics
- `volnix_errors_total` - Общее количество ошибок

### Node Health
- `volnix_node_healthy` - Статус здоровья узла (1 = здоров, 0 = нездоров)

## Настройка

### Порт мониторинга
По умолчанию мониторинг работает на порту `9090`. Можно изменить через переменную окружения:

```bash
export VOLNIX_METRICS_PORT=9090
./build/volnixd-standalone start
```

## Prometheus Configuration

Добавьте в `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'volnix'
    static_configs:
      - targets: ['localhost:9090']
    scrape_interval: 10s
```

## Grafana Dashboard

Импортируйте дашборд из `infrastructure/grafana/dashboards/volnix-network.json`:

1. Откройте Grafana
2. Перейдите в Dashboards → Import
3. Загрузите файл `infrastructure/grafana/dashboards/volnix-network.json`
4. Выберите Prometheus data source
5. Сохраните дашборд

Дашборд включает:
- Block Height - высота блока
- Transaction Rate - скорость транзакций
- Connected Peers - подключенные пиры
- Block Time - время создания блока
- Transaction Success Rate - процент успешных транзакций
- Node Health - статус здоровья узла
- Sync Status - статус синхронизации
- Validators - количество валидаторов
- Total Errors - общее количество ошибок
- Gas Usage - использование газа
- Transaction Latency - задержка транзакций

## Примеры запросов Prometheus

### Скорость транзакций
```promql
rate(volnix_tx_total[5m])
```

### Процент успешных транзакций
```promql
rate(volnix_tx_success_total[5m]) / rate(volnix_tx_total[5m]) * 100
```

### 95-й перцентиль времени блока
```promql
histogram_quantile(0.95, volnix_block_time_seconds_bucket)
```

### Средняя задержка транзакций
```promql
histogram_quantile(0.50, volnix_tx_latency_seconds_bucket)
```

## Health Check Response

### Healthy Node
```json
{
  "status": "healthy",
  "timestamp": "2025-11-20T12:00:00Z",
  "version": "0.1.0",
  "chain_id": "test-volnix-standalone",
  "block": {
    "height": 100,
    "time": "2025-11-20T12:00:00Z",
    "synced": true
  },
  "network": {
    "peers": 3,
    "inbound_peers": 1,
    "outbound_peers": 2
  },
  "node": {
    "running": true,
    "rpc_address": "tcp://0.0.0.0:26657",
    "p2p_address": "tcp://0.0.0.0:26656"
  }
}
```

### Unhealthy Node
HTTP Status: `503 Service Unavailable`

```json
{
  "status": "unhealthy",
  "timestamp": "2025-11-20T12:00:00Z",
  ...
}
```

## Интеграция с Kubernetes

Для использования в Kubernetes добавьте liveness и readiness probes:

```yaml
livenessProbe:
  httpGet:
    path: /healthz
    port: 9090
  initialDelaySeconds: 30
  periodSeconds: 10

readinessProbe:
  httpGet:
    path: /healthz
    port: 9090
  initialDelaySeconds: 5
  periodSeconds: 5
```

## Troubleshooting

### Метрики не обновляются
- Проверьте, что узел запущен и работает
- Убедитесь, что порт 9090 доступен
- Проверьте логи узла на наличие ошибок

### Health check возвращает unhealthy
- Проверьте, что CometBFT узел запущен
- Проверьте логи на наличие ошибок
- Убедитесь, что база данных доступна

### Prometheus не может скрапить метрики
- Проверьте, что порт 9090 открыт в firewall
- Убедитесь, что Prometheus может достичь узла
- Проверьте конфигурацию Prometheus



