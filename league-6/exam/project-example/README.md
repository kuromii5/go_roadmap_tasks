# Poller

Сервис для создания опросов с ограниченным временем жизни. Создаёшь опрос с вариантами ответов — получаешь ссылку. Участники голосуют (один голос с одного IP). Опрос автоматически удаляется после истечения TTL.

## Видео-обзор проекта

https://drive.google.com/file/d/16f_EcbWnqIews8ymTeeTOgU1tL8WPGSP/view?usp=drive_link

## Стек

- **Go 1.26**
- **PostgreSQL** — хранение опросов, вариантов и записей голосов
- **Redis** — кэш результатов опросов + rate limiter на создание
- **goose** — SQL-миграции
- **log/slog** — структурированное логирование (stdlib)
- **caarlos0/env** — конфигурация через переменные окружения со struct tags

## Быстрый старт

```bash
cp .env.example .env
make compose-up
```

Сервис будет доступен на `http://localhost:8080`.

## Конфигурация (.env)

| Переменная      | По умолчанию      | Описание                        |
|-----------------|-------------------|---------------------------------|
| `PORT`          | `8080`            | Порт сервера                    |
| `DB_HOST`       | `localhost`       | Хост PostgreSQL                 |
| `DB_PORT`       | `5432`            | Порт PostgreSQL                 |
| `DB_USER`       | `postgres`        | Пользователь БД                 |
| `DB_PASSWORD`   | —                 | Пароль БД (обязательно)         |
| `DB_NAME`       | `poller`         | Имя базы данных                 |
| `DB_SSLMODE`    | `disable`         | SSL режим                       |
| `REDIS_ADDR`    | `localhost:6379`  | Адрес Redis                     |
| `REDIS_PASSWORD`| —                 | Пароль Redis (если нужен)       |
| `REDIS_DB`      | `0`               | Номер базы Redis                |

## Эндпоинты

### POST /api/polls — создать опрос

**Rate limit:** не более 10 запросов в минуту с одного IP.

```bash
curl -X POST http://localhost:8080/api/polls \
  -H "Content-Type: application/json" \
  -d '{
    "question": "Какой язык программирования лучший?",
    "options": ["Go", "Rust", "Python", "TypeScript"],
    "ttl_minutes": 60
  }'
```

Тело запроса:
| Поле          | Тип       | Описание                                          |
|---------------|-----------|---------------------------------------------------|
| `question`    | string    | Текст вопроса (обязательно)                       |
| `options`     | []string  | Варианты ответов (минимум 2, максимум 10)         |
| `ttl_minutes` | int       | Время жизни в минутах (по умолчанию 60, макс 10080) |

Ответ `201 Created`:
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "link": "/api/polls/550e8400-e29b-41d4-a716-446655440000",
  "expires_at": "2024-01-01T13:00:00Z"
}
```

Содержимое опроса в ответе **не возвращается** — только ID и ссылка.

---

### GET /api/polls/{id} — получить опрос с результатами

```bash
curl http://localhost:8080/api/polls/550e8400-e29b-41d4-a716-446655440000
```

Ответ `200 OK`:
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "question": "Какой язык программирования лучший?",
  "options": [
    {"id": "aaa-111", "text": "Go",         "votes": 5},
    {"id": "bbb-222", "text": "Rust",       "votes": 3},
    {"id": "ccc-333", "text": "Python",     "votes": 1},
    {"id": "ddd-444", "text": "TypeScript", "votes": 0}
  ],
  "expires_at": "2024-01-01T13:00:00Z"
}
```

Коды ошибок:
- `404` — опрос не найден
- `410 Gone` — опрос истёк

---

### POST /api/polls/{id}/vote — проголосовать

Один IP — один голос в каждом опросе. Повторный голос вернёт `409`.

```bash
curl -X POST http://localhost:8080/api/polls/550e8400-e29b-41d4-a716-446655440000/vote \
  -H "Content-Type: application/json" \
  -d '{"option_id": "aaa-111"}'
```

ID варианта берётся из ответа GET.

Ответ `204 No Content` при успехе.

Коды ошибок:
- `400` — не передан `option_id` или такого варианта нет в опросе
- `404` — опрос не найден
- `409 Conflict` — с этого IP уже голосовали
- `410 Gone` — опрос истёк

---

### DELETE /api/polls/{id} — удалить опрос

```bash
curl -X DELETE http://localhost:8080/api/polls/550e8400-e29b-41d4-a716-446655440000
```

Ответ `204 No Content`.

- `404` — опрос не найден

---

### GET /api/health — healthcheck

```bash
curl http://localhost:8080/api/health
```

```json
{"status":"ok","postgres":"ok","redis":"ok"}
```

При недоступности любого из сервисов — `503 Service Unavailable`.

---

## Архитектура

```
cmd/main.go                        — точка входа, сборка всех зависимостей

config/                            — конфигурация через env-переменные (struct tags)

internal/
  domain/                          — сущности и ошибки (без зависимостей)
    poll.go                        — Poll, Option, PollResult
    errors.go                      — sentinel errors

  service/poll/                    — бизнес-логика
    service.go                     — интерфейсы PollRepo и Cache, структура Service
    create.go                      — валидация, генерация UUID, сохранение
    get.go                         — кэш → БД, проверка TTL; Vote: проверка + инвалидация кэша
    delete.go                      — удаление из БД и кэша

  adapters/
    postgres/                      — работа с PostgreSQL (sqlx)
      postgres.go                  — SavePoll, GetByID, Vote (транзакция), Delete
      queries.go                   — SQL-константы
    redis/
      redis.go                     — создание клиента
      cache.go                     — Get/Set/Delete PollResult с TTL
      ratelimit.go                 — INCR + EXPIRE, 10 запросов/мин на IP

  handlers/
    middleware/
      requestid.go                 — X-Request-ID в каждый запрос
      logger.go                    — логирование метода, пути, статуса, времени
      ratelimit.go                 — применяет RateLimiter к хендлеру
    poll/                          — HTTP-хендлеры (decode → service → encode)
    health/                        — пинг PG и Redis
    router.go                      — маршрутизация, применение middleware
```

**Поток запроса:**
```
HTTP Request
  → RequestID middleware   (добавляет X-Request-ID)
  → Logger middleware      (замеряет время, логирует результат)
  → ServeMux               (роутинг по методу + пути)
  → RateLimit middleware   (только для POST /api/polls)
  → Handler                (decode JSON → вызов сервиса → encode JSON)
  → Service                (бизнес-логика, оркестрация)
  → Repo / Cache           (PostgreSQL / Redis)
```

## Как устроено голосование

Атомарность гарантируется транзакцией в PostgreSQL:

1. `INSERT INTO poll_votes (poll_id, ip)` — если этот IP уже голосовал, PRIMARY KEY нарушается и транзакция откатывается с ошибкой `ErrAlreadyVoted`
2. `UPDATE poll_options SET votes = votes + 1 WHERE id = $1` — инкремент счётчика

Redis-кэш инвалидируется после каждого голоса — следующий GET пойдёт в PostgreSQL и получит актуальные данные.

## Разработка

```bash
# Локальный запуск (PostgreSQL и Redis должны быть запущены)
make run

# Сборка бинаря в ./bin/poller
make build

# Миграции вручную
export DATABASE_URL="postgres://poller:poller@localhost:5432/poller?sslmode=disable"
make migrate-up
make migrate-down

# Поднять всё через Docker
make compose-up

# Остановить и удалить контейнеры (данные PostgreSQL сохранятся)
make compose-down

# Удалить вместе с данными
docker compose down -v
```

## Заголовки в каждом ответе

| Заголовок        | Описание                        |
|------------------|---------------------------------|
| `X-Request-ID`   | Уникальный ID запроса (UUID v4) |
| `Content-Type`   | `application/json`              |
