# 🐳 Docker Deployment Guide

## Быстрый старт

### Запуск мультинод сети

```bash
# Сборка образов
docker-compose build

# Запуск сети (3 валидатора + frontend)
docker-compose up -d

# Проверка статуса
docker-compose ps

# Логи
docker-compose logs -f validator-0
```

## Endpoints

- **Validator 0 RPC:** http://localhost:26657
- **Validator 1 RPC:** http://localhost:26757
- **Validator 2 RPC:** http://localhost:26857
- **Wallet UI:** http://localhost:3000

## Управление

### Запуск
```bash
docker-compose up -d
```

### Остановка
```bash
docker-compose down
```

### Остановка с удалением данных
```bash
docker-compose down -v
```

### Перезапуск
```bash
docker-compose restart
```

### Логи
```bash
# Все сервисы
docker-compose logs -f

# Конкретный валидатор
docker-compose logs -f validator-0

# Последние 100 строк
docker-compose logs --tail=100 validator-0
```

### Проверка статуса
```bash
# Статус контейнеров
docker-compose ps

# Health checks
docker ps --format "table {{.Names}}\t{{.Status}}"

# Статус узла
curl http://localhost:26657/status | jq
```

## Volumes

Данные сохраняются в Docker volumes:
- `validator-0-data` - данные узла 0
- `validator-1-data` - данные узла 1
- `validator-2-data` - данные узла 2

### Backup данных
```bash
# Создание backup
docker run --rm -v volnix_validator-0-data:/data -v $(pwd):/backup alpine tar czf /backup/validator-0-backup.tar.gz -C /data .

# Восстановление
docker run --rm -v volnix_validator-0-data:/data -v $(pwd):/backup alpine tar xzf /backup/validator-0-backup.tar.gz -C /data
```

## Production considerations

### Secrets management

Используйте Docker secrets для приватных ключей:

```yaml
services:
  validator-0:
    secrets:
      - validator_key
      - node_key

secrets:
  validator_key:
    file: ./secrets/validator_key.json
  node_key:
    file: ./secrets/node_key.json
```

### Resource limits

```yaml
services:
  validator-0:
    deploy:
      resources:
        limits:
          cpus: '2.0'
          memory: 4G
        reservations:
          cpus: '1.0'
          memory: 2G
```

### Мониторинг

Добавьте Prometheus и Grafana:

```yaml
services:
  prometheus:
    image: prom/prometheus
    volumes:
      - ./monitoring/prometheus.yml:/etc/prometheus/prometheus.yml
    ports:
      - "9090:9090"

  grafana:
    image: grafana/grafana
    ports:
      - "3001:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
```

## Kubernetes deployment

Для Kubernetes используйте Helm charts (см. `k8s/` директорию).

## Troubleshooting

### Контейнер не запускается
```bash
# Проверить логи
docker-compose logs validator-0

# Войти в контейнер
docker-compose exec validator-0 sh

# Проверить конфигурацию
docker-compose exec validator-0 cat /home/volnix/.volnix/config/config.toml
```

### Порты заняты
```bash
# Проверить занятые порты
lsof -i :26657

# Изменить порты в docker-compose.yml
ports:
  - "27657:26657"  # Используйте другой host port
```

### Проблемы с производительностью
```bash
# Проверить ресурсы
docker stats

# Увеличить лимиты в docker-compose.yml
```

## Масштабирование

### Добавление валидатора

1. Создайте новый сервис в `docker-compose.yml`
2. Настройте порты
3. Добавьте в persistent_peers
4. Запустите: `docker-compose up -d validator-3`

### Horizontal scaling

Используйте Kubernetes для автоматического масштабирования.

