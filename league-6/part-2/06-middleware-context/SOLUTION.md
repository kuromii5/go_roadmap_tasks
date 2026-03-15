# 🔑 Решения — Middleware, Context в Go (chi)

---

## Задача 1 — Три middleware

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// --- ResponseWriter обёртка для перехвата статус-кода ---

type wrappedWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *wrappedWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

// --- Middleware 1: Logging ---

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		wrapped := &wrappedWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(wrapped, r)

		log.Printf("%s %s %d %s", r.Method, r.URL.Path, wrapped.statusCode, time.Since(start))
	})
}

// --- Middleware 2: Auth ---

type contextKey string

const userIDKey contextKey = "userID"

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token != "Bearer valid-token-123" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, "42")
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// --- Middleware 3: RequestID ---

const requestIDKey contextKey = "requestID"

func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := fmt.Sprintf("%d", time.Now().UnixNano())

		ctx := context.WithValue(r.Context(), requestIDKey, reqID)
		w.Header().Set("X-Request-ID", reqID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// --- Main ---

func main() {
	r := chi.NewRouter()

	// Глобальные middleware — для всех роутов
	r.Use(RequestIDMiddleware) // первый: ID нужен всем, включая логи
	r.Use(LoggingMiddleware)   // второй: логирует всё, даже 401

	// Публичные роуты
	r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("pong"))
	})

	// Защищённые роуты
	r.Group(func(r chi.Router) {
		r.Use(AuthMiddleware)

		r.Get("/me", func(w http.ResponseWriter, r *http.Request) {
			userID := r.Context().Value(userIDKey)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"user_id": userID,
			})
		})
	})

	http.ListenAndServe(":8080", r)
}
```

### Что здесь важно

- **Порядок middleware** — RequestID → Logging → Auth. Logging должен быть снаружи Auth, чтобы логировать и неавторизованные запросы. RequestID — самый внешний, чтобы ID был доступен всем.
- **`wrappedWriter`** — стандартный `http.ResponseWriter` не даёт узнать статус-код после записи. Обёртка перехватывает `WriteHeader` и сохраняет код.
- **`r.Group`** — middleware применяется только к роутам внутри группы. `/ping` работает без авторизации, `/me` — только с ней.
- **`contextKey` как тип** — `context.WithValue` с ключом `"userID"` (строка) опасен: любой пакет может случайно использовать тот же ключ. Кастомный тип `contextKey` исключает коллизии.

---

## Задача 2 — Context timeout

```go
package main

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

func heavyOperation(ctx context.Context) (string, error) {
	select {
	case <-time.After(3 * time.Second):
		return "data from db", nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func main() {
	r := chi.NewRouter()

	r.Get("/slow-query", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		result, err := heavyOperation(ctx)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				http.Error(w, "gateway timeout", http.StatusGatewayTimeout)
				return
			}
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.Write([]byte(result))
	})

	http.ListenAndServe(":8080", r)
}
```

### Что здесь важно

- **`defer cancel()`** — обязательно. Без него контекст утечёт и будет висеть до истечения таймаута, даже если операция завершилась раньше. Это утечка ресурсов.
- **`context.WithTimeout` от `r.Context()`** — дочерний контекст. Если клиент разорвёт соединение, родительский контекст отменится → дочерний тоже. Цепочка отмены работает автоматически.
- **`select` в `heavyOperation`** — паттерн «гонка» между результатом и отменой. В реальном коде вместо `time.After` будет запрос к базе или внешнему сервису, который тоже принимает `ctx`.
- **`context.DeadlineExceeded` vs `context.Canceled`** — `DeadlineExceeded` = таймаут истёк. `Canceled` = кто-то вызвал `cancel()` вручную или клиент отключился.

### middleware.Timeout из chi

```go
r.Use(middleware.Timeout(2 * time.Second))
```

Разница: `middleware.Timeout` ставит таймаут на **весь** запрос глобально. Ручной `context.WithTimeout` — на конкретную операцию внутри хендлера. В реальности используют оба: глобальный таймаут как страховку + точечные таймауты на тяжёлые операции.

---

## Задача 3 — Собери пайплайн

### 3 проблемы:

| # | Где | Проблема |
|---|-----|----------|
| 1 | `AuthMiddleware` | Нет `return` после `http.Error`. Запрос с невалидным токеном получит 401, но выполнение **продолжится** — `next.ServeHTTP` всё равно вызовется. Результат: 401 в заголовке, но тело от хендлера тоже запишется |
| 2 | Порядок `r.Use(...)` | Auth подключен **перед** Logging. Логирование не увидит неавторизованные запросы — они отвалятся до логера. Logging должен быть первым |
| 3 | `context.WithValue(r.Context(), "role", ...)` | Ключ `"role"` — голая строка. Любой пакет может случайно перезаписать. Нужен кастомный тип ключа |

### Исправленный код:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

type contextKey string

const roleKey contextKey = "role"

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token != "Bearer secret" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return // ← FIX: без return запрос пойдёт дальше
		}

		ctx := context.WithValue(r.Context(), roleKey, "admin") // ← FIX: кастомный тип ключа
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func AdminOnlyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role := r.Context().Value(roleKey)
		if role != "admin" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	r := chi.NewRouter()

	r.Use(LoggingMiddleware) // ← FIX: логирование первым — видит все запросы
	r.Use(AuthMiddleware)

	r.Get("/dashboard", func(w http.ResponseWriter, r *http.Request) {
		role := r.Context().Value(roleKey)
		fmt.Fprintf(w, "welcome, %s", role)
	})

	r.Route("/admin", func(r chi.Router) {
		r.Use(AdminOnlyMiddleware)
		r.Get("/stats", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("secret stats"))
		})
	})

	http.ListenAndServe(":8080", r)
}
```

### Что здесь важно

- **Забытый `return`** — самый коварный баг в middleware. Код работает «почти правильно»: клиент получает 401, но хендлер всё равно выполняется. В лучшем случае — лишняя работа. В худшем — неавторизованный пользователь получает данные.
- **Порядок middleware = порядок матрёшки.** Первый `Use` — самый внешний слой. Logging снаружи видит всё. Auth внутри — фильтрует до хендлера.
- **Строковые ключи в context** — бомба замедленного действия. Два пакета используют `"role"` — один перезапишет другой. Кастомный неэкспортируемый тип решает проблему.
