# Финальный проект — ShopTrack

Два микросервиса: один принимает заказы, второй считает аналитику. Связаны через Kafka. Простая бизнес-логика — весь фокус на архитектуре и инфраструктуре.

---

## Order Service — REST API

### `POST /api/v1/products` — добавить товар

Принимает: название, цену, количество на складе.

Возвращает: ID товара, название, цену.

### `GET /api/v1/products` — список товаров

Возвращает каталог товаров. **Ответ должен кэшироваться в Redis на 60 секунд.** При изменении товара — инвалидировать кэш.

### `POST /api/v1/orders` — создать заказ

Принимает: список товаров с количеством.

Возвращает: ID заказа, итоговую сумму, статус `pending`.

После сохранения в PostgreSQL — **публикует событие в Kafka** топик `order.created`.

Требует JWT в заголовке `Authorization: Bearer <token>`. Без токена — `401`.

### `GET /api/v1/orders/{id}` — получить заказ

Возвращает заказ с позициями. Чужой заказ — `403`.

### `GET /api/v1/orders` — заказы текущего пользователя

Только заказы авторизованного пользователя.

### `POST /api/v1/auth/register` и `POST /api/v1/auth/login`

Регистрация и логин. Логин возвращает JWT-токен. Пароль — хешированный (bcrypt).

### `GET /api/health`

Статус сервиса, подключение к PostgreSQL и Redis.

---

## Analytics Service — gRPC API

Читает события из Kafka и отдаёт статистику через gRPC.

### `GetTopProducts` — топ товаров по количеству продаж

Принимает: период (from, to), лимит (top N).

Возвращает: список товаров с количеством продаж и выручкой.

### `GetRevenue` — выручка за период

Принимает: from, to, опционально — группировка по дням/неделям/месяцам.

Возвращает: суммарную выручку и разбивку по периодам.

### `GetOrdersCount` — количество заказов

Принимает: период. Возвращает: общее количество и среднее в день.

---

## Требования

### PostgreSQL (Order Service)

Хранит пользователей, товары, заказы, позиции заказов. Пароли — bcrypt. Миграции через **goose**. Связи через внешние ключи, индексы на `user_id` и `created_at`.

```sql
-- Пример структуры
users(id, email, password_hash, created_at)
products(id, name, price, stock, created_at)
orders(id, user_id, total_amount, status, created_at)
order_items(id, order_id, product_id, quantity, price_at_purchase)
```

### Redis (Order Service)

- **Кэш** каталога товаров с TTL 60 секунд
- При `POST /api/v1/products` и обновлении — инвалидировать ключ `products:catalog`
- Rate limiter на создание заказов — не более 10 в минуту с одного пользователя

### Kafka

Order Service публикует в топик `order.created` событие:

```json
{
  "order_id": "uuid",
  "user_id": "uuid",
  "total_amount": 1500.00,
  "items": [{"product_id": "uuid", "quantity": 2, "price": 750.00}],
  "created_at": "2024-01-15T10:30:00Z"
}
```

Analytics Service читает этот топик и пишет в ClickHouse.

### ClickHouse (Analytics Service)

```sql
CREATE TABLE orders_analytics (
    order_id     UUID,
    user_id      UUID,
    product_id   UUID,
    quantity     UInt32,
    price        Float64,
    total_amount Float64,
    created_at   DateTime
) ENGINE = MergeTree()
ORDER BY (created_at, product_id);
```

### Архитектура — 3 слоя

Оба сервиса строятся по одной схеме:

```
handler (transport)  →  service (business logic)  →  repository (data)
```

Зависимости только через интерфейсы. Сервис не знает про HTTP или SQL — только про бизнес-логику.

### Middleware (Order Service)

- JWT-проверка на защищённых роутах
- Логирование: метод, путь, статус, время выполнения, request ID
- Rate limiter на `POST /api/v1/orders`
- Request ID пробрасывается в контекст и в ответ через заголовок `X-Request-ID`

### Метрики — Prometheus + Grafana

Оба сервиса экспортируют `/metrics`. Собирать:

- Количество HTTP-запросов по методу/пути/статусу
- Latency запросов (гистограмма)
- Количество событий из Kafka (в Analytics Service)
- Активные горутины, использование памяти

Grafana — минимум 2 дашборда: один для Order Service, один для Analytics.

### Логирование — zap или slog

Структурированный JSON-лог. Каждый лог содержит `request_id`, `service`, `level`. Уровень логирования через переменную окружения `LOG_LEVEL`.

### Трейсинг — OpenTelemetry + Jaeger

Трейс должен проходить через оба сервиса. Span создаётся в Order Service при получении запроса, пробрасывается в Kafka-событии через заголовки, продолжается в Analytics Service при обработке.

### Docker

`docker-compose up --build` поднимает всё: Order Service, Analytics Service, PostgreSQL, Redis, Kafka, Zookeeper, ClickHouse, Prometheus, Grafana, Jaeger.

Данные PostgreSQL и ClickHouse не теряются при перезапуске (volumes).

### Остальное

- Конфигурация через `.env`
- Makefile: `run`, `build`, `migrate`, `compose`, `test`, `proto`
- `.proto`-файл для gRPC лежит в отдельной папке `proto/`
- README с архитектурой, как запустить, примеры запросов

---

## Проверка

```bash
# Регистрация и логин
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email": "user@test.com", "password": "secret123"}'

curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "user@test.com", "password": "secret123"}'
# → получаешь JWT token

# Добавить товар
curl -X POST http://localhost:8080/api/v1/products \
  -H "Authorization: Bearer <token>" \
  -d '{"name": "Ноутбук", "price": 75000, "stock": 10}'

# Список товаров (первый раз — из PostgreSQL, второй — из Redis)
curl http://localhost:8080/api/v1/products

# Создать заказ (триггерит событие в Kafka)
curl -X POST http://localhost:8080/api/v1/orders \
  -H "Authorization: Bearer <token>" \
  -d '{"items": [{"product_id": "<id>", "quantity": 2}]}'

# Без токена — 401
curl -X POST http://localhost:8080/api/v1/orders \
  -d '{"items": [...]}'

# gRPC — топ товаров (через grpcurl)
grpcurl -d '{"from": "2024-01-01T00:00:00Z", "to": "2024-12-31T00:00:00Z", "limit": 5}' \
  localhost:9090 analytics.AnalyticsService/GetTopProducts

# Метрики
curl http://localhost:8080/metrics
curl http://localhost:9090/metrics

# Трейсы — открыть в браузере
open http://localhost:16686  # Jaeger UI

# Grafana
open http://localhost:3000   # admin/admin
```