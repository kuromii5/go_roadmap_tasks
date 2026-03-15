# Практика — Docker

> **Видео:** [Docker и docker-compose](https://youtu.be/MNyNxloZR0k?t=27996)

---

## Задача 1 — Исправь Dockerfile

Ниже — Dockerfile для Go-приложения. Он работает, но содержит **5 проблем** (безопасность, размер образа, кэширование, практики).

```dockerfile
FROM golang:1.23

WORKDIR /app

COPY . .

RUN go mod download
RUN go build -o main ./cmd/server

EXPOSE 8080

CMD ["./main"]
```

**Что нужно сделать:**

Перепиши с учётом:

1. Финальный образ не должен содержать компилятор Go и исходники (multi-stage build)
2. Зависимости должны кэшироваться отдельно от кода
3. Приложение не должно запускаться от root
4. Финальный базовый образ — минимальный
5. Подумай про порядок слоёв — что меняется чаще, то копируется позже

---

## Задача 2 — docker-compose с нуля

Напиши `docker-compose.yml` для проекта, который состоит из:

- **app** — Go-приложение (собирается из Dockerfile)
- **postgres** — PostgreSQL 16
- **redis** — Redis Alpine

**Требования:**

| Сервис | Порт | Переменные |
|--------|------|-----------|
| app | 8080:8080 | `DATABASE_URL`, `REDIS_ADDR` |
| postgres | 5432:5432 | `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB` |
| redis | — (доступен только внутри сети) | — |

- Переменные окружения — через `.env` файл
- PostgreSQL должен сохранять данные при перезапуске (volume)
- App должен стартовать **после** postgres и redis (`depends_on`)
- Все сервисы — в одной сети

**Проверка:**

```bash
docker-compose up --build
# App подключается к postgres и redis по именам сервисов
```

---

## Задача 3 — Что пойдёт не так?

Дан `docker-compose.yml`. Найди **4 проблемы**.

```yaml
version: "3.8"

services:
  app:
    build: .
    ports:
      - "8080:8080"
    environment:
      - DATABASE_URL=postgres://admin:secret@localhost:5432/mydb?sslmode=disable
      - REDIS_ADDR=redis:6379

  postgres:
    image: postgres:16
    ports:
      - "5432:5432"
    environment:
      - POSTGRES_PASSWORD=secret

  redis:
    image: redis:alpine
    ports:
      - "6379:6379"
```

**Подсказки:**

1. Как контейнеры общаются друг с другом — через `localhost`?
2. Что будет с данными PostgreSQL при `docker-compose down`?
3. Нужен ли Redis снаружи?
4. Посмотри внимательно на переменные postgres — всё ли указано?

