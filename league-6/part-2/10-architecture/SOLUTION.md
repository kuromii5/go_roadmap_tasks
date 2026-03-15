# 🔑 Решения — Основы архитектуры в Go

---

## Задача 1 — Разложи спагетти

### `internal/repo/task.go`

```go
package repo

import (
	"fmt"
	"github.com/jmoiron/sqlx"
)

type Task struct {
	ID    int    `db:"id"`
	Title string `db:"title"`
	Done  bool   `db:"done"`
}

type TaskRepo struct{ db *sqlx.DB }

func NewTaskRepo(db *sqlx.DB) *TaskRepo {
	return &TaskRepo{db: db}
}

func (r *TaskRepo) Create(title string) (Task, error) {
	var t Task
	err := r.db.QueryRow(
		"INSERT INTO tasks (title, done) VALUES ($1, false) RETURNING id, title, done", title,
	).Scan(&t.ID, &t.Title, &t.Done)
	return t, err
}

func (r *TaskRepo) List() ([]Task, error) {
	var tasks []Task
	err := r.db.Select(&tasks, "SELECT * FROM tasks ORDER BY id")
	return tasks, err
}

func (r *TaskRepo) SetDone(id int, done bool) error {
	res, err := r.db.Exec("UPDATE tasks SET done = $1 WHERE id = $2", done, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("task not found: id=%d", id)
	}
	return nil
}
```

### `internal/service/task.go`

```go
package service

import (
	"fmt"
	"myapp/internal/repo"
)

type TaskService struct{ repo *repo.TaskRepo }

func NewTaskService(r *repo.TaskRepo) *TaskService {
	return &TaskService{repo: r}
}

func (s *TaskService) Create(title string) (repo.Task, error) {
	if title == "" {
		return repo.Task{}, fmt.Errorf("title required")
	}
	if len(title) > 200 {
		return repo.Task{}, fmt.Errorf("title too long")
	}
	return s.repo.Create(title)
}

func (s *TaskService) List() ([]repo.Task, error) {
	return s.repo.List()
}

func (s *TaskService) SetDone(id int, done bool) error {
	return s.repo.SetDone(id, done)
}
```

### `internal/handler/task.go`

```go
package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"myapp/internal/service"
)

type TaskHandler struct{ svc *service.TaskService }

func NewTaskHandler(s *service.TaskService) *TaskHandler {
	return &TaskHandler{svc: s}
}

func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body struct{ Title string `json:"title"` }
	json.NewDecoder(r.Body).Decode(&body)

	task, err := h.svc.Create(body.Title)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(task)
}

func (h *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	tasks, err := h.svc.List()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(tasks)
}

func (h *TaskHandler) SetDone(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))

	var body struct{ Done *bool `json:"done"` }
	json.NewDecoder(r.Body).Decode(&body)
	if body.Done == nil {
		http.Error(w, "done field required", http.StatusBadRequest)
		return
	}

	if err := h.svc.SetDone(id, *body.Done); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Write([]byte("updated"))
}
```

### Что здесь важно

- Handler знает только про HTTP и service. Не импортирует `sqlx`.
- Service знает только про бизнес-правила и repo. Не импортирует `net/http`.
- Repo знает только про SQL. Не знает, кто его вызывает и зачем.
- Поток: **HTTP → Handler → Service → Repo → DB**. Обратных зависимостей нет.

---

## Задача 2 — Найди нарушения

| # | Где | Нарушение | Почему плохо |
|---|-----|-----------|-------------|
| 1 | `repo/user.go` — сигнатура | Repo возвращает HTTP-статус (`int`) | Repo не должен знать про HTTP. Возвращай `error` — пусть handler решает, какой статус отдать |
| 2 | `repo/user.go` — import | Repo импортирует `net/http` | Прямое следствие #1. Repo зависит от транспортного слоя — при смене на gRPC придётся менять repo |
| 3 | `service/order.go` — struct | Service хранит `*sqlx.DB` напрямую | Service не должен знать про базу. Он должен зависеть от repo, а не от коннекта |
| 4 | `service/order.go` — Exec | Service пишет SQL | SQL — ответственность repo. Service считает total и вызывает `repo.Create(userID, total)` |
| 5 | `handler/order.go` — repo | Handler обращается к repo напрямую | Handler → Service → Repo. Handler не должен перепрыгивать через service, иначе бизнес-логика размазывается |

---

## Задача 3 — Добавь интерфейсы

```go
package handler

import "net/http"

// Интерфейс определяет потребитель (handler), а не поставщик (repo)
type TaskRepository interface {
	List() ([]Task, error)
	Save(task Task) (int, error)
}

type TaskHandler struct {
	repo TaskRepository  // интерфейс, не конкретный тип
}

func NewTaskHandler(r TaskRepository) *TaskHandler {
	return &TaskHandler{repo: r}
}
```

### Где жить интерфейсу?

В пакете **потребителя** — то есть в `handler` (или `service`, если service вызывает repo). В Go интерфейс определяет тот, кто использует, а не тот, кто реализует. `PostgresTaskRepo` даже не знает, что реализует `TaskRepository` — implicit implementation.

### Что это даёт?

1. **Тестируемость** — в тестах handler подставляешь мок вместо реальной базы.
2. **Заменяемость** — сегодня PostgreSQL, завтра SQLite, послезавтра in-memory кэш. Handler не меняется.
3. **Независимость сборки** — пакет `handler` не импортирует пакет `repo`. Нет цикла зависимостей.
