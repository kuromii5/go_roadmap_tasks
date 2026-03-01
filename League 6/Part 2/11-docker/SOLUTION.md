# 🔑 Решения — Docker + docker-compose

---

## Задача 1 — Исправь Dockerfile

```dockerfile
# --- Build stage ---
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Сначала зависимости — они меняются реже
COPY go.mod go.sum ./
RUN go mod download

# Потом код
COPY . .
RUN CGO_ENABLED=0 go build -o main ./cmd/server

# --- Final stage ---
FROM alpine:3.20

RUN adduser -D -s /bin/sh appuser

WORKDIR /app
COPY --from=builder /app/main .

USER appuser

EXPOSE 8080
CMD ["./main"]
```

### 5 исправлений

| # | Было | Стало | Зачем |
|---|------|-------|-------|
| 1 | `FROM golang:1.23` (финальный) | Multi-stage: `builder` + `alpine` | Образ ~900MB → ~15MB. Нет компилятора и исходников в проде |
| 2 | `COPY . .` потом `go mod download` | Сначала `go.mod/go.sum`, потом `COPY . .` | Кэш зависимостей не сбрасывается при изменении кода |
| 3 | Запуск от root | `adduser` + `USER appuser` | Если уязвимость — атакер не получит root в контейнере |
| 4 | `golang:1.23` как база | `alpine:3.20` | Минимальный образ, меньше attack surface |
| 5 | Нет `CGO_ENABLED=0` | Добавлено | Статическая линковка — бинарник работает в alpine без glibc |

---

## Задача 2 — docker-compose с нуля

### `docker-compose.yml`

```yaml
services:
  app:
    build: .
    ports:
      - "8080:8080"
    env_file:
      - .env
    environment:
      - DATABASE_URL=postgres://${DB_USER}:${DB_PASSWORD}@postgres:5432/${DB_NAME}?sslmode=disable
      - REDIS_ADDR=redis:6379
    depends_on:
      - postgres
      - redis
    networks:
      - backend

  postgres:
    image: postgres:16-alpine
    ports:
      - "5432:5432"
    env_file:
      - .env
    environment:
      - POSTGRES_USER=${DB_USER}
      - POSTGRES_PASSWORD=${DB_PASSWORD}
      - POSTGRES_DB=${DB_NAME}
    volumes:
      - pgdata:/var/lib/postgresql/data
    networks:
      - backend

  redis:
    image: redis:alpine
    networks:
      - backend

volumes:
  pgdata:

networks:
  backend:
```

### `.env`

```env
DB_USER=postgres
DB_PASSWORD=secret
DB_NAME=myapp
```

### Что здесь важно

- **Имена сервисов = хостнеймы.** `postgres:5432`, не `localhost:5432`. Внутри docker-compose сервисы видят друг друга по имени.
- **`volumes: pgdata`** — именованный volume. Данные PostgreSQL переживают `docker-compose down`. Без volume — всё удалится.
- **Redis без `ports`** — наружу не торчит. Доступен только для app внутри сети `backend`.
- **`depends_on`** — гарантирует порядок запуска контейнеров, но **не** ждёт готовности базы. В реальности app должен retry подключение.

---

## Задача 3 — Что пойдёт не так?

| # | Проблема | Почему | Исправление |
|---|----------|--------|-------------|
| 1 | `localhost` в `DATABASE_URL` | Контейнеры — изолированные сети. `localhost` внутри app — это сам app, не postgres | `postgres:5432` (имя сервиса) |
| 2 | Нет volume у postgres | `docker-compose down` → данные потеряны навсегда | `volumes: - pgdata:/var/lib/postgresql/data` |
| 3 | Redis выставлен наружу (`6379:6379`) | Redis без пароля доступен всем в сети. Если Redis нужен только app — порт наружу не нужен | Убрать `ports` у redis |
| 4 | У postgres нет `POSTGRES_USER` и `POSTGRES_DB` | Будет дефолтный user `postgres` и БД `postgres`. `DATABASE_URL` ссылается на `admin` и `mydb` — не совпадёт | Добавить `POSTGRES_USER=admin` и `POSTGRES_DB=mydb` |
