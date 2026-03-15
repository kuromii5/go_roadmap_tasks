# Финальный проект — One-Time Secret

Сервис одноразовых секретов. Отправляешь конфиденциальный текст (пароль, ключ, сообщение) — получаешь ссылку. Получатель открывает ссылку **один раз** — секрет удаляется навсегда.

Аналог: [onetimesecret.com](https://onetimesecret.com)

---

## Эндпоинты

### `POST /api/secrets` — создать секрет

Принимает: текст секрета, опциональный passphrase (доп. защита), время жизни в минутах (по умолчанию 24 часа, максимум 7 дней).

Возвращает: ID секрета, ссылку, время истечения. **Текст секрета не возвращается.**

### `GET /api/secrets/{id}` — прочитать секрет

Возвращает текст секрета и **удаляет его**. Повторный запрос — `404`.

Если при создании был passphrase — требуй его в заголовке `X-Passphrase`. Неверный — `403`.

Если секрет просрочен — `410 Gone`.

### `DELETE /api/secrets/{id}` — удалить секрет вручную

Автор может удалить секрет до того, как его прочитают. `204 No Content`.

### `GET /api/health` — healthcheck

Статус сервиса, подключение к PostgreSQL и Redis.

---

## Требования

### PostgreSQL

Хранит секреты. Текст не должен лежать в открытом виде — минимум base64, в идеале шифрование. Passphrase — хешированный (bcrypt). После прочтения секрет удаляется из базы.

### Redis

- **Кэш** горячих секретов с TTL (чтобы не ходить в PostgreSQL каждый раз)
- **Rate limiter** на создание секретов — не более 5 в минуту с одного IP

### Middleware

- Логирование запросов (метод, путь, статус, время выполнения)
- Rate limiter на `POST /api/secrets`
- Request ID в каждом запросе

### Docker

`docker-compose up --build` должен поднять всё: приложение, PostgreSQL, Redis. Данные PostgreSQL не теряются при перезапуске.

### Остальное

- Миграции через goose
- Makefile с основными командами (run, build, migrate, compose)
- Конфигурация через `.env`
- README.md — что за проект, как запустить, примеры curl-запросов

---

## Проверка

```bash
# Создать
curl -X POST http://localhost:8080/api/secrets \
  -d '{"text": "пароль: qwerty123", "expires_in_minutes": 30}'

# Прочитать — работает
curl http://localhost:8080/api/secrets/<id>

# Прочитать повторно — 404
curl http://localhost:8080/api/secrets/<id>

# С passphrase
curl -X POST http://localhost:8080/api/secrets \
  -d '{"text": "ключ", "passphrase": "1234"}'

curl -H "X-Passphrase: wrong" http://localhost:8080/api/secrets/<id>  # 403
curl -H "X-Passphrase: 1234" http://localhost:8080/api/secrets/<id>   # 200

# Rate limit — 6-й запрос подряд должен вернуть 429
```
