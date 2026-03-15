# Практика — Middleware и Context

> **Видео:** [Middleware и Context](https://youtu.be/2cxmJUJ2Ge0)

---

## Прежде чем начать — chi

В видео ты увидишь middleware на стандартном `net/http`. Это работает, но в продакшене большинство Go-проектов используют **[go-chi/chi](https://github.com/go-chi/chi)** — легковесный роутер, построенный поверх стандартной библиотеки.

**Зачем chi, если есть net/http?**

Стандартный `http.ServeMux` не умеет:
- URL-параметры (`/users/{id}`)
- Группировку роутов с общими middleware
- Удобную цепочку middleware

```go
// net/http — всё руками
mux := http.NewServeMux()
mux.HandleFunc("/users/", func(w http.ResponseWriter, r *http.Request) {
    id := strings.TrimPrefix(r.URL.Path, "/users/")
    // ...
})
```

```go
// chi — чисто и выразительно
r := chi.NewRouter()
r.Get("/users/{id}", func(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")
    // ...
})
```

**Что даёт chi:**

- URL-параметры через `{name}`
- `r.Use(middleware)` — глобальные middleware
- `r.Group` / `r.Route` — middleware на группу роутов
- Встроенные middleware: `middleware.Logger`, `middleware.Recoverer`, `middleware.Timeout`
- 100% совместим с `net/http` — `chi.Router` реализует `http.Handler`
- Нулевые внешние зависимости

**Кто использует:**

chi — самый популярный роутер в Go-экосистеме. 19k+ звёзд на GitHub. Используется в Cloudflare, Heroku, 99designs. Его выбирают за то, что он не пытается быть фреймворком — просто роутер и middleware, без магии.

**Документация:**

- **GitHub:** https://github.com/go-chi/chi
- **Примеры:** https://github.com/go-chi/chi/tree/master/_examples

> В задачах ниже мы **не объясняем** API chi — разберись сам по документации и примерам. Это часть задания.

---

## Задача 1 — Три middleware

Дан готовый сервер на chi с двумя эндпоинтами. Твоя задача — написать 3 middleware и подключить их.

**Готовый код (не меняй):**

```go
package main

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func main() {
	r := chi.NewRouter()

	// TODO: подключи middleware здесь

	r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("pong"))
	})

	r.Get("/me", func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("userID")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"user_id": userID,
		})
	})

	http.ListenAndServe(":8080", r)
}
```

**Middleware 1 — Logging:**

Логирует каждый запрос **после** его выполнения:

```
GET /ping 200 1.23ms
GET /me 401 0.15ms
```

Что нужно: метод, путь, статус-код, время выполнения. Для перехвата статус-кода тебе понадобится обернуть `http.ResponseWriter` — разберись как.

**Middleware 2 — Auth:**

Проверяет заголовок `Authorization`:
- Если заголовок `Bearer valid-token-123` → кладёт `userID = "42"` в `context` и пропускает дальше
- Если заголовок отсутствует или невалидный → возвращает `401 Unauthorized`, дальше запрос не идёт

Для записи в context используй `context.WithValue` и `r.WithContext`.

**Middleware 3 — RequestID:**

Генерирует уникальный ID запроса и:
- Кладёт его в `context`
- Добавляет заголовок `X-Request-ID` в ответ

Для генерации ID используй `fmt.Sprintf("%d", time.Now().UnixNano())` или `uuid`.

**Подключение:**

Подключи middleware в правильном порядке через `r.Use(...)`. Подумай — какой должен быть первым, какой последним, и почему.

**Проверка через curl:**

```bash
# Без токена — 401
curl -v http://localhost:8080/me

# С токеном — 200 + userID в теле + X-Request-ID в заголовке
curl -v -H "Authorization: Bearer valid-token-123" http://localhost:8080/me

# ping — 200 (тоже требует токен? или нет? — подумай)
curl http://localhost:8080/ping
```

**Бонус:**

Сделай так, чтобы `/ping` работал **без** авторизации, а `/me` — только с ней. Посмотри в документации chi, как применять middleware не глобально, а к группе роутов.

---

## Задача 2 — Context timeout

Напиши эндпоинт, который имитирует тяжёлый запрос и корректно обрабатывает таймаут через `context`.

**Что нужно сделать:**

```go
r.Get("/slow-query", func(w http.ResponseWriter, r *http.Request) {
    // 1. Создай контекст с таймаутом 2 секунды из r.Context()
    // 2. Вызови heavyOperation(ctx)
    // 3. Если успел — верни результат (200)
    // 4. Если таймаут — верни 504 Gateway Timeout
})
```

Функция `heavyOperation` — имитация запроса к базе:

```go
func heavyOperation(ctx context.Context) (string, error) {
    select {
    case <-time.After(3 * time.Second): // "запрос" длится 3 секунды
        return "data from db", nil
    case <-ctx.Done():
        return "", ctx.Err()
    }
}
```

**Проверка:**

```bash
# Должен вернуть 504 — операция длится 3с, таймаут 2с
curl http://localhost:8080/slow-query
```

Теперь поменяй таймаут на 5 секунд — запрос должен вернуть 200.

**Дополнительно:**

Попробуй вместо ручного `context.WithTimeout` использовать `middleware.Timeout` из chi. В чём разница? Какой подход где уместен?

---

## Задача 3 — Собери пайплайн

Дан набор middleware и `main`. Код компилируется, но работает неправильно. Найди **3 проблемы** и исправь.

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
		}

		ctx := context.WithValue(r.Context(), "role", "admin")
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func AdminOnlyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role := r.Context().Value("role")
		if role != "admin" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	r := chi.NewRouter()

	r.Use(AuthMiddleware)
	r.Use(LoggingMiddleware)

	r.Get("/dashboard", func(w http.ResponseWriter, r *http.Request) {
		role := r.Context().Value("role")
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

**Что не работает:**

Протестируй:

```bash
# Без токена — ожидаем 401, но что реально происходит?
curl http://localhost:8080/dashboard

# С токеном — логи должны показывать время, но в каком порядке?
curl -H "Authorization: Bearer secret" http://localhost:8080/dashboard
```

**Подсказки:**

1. Что произойдёт в `AuthMiddleware`, если токен неверный? Выполнится ли `next.ServeHTTP`?
2. В каком порядке вызываются `r.Use(AuthMiddleware)` и `r.Use(LoggingMiddleware)`? Что логируется — все запросы или только авторизованные?
3. Запрос без токена на `/dashboard` — какой статус он реально вернёт?

