# Экзамен — URL Shortener

Сервис для сокращения ссылок. Пользователь отправляет длинный URL — получает короткую ссылку. Переходит по ней — попадает на оригинальный адрес. Сервис считает переходы.

Аналог: [clck.ru](https://clck.ru), [bit.ly](https://bit.ly)

---

## Эндпоинты

### `POST /api/links` — создать короткую ссылку

Принимает: оригинальный URL, опциональный кастомный alias (если не передан — генерируется автоматически), опциональное время жизни в часах (по умолчанию 24 часа).

Возвращает: короткий код, полную короткую ссылку, оригинальный URL, время истечения.

```json
// Запрос
{ "url": "https://very-long-url.com/some/path", "alias": "mylink", "expires_in_hours": 48 }

// Ответ 201
{ "code": "mylink", "short_url": "http://localhost:8080/mylink", "original_url": "...", "expires_at": "..." }
```

### `GET /:code` — редирект по короткой ссылке

Ищет оригинальный URL по коду. Увеличивает счётчик переходов. Отвечает `301 Moved Permanently` с заголовком `Location`.

Если ссылка не найдена — `404`. Если истекла — `410 Gone`.

### `GET /api/links/:code` — информация о ссылке

Возвращает оригинальный URL, код, количество переходов, время создания и истечения.

### `DELETE /api/links/:code` — удалить ссылку

`204 No Content`. Если не найдена — `404`.

### `GET /api/health` — healthcheck

Возвращает статус сервиса, подключение к PostgreSQL и Redis.

---

## Требования

### Архитектура — handler / service / repository

```
internal/
├── handler/       # HTTP-обработчики, валидация запроса, формирование ответа
├── service/       # бизнес-логика (генерация кода, проверка TTL, подсчёт переходов)
├── repository/    # интерфейс + реализация работы с PostgreSQL
├── model/         # структуры (Link)
└── config/        # конфигурация
```

Слои общаются через **интерфейсы**. `handler` знает только об интерфейсе `service`, `service` — только об интерфейсе `repository`. Конкретные реализации нигде не импортируются напрямую.

### PostgreSQL

Таблица `links`:

| Поле | Тип | Описание |
|---|---|---|
| id | SERIAL PRIMARY KEY | — |
| code | VARCHAR(32) UNIQUE | короткий код |
| original_url | TEXT | оригинальный URL |
| clicks | INTEGER | счётчик переходов |
| created_at | TIMESTAMP | — |
| expires_at | TIMESTAMP | время истечения |

Схема создаётся через **миграции** (goose или migrate). Не `CREATE TABLE IF NOT EXISTS` в коде — только миграции.

### Redis

- **Кэш**: при редиректе сначала смотреть в Redis, только потом в PostgreSQL. Ключ — код ссылки, TTL — соответствует `expires_at`.
- **Rate limiter**: не более 10 запросов на создание ссылки в минуту с одного IP.

### Middleware

- **Логирование** каждого запроса: метод, путь, статус, время выполнения (logrus или slog)
- **Request ID**: каждый запрос получает уникальный ID, он пробрасывается через `context` и логируется
- **Rate limiter**: применяется к `POST /api/links`

### Конфигурация

Все параметры через `.env` и `config.yml`:
- `HTTP_PORT` — порт сервера (по умолчанию `8080`)
- `POSTGRES_DSN` — строка подключения к PostgreSQL
- `REDIS_ADDR` — адрес Redis
- `LOG_LEVEL` — уровень логирования
- `DEFAULT_TTL_HOURS` — дефолтный TTL ссылки (по умолчанию `24`)
- `BASE_URL` — базовый URL сервиса для формирования короткой ссылки

### Docker

`docker-compose up --build` поднимает три сервиса: приложение, PostgreSQL, Redis. Данные PostgreSQL сохраняются между перезапусками (volume). Конфиг читается из `.env`.

### Error handling

Нет `panic` в бизнес-логике. Все ошибки оборачиваются через `fmt.Errorf("...: %w", err)` и пробрасываются наверх. Обработка — на уровне handler, который переводит ошибку в нужный HTTP-статус.

Свои типы ошибок для разных ситуаций — `ErrNotFound`, `ErrExpired`, `ErrAliasConflict`.

### Makefile

```makefile
make run        # запустить сервер
make build      # собрать бинарник
make compose    # docker-compose up --build
make migrate    # применить миграции
make lint       # запустить линтер
```

### README.md

Описание проекта, инструкция по запуску, примеры curl-запросов.

---

## Структура проекта

```
url-shortener/
├── cmd/
│   └── main.go
├── internal/
│   ├── config/
│   │   └── config.go
│   ├── handler/
│   │   ├── handler.go
│   │   └── links.go
│   ├── service/
│   │   ├── interface.go
│   │   └── links.go
│   ├── repository/
│   │   ├── interface.go
│   │   └── postgres/
│   │       └── links.go
│   ├── middleware/
│   │   ├── logger.go
│   │   └── ratelimit.go
│   └── model/
│       └── link.go
├── migrations/
│   ├── 001_create_links.up.sql
│   └── 001_create_links.down.sql
├── .env
├── .env.example
├── config.yml
├── docker-compose.yml
├── Dockerfile
├── Makefile
└── README.md
```

---

## Проверка

```bash
# Запустить всё
docker-compose up --build

# Создать ссылку
curl -X POST http://localhost:8080/api/links \
  -H "Content-Type: application/json" \
  -d '{"url": "https://google.com", "expires_in_hours": 1}'

# Ответ:
# {"code": "aB3kZ9", "short_url": "http://localhost:8080/aB3kZ9", ...}

# Перейти по ссылке (должен вернуть 301)
curl -v http://localhost:8080/aB3kZ9

# Статистика
curl http://localhost:8080/api/links/aB3kZ9
# {"code": "aB3kZ9", "original_url": "https://google.com", "clicks": 1, ...}

# Кастомный alias
curl -X POST http://localhost:8080/api/links \
  -d '{"url": "https://github.com", "alias": "gh"}'

# Конфликт alias — 409
curl -X POST http://localhost:8080/api/links \
  -d '{"url": "https://example.com", "alias": "gh"}'

# Удалить
curl -X DELETE http://localhost:8080/api/links/gh
# 204

# Истёкшая ссылка — 410
curl -X POST http://localhost:8080/api/links \
  -d '{"url": "https://example.com", "expires_in_hours": 0}'
curl http://localhost:8080/<code>
# 410 Gone

# Rate limit — 11-й запрос подряд должен вернуть 429
for i in $(seq 1 11); do
  curl -X POST http://localhost:8080/api/links -d '{"url": "https://example.com"}'
done

# Healthcheck
curl http://localhost:8080/api/health
# {"status": "ok", "postgres": "ok", "redis": "ok"}
```

---

## Критерии оценки

| Критерий | Баллы    |
|---|----------|
| Все 5 эндпоинтов работают | 10 баллов |
| Архитектура handler/service/repo | 10 баллов |
| Слои общаются через интерфейсы | 10 баллов |
| PostgreSQL с миграциями | 10 баллов |
| Redis кэш | 10 баллов |
| rate limiter | 10 баллов |
| Middleware: логирование + request ID | 10 баллов |
| Error handling без паники, свои типы ошибок | 10 баллов |
| Конфиг через .env | 10 баллов |
| docker-compose up поднимает всё | 10 баллов |
| Makefile | 10 баллов |
| README с curl-примерами | 10 баллов |

Для прохождения экзамена необходимо набрать минимум 80 баллов 
---

