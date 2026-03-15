# Практика — Архитектура

> **Видео:** [Архитектура в Go](https://youtu.be/lc3ATNxWQbI) (до ~1:17:00)

---

## Задача 1 — Разложи спагетти

Весь код ниже живёт в одном `main.go`. Всё работает, но масштабировать невозможно.

**Твоя задача:** разложи на 3 слоя — `handler/`, `service/`, `repo/`. Каждый слой — отдельный пакет.

```go
package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

var db *sqlx.DB

type Task struct {
	ID    int    `db:"id" json:"id"`
	Title string `db:"title" json:"title"`
	Done  bool   `db:"done" json:"done"`
}

func main() {
	db, _ = sqlx.Connect("postgres", "postgres://user:pass@localhost:5432/tasks?sslmode=disable")

	r := chi.NewRouter()

	r.Post("/tasks", func(w http.ResponseWriter, r *http.Request) {
		var t Task
		json.NewDecoder(r.Body).Decode(&t)

		if t.Title == "" {
			http.Error(w, "title required", 400)
			return
		}
		if len(t.Title) > 200 {
			http.Error(w, "title too long", 400)
			return
		}

		var id int
		db.QueryRow("INSERT INTO tasks (title, done) VALUES ($1, $2) RETURNING id", t.Title, false).Scan(&id)
		t.ID = id
		t.Done = false

		w.WriteHeader(201)
		json.NewEncoder(w).Encode(t)
	})

	r.Get("/tasks", func(w http.ResponseWriter, r *http.Request) {
		var tasks []Task
		db.Select(&tasks, "SELECT * FROM tasks ORDER BY id")
		json.NewEncoder(w).Encode(tasks)
	})

	r.Patch("/tasks/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		var body struct {
			Done *bool `json:"done"`
		}
		json.NewDecoder(r.Body).Decode(&body)

		if body.Done == nil {
			http.Error(w, "done field required", 400)
			return
		}

		res, _ := db.Exec("UPDATE tasks SET done = $1 WHERE id = $2", *body.Done, id)
		n, _ := res.RowsAffected()
		if n == 0 {
			http.Error(w, "not found", 404)
			return
		}

		w.Write([]byte("updated"))
	})

	http.ListenAndServe(":8080", r)
}
```

**Результат:**

```
internal/
├── handler/
│   └── task.go      ← HTTP: decode, encode, статус-коды
├── service/
│   └── task.go      ← Бизнес-логика: валидация
└── repo/
    └── task.go       ← SQL-запросы
```

**Правила:**

- Handler не знает про SQL
- Repo не знает про HTTP
- Service не знает ни про HTTP, ни про SQL-библиотеку напрямую — только вызывает repo
- Валидация (`title == ""`, `len > 200`) — это бизнес-логика → service

---

## Задача 2 — Найди нарушения

Проект разложен по слоям, но в нём **5 нарушений** архитектурных границ. Найди каждое и объясни, почему это плохо.

```go
// --- repo/user.go ---
package repo

import (
	"net/http"
	"github.com/jmoiron/sqlx"
)

type UserRepo struct{ db *sqlx.DB }

func (r *UserRepo) GetByID(id int) (*User, int) {        // (1)
	var u User
	err := r.db.Get(&u, "SELECT * FROM users WHERE id = $1", id)
	if err != nil {
		return nil, http.StatusNotFound                   // (2)
	}
	return &u, http.StatusOK
}
```

```go
// --- service/order.go ---
package service

import "github.com/jmoiron/sqlx"

type OrderService struct{ db *sqlx.DB }                   // (3)

func (s *OrderService) Create(userID int, items []Item) error {
	var total float64
	for _, i := range items {
		total += i.Price
	}
	_, err := s.db.Exec(                                  // (4)
		"INSERT INTO orders (user_id, total) VALUES ($1, $2)",
		userID, total,
	)
	return err
}
```

```go
// --- handler/order.go ---
package handler

import "myapp/internal/repo"

type OrderHandler struct {
	repo    *repo.OrderRepo                               // (5)
	service *service.OrderService
}

func (h *OrderHandler) Create(w http.ResponseWriter, r *http.Request) {
	// ...
	order, err := h.repo.GetByUserID(userID)              // (5)
	// ...
}
```

**Формат ответа:**

| # | Где | Нарушение | Почему плохо |
|---|-----|-----------|-------------|
| 1 | ... | ... | ... |

---

## Задача 3 — Добавь интерфейсы

Дан handler, который зависит от конкретной реализации:

```go
package handler

import "myapp/internal/repo"

type TaskHandler struct {
	repo *repo.PostgresTaskRepo  // конкретный тип
}

func (h *TaskHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	tasks, err := h.repo.List()
	// ...
}

func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	// ...
	id, err := h.repo.Save(task)
	// ...
}
```

**Что нужно сделать:**

1. Определи интерфейс `TaskRepository` — где он должен жить? В пакете `handler`, `service`, или `repo`?
2. Перепиши `TaskHandler`, чтобы он зависел от интерфейса
3. Ответь: что это даёт? Зачем нужен интерфейс, если `PostgresTaskRepo` и так работает?

**Подсказка:**

В Go интерфейс определяет **потребитель**, а не поставщик. Это принципиальное отличие от Java/C#.

