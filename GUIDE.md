# Руководство по эксплуатации

## Содержание

1. [Запуск системы](#запуск-системы)
2. [REST API](#rest-api)
3. [Grafana](#grafana)
4. [Prometheus](#prometheus)
5. [Остановка и очистка](#остановка-и-очистка)

---

## Запуск системы

```bash
docker-compose up --build
```

После запуска доступны:

| Сервис     | URL                        |
|------------|----------------------------|
| REST API   | http://localhost:8080      |
| Prometheus | http://localhost:9090      |
| Grafana    | http://localhost:3000      |

> Первый запуск может занять 1–2 минуты — Grafana и Prometheus скачивают образы.

---

## REST API

### Эндпоинты

| Метод  | URL             | Описание                  |
|--------|-----------------|---------------------------|
| GET    | /ping           | Проверка доступности      |
| GET    | /health         | Health check              |
| GET    | /items          | Получить все элементы     |
| POST   | /items          | Создать новый элемент     |
| GET    | /metrics        | Метрики Prometheus (raw)  |

### Примеры запросов

```bash
# Ping
curl http://localhost:8080/ping

# Получить все элементы
curl http://localhost:8080/items

# Создать элемент
curl -X POST http://localhost:8080/items \
  -H "Content-Type: application/json" \
  -d '{"name": "item-3", "value": "value-3"}'
```

Все примеры также доступны в файле `requests.http` (IntelliJ IDEA / VS Code REST Client).

---

## Grafana

### Вход

1. Откройте http://localhost:3000
2. Введите логин: **admin**
3. Введите пароль: **admin**
4. При первом входе Grafana предложит сменить пароль — можно пропустить кнопкой **Skip**

### Дашборд

Дашборд **REST API Metrics** подключается автоматически при старте.

Чтобы открыть его:

1. В левом меню нажмите **Dashboards** (иконка с четырьмя квадратами)
2. Выберите папку **General**
3. Нажмите на **REST API Metrics**

### Панели дашборда

| Панель                          | Описание                                      |
|---------------------------------|-----------------------------------------------|
| Total HTTP Requests             | Общее количество запросов                     |
| Request Rate (req/s)            | Количество запросов в секунду                 |
| Active Requests                 | Текущие активные запросы                      |
| Items in Store                  | Количество элементов в хранилище              |
| Request Rate by Endpoint        | RPS по каждому эндпоинту и методу             |
| Request Duration p50/p95/p99    | Латентность (медиана, 95-й и 99-й перцентили) |
| Requests by Status              | Запросы по HTTP-статусам                      |
| Total Requests by Endpoint      | Суммарные запросы по эндпоинтам (бар-чарт)   |

### Смена пароля администратора

1. Нажмите на иконку пользователя в левом нижнем углу
2. Выберите **Profile**
3. Перейдите на вкладку **Security**
4. Введите новый пароль и сохраните

Либо через переменные окружения в `docker-compose.yml`:

```yaml
environment:
  - GF_SECURITY_ADMIN_USER=admin
  - GF_SECURITY_ADMIN_PASSWORD=my_new_password
```

---

## Prometheus

### Веб-интерфейс

Откройте http://localhost:9090

### Проверка targets

1. Перейдите в **Status → Targets**
2. Убедитесь, что `rest-api` имеет статус **UP**

### Полезные запросы (PromQL)

```promql
# Общий RPS
sum(rate(http_requests_total[1m]))

# RPS по эндпоинту
sum(rate(http_requests_total[1m])) by (endpoint)

# Медианная латентность
histogram_quantile(0.50, sum(rate(http_request_duration_seconds_bucket[1m])) by (le))

# 99-й перцентиль латентности
histogram_quantile(0.99, sum(rate(http_request_duration_seconds_bucket[1m])) by (le))

# Количество ошибок (не 2xx)
sum(rate(http_requests_total[1m])) by (status)

# Текущее количество элементов
items_total
```

---

## Остановка и очистка

### Остановить контейнеры (данные сохраняются)

```bash
docker-compose down
```

### Остановить и удалить все данные (volumes)

```bash
docker-compose down -v
```

### Пересобрать только приложение

```bash
docker-compose up --build app
```

### Посмотреть логи

```bash
# Все сервисы
docker-compose logs -f

# Только приложение
docker-compose logs -f app

# Только Prometheus
docker-compose logs -f prometheus
```
