# 🔑 Решения — Redis + Go

---

## Задача 1 — Кэш поверх базы

```go
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

type UserProfile struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Bio  string `json:"bio"`
}

func getUserFromDB(id int) (*UserProfile, error) {
	time.Sleep(500 * time.Millisecond)
	return &UserProfile{
		ID:   id,
		Name: fmt.Sprintf("User_%d", id),
		Bio:  "Go developer",
	}, nil
}

func GetUser(ctx context.Context, rdb *redis.Client, id int) (*UserProfile, error) {
	key := fmt.Sprintf("user:%d", id)

	// Пробуем кэш
	val, err := rdb.Get(ctx, key).Result()
	if err == nil {
		// cache hit
		var profile UserProfile
		if err := json.Unmarshal([]byte(val), &profile); err != nil {
			return nil, fmt.Errorf("unmarshal cache: %w", err)
		}
		fmt.Printf("cache hit: %s\n", key)
		return &profile, nil
	}

	if !errors.Is(err, redis.Nil) {
		return nil, fmt.Errorf("redis get: %w", err)
	}

	// cache miss — идём в базу
	profile, err := getUserFromDB(id)
	if err != nil {
		return nil, fmt.Errorf("db: %w", err)
	}

	// Сохраняем в кэш
	data, err := json.Marshal(profile)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	if err := rdb.Set(ctx, key, data, 5*time.Minute).Err(); err != nil {
		// Кэш не записался — не фатально, логируем и отдаём данные
		log.Printf("warning: cache set failed: %v", err)
	}

	fmt.Printf("cache miss: %s\n", key)
	return profile, nil
}

func main() {
	ctx := context.Background()
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	defer rdb.Close()

	for _, id := range []int{1, 1, 2} {
		start := time.Now()
		user, err := GetUser(ctx, rdb, id)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("  → %s (%s)\n\n", user.Name, time.Since(start).Truncate(time.Millisecond))
	}
}
```

### Что здесь важно

- **`redis.Nil`** — не ошибка, а индикатор отсутствия ключа. Обрабатывается отдельно от реальных ошибок Redis (сеть, таймаут).
- **Ошибка записи в кэш — не фатальна.** Если Redis упал, данные всё равно есть в базе. Логируем и отдаём. Кэш — это оптимизация, а не источник правды.
- **TTL** — без него ключи живут вечно. Обновил профиль в базе → кэш устарел. TTL гарантирует, что максимум через 5 минут кэш обновится.
- **JSON** — Redis хранит строки/байты. Структуры нужно сериализовать. Альтернатива — `HSet`/`HGetAll` для маппинга полей в Redis hash, но JSON проще для начала.

---

## Задача 2 — Rate limiter

```go
package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
)

func RateLimitMiddleware(rdb *redis.Client) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			ip, _, _ := net.SplitHostPort(r.RemoteAddr)
			key := fmt.Sprintf("ratelimit:%s", ip)

			count, err := rdb.Incr(ctx, key).Result()
			if err != nil {
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}

			// Первый запрос — ставим TTL
			if count == 1 {
				rdb.Expire(ctx, key, 60*time.Second)
			}

			if count > 10 {
				w.Header().Set("Retry-After", "60")
				http.Error(w, "too many requests", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func main() {
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})

	r := chi.NewRouter()
	r.Use(RateLimitMiddleware(rdb))
	r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("pong"))
	})

	fmt.Println("listening :8080")
	http.ListenAndServe(":8080", r)
}
```

### Что здесь важно

- **`INCR` атомарен** — даже при параллельных запросах счётчик не сломается. Redis выполняет команды последовательно (single-threaded).
- **Race condition с `EXPIRE`** — между `INCR` и `EXPIRE` сервер может упасть. Ключ останется без TTL → вечный блок. В продакшене используют Lua-скрипт, который делает обе операции атомарно. Для учебной задачи — ок.
- **`net.SplitHostPort`** — `RemoteAddr` приходит в формате `127.0.0.1:54321`. Нужен только IP.
- Этот подход называется **fixed window** — простой, но неточный на границах окна. Продвинутые варианты: sliding window, token bucket.

---

## Задача 3 — Сессии

```go
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
)

// --- Session Store ---

type SessionStore struct {
	rdb *redis.Client
	ttl time.Duration
}

func NewSessionStore(rdb *redis.Client, ttl time.Duration) *SessionStore {
	return &SessionStore{rdb: rdb, ttl: ttl}
}

func (s *SessionStore) Create(ctx context.Context, userID string) (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate session id: %w", err)
	}
	sessionID := hex.EncodeToString(b)

	key := fmt.Sprintf("session:%s", sessionID)
	if err := s.rdb.Set(ctx, key, userID, s.ttl).Err(); err != nil {
		return "", fmt.Errorf("create session: %w", err)
	}
	return sessionID, nil
}

func (s *SessionStore) Get(ctx context.Context, sessionID string) (string, error) {
	key := fmt.Sprintf("session:%s", sessionID)
	userID, err := s.rdb.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", fmt.Errorf("session not found or expired")
	}
	if err != nil {
		return "", fmt.Errorf("get session: %w", err)
	}
	return userID, nil
}

func (s *SessionStore) Delete(ctx context.Context, sessionID string) error {
	key := fmt.Sprintf("session:%s", sessionID)
	if err := s.rdb.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

func (s *SessionStore) Extend(ctx context.Context, sessionID string) error {
	key := fmt.Sprintf("session:%s", sessionID)
	ok, err := s.rdb.Expire(ctx, key, s.ttl).Result()
	if err != nil {
		return fmt.Errorf("extend session: %w", err)
	}
	if !ok {
		return fmt.Errorf("session not found or expired")
	}
	return nil
}

// --- Handlers ---

func main() {
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	store := NewSessionStore(rdb, 30*time.Minute)

	r := chi.NewRouter()

	r.Post("/login", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			UserID string `json:"user_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		sid, err := store.Create(r.Context(), body.UserID)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]string{"session_id": sid})
	})

	r.Get("/me", func(w http.ResponseWriter, r *http.Request) {
		sid := r.Header.Get("X-Session-ID")
		if sid == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		userID, err := store.Get(r.Context(), sid)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		// Продлеваем сессию при активности
		store.Extend(r.Context(), sid)

		json.NewEncoder(w).Encode(map[string]string{"user_id": userID})
	})

	r.Post("/logout", func(w http.ResponseWriter, r *http.Request) {
		sid := r.Header.Get("X-Session-ID")
		if sid == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		if err := store.Delete(r.Context(), sid); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	})

	fmt.Println("listening :8080")
	http.ListenAndServe(":8080", r)
}
```

### Что здесь важно

- **`crypto/rand`** — для генерации session ID. Не `math/rand` — тот предсказуем, можно угадать чужую сессию.
- **`Expire` возвращает `bool`** — `true` если ключ существует, `false` если нет. Это встроенная проверка «сессия ещё жива?».
- **`Extend` при каждом запросе** — паттерн sliding expiration. Пока юзер активен, сессия не протухает. 30 минут бездействия → автоматический логаут.
- **Redis как session store** — классический кейс. Быстро (in-memory), TTL из коробки, горизонтально масштабируется. Альтернатива — хранить сессии в PostgreSQL, но это медленнее и TTL руками.
- **Нет JWT** — намеренно. JWT и серверные сессии — разные подходы с разными trade-offs. Здесь — серверные сессии, stateful. Состояние живёт в Redis.
