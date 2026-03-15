# Практика — Redis

> **Видео:** [Redis основы](https://www.youtube.com/watch?v=GgRzHru9Hag)

---

## Прежде чем начать — go-redis

В видео ты работаешь с Redis напрямую через CLI (`redis-cli`). В Go для работы с Redis используют **[go-redis/redis](https://github.com/redis/go-redis)** — основную библиотеку, которую юзают все.

**Быстрый старт:**

```go
import "github.com/redis/go-redis/v9"

rdb := redis.NewClient(&redis.Options{
    Addr: "localhost:6379",
})

// Все операции принимают context первым аргументом
ctx := context.Background()

rdb.Set(ctx, "key", "value", 10*time.Minute)  // SET с TTL
val, err := rdb.Get(ctx, "key").Result()       // GET
```

**Что даёт:**

- Все команды Redis доступны как методы: `Set`, `Get`, `HSet`, `LPush`, `SAdd`...
- Пайплайны и транзакции
- Pub/Sub
- Автоматический пул соединений (не нужно как с `sql.DB` — уже встроено)
- Каждый метод возвращает типизированный результат (`.Result()`, `.Int()`, `.Bool()`)

**Кто использует:**

go-redis — единственный серьёзный выбор для Go. 20k+ звёзд, поддерживается Redis Ltd. Альтернатив, о которых стоит говорить, нет.

**Документация:**

- **GitHub:** https://github.com/redis/go-redis
- **Docs:** https://redis.uptrace.dev/

> Redis должен быть запущен локально. Если ты проходил тему Docker — `docker run -d -p 6379:6379 redis:alpine`.

---

## Задача 1 — Кэш поверх базы

У тебя есть функция, которая «ходит в базу» за профилем пользователя (имитация через `time.Sleep`). Оберни её кэшем на Redis.

**Готовый код:**

```go
type UserProfile struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
    Bio  string `json:"bio"`
}

// "Запрос в базу" — медленный
func getUserFromDB(id int) (*UserProfile, error) {
    time.Sleep(500 * time.Millisecond) // имитация
    return &UserProfile{
        ID:   id,
        Name: fmt.Sprintf("User_%d", id),
        Bio:  "Go developer",
    }, nil
}
```

**Что нужно написать:**

```go
func GetUser(ctx context.Context, rdb *redis.Client, id int) (*UserProfile, error)
```

Логика:
1. Попробуй достать из Redis по ключу `user:<id>`
2. Если есть (cache hit) — десериализуй JSON и верни
3. Если нет (cache miss) — сходи в `getUserFromDB`, сохрани в Redis с TTL 5 минут, верни
4. Логируй в stdout: `cache hit: user:5` или `cache miss: user:5`

**main:**

```go
func main() {
    // 1. Создай redis.Client
    // 2. Вызови GetUser(ctx, rdb, 1) — cache miss, ~500ms
    // 3. Вызови GetUser(ctx, rdb, 1) — cache hit, ~0ms
    // 4. Вызови GetUser(ctx, rdb, 2) — cache miss
    // Выводи время каждого вызова
}
```

**Ожидаемый вывод:**

```
cache miss: user:1 (501ms)
cache hit: user:1 (1ms)
cache miss: user:2 (500ms)
```

**Требования:**

- Сериализация через `encoding/json`
- Ключ формата `user:<id>`
- TTL = 5 минут
- Обработка `redis.Nil` — это не ошибка, а cache miss

---

## Задача 2 — Rate limiter

Напиши middleware для chi, который ограничивает количество запросов по IP-адресу: **не более 10 запросов в минуту**.

**Как реализовать:**

Используй Redis-команду `INCR` + `EXPIRE`:
1. Ключ: `ratelimit:<ip>`
2. `INCR` увеличивает счётчик на 1
3. Если счётчик стал 1 (первый запрос) — поставь `EXPIRE` 60 секунд
4. Если счётчик > 10 — верни `429 Too Many Requests`

**Сигнатура:**

```go
func RateLimitMiddleware(rdb *redis.Client) func(http.Handler) http.Handler
```

**Тестовый сервер:**

```go
func main() {
    rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})

    r := chi.NewRouter()
    r.Use(RateLimitMiddleware(rdb))
    r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("pong"))
    })

    http.ListenAndServe(":8080", r)
}
```

**Проверка:**

```bash
# Отправь 12 запросов подряд
for i in $(seq 1 12); do
    echo "Request $i: $(curl -s -o /dev/null -w '%{http_code}' http://localhost:8080/ping)"
done
```

**Ожидаемый вывод:**

```
Request 1: 200
...
Request 10: 200
Request 11: 429
Request 12: 429
```

**Требования:**

- IP бери из `r.RemoteAddr` (для локалки будет `127.0.0.1:порт` — используй только IP-часть)
- Заголовок `Retry-After: 60` в ответе 429
- Не используй Lua-скрипты — обычные команды

---

## Задача 3 — Сессии

Реализуй простое хранилище сессий на Redis. Без JWT — сессия живёт только в Redis.

**Что нужно написать:**

```go
type SessionStore struct {
    rdb *redis.Client
    ttl time.Duration
}

func NewSessionStore(rdb *redis.Client, ttl time.Duration) *SessionStore
```

Методы:

| Метод | Что делает |
|-------|-----------|
| `Create(ctx, userID) (sessionID, error)` | Генерирует уникальный ID, сохраняет `session:<id>` → `userID` с TTL |
| `Get(ctx, sessionID) (userID, error)` | Возвращает userID по ID сессии. Если истекла/нет — ошибка |
| `Delete(ctx, sessionID) error` | Удаляет сессию (логаут) |
| `Extend(ctx, sessionID) error` | Продлевает TTL сессии (юзер активен) |

**Два эндпоинта для проверки:**

```go
// POST /login — принимает {"user_id": "42"}, создаёт сессию, возвращает session_id
// GET /me — читает session_id из заголовка X-Session-ID, возвращает user_id
// POST /logout — удаляет сессию
```

**Проверка:**

```bash
# Логин
SID=$(curl -s -X POST http://localhost:8080/login \
  -d '{"user_id":"42"}' | jq -r '.session_id')

echo "session: $SID"

# Кто я
curl -H "X-Session-ID: $SID" http://localhost:8080/me

# Логаут
curl -X POST -H "X-Session-ID: $SID" http://localhost:8080/logout

# Кто я после логаута — должен быть 401
curl -H "X-Session-ID: $SID" http://localhost:8080/me
```

**Требования:**

- ID сессии — `uuid` (используй любой пакет или `crypto/rand`)
- TTL — 30 минут
- `Get` при отсутствии сессии — понятная ошибка, не паника
- `Extend` — сбрасывает TTL заново (юзер кликнул → сессия продлена)

