# Практика — PostgreSQL

> **Видео:** [PostgreSQL и Go](https://youtu.be/MNyNxloZR0k?t=7508)

---

## Прежде чем начать — sqlx

В видео ты увидишь работу через стандартный `database/sql`. Он работает, но в продакшене почти все Go-проекты используют **[jmoiron/sqlx](https://github.com/jmoiron/sqlx)** — обёртку поверх `database/sql`, которая убирает боль.

**Зачем sqlx, если есть database/sql?**

Стандартная библиотека заставляет руками перечислять каждое поле в `Scan`:

```go
// database/sql — ручной маппинг
row := db.QueryRow("SELECT id, name, price, in_stock FROM products WHERE id = $1", id)
err := row.Scan(&p.ID, &p.Name, &p.Price, &p.InStock)
```

Добавил колонку в таблицу — иди обновляй каждый `Scan` в проекте. `sqlx` маппит строки на структуры автоматически через теги `db`:

```go
// sqlx — автомаппинг
var p Product
err := db.Get(&p, "SELECT * FROM products WHERE id = $1", id)
```

**Что даёт sqlx:**

- `Get` — одна строка → структура
- `Select` — много строк → слайс структур
- `NamedExec` — запросы с именованными параметрами (`:name` вместо `$1`)
- `StructScan` — маппинг через теги `db:"column_name"`
- Полная совместимость с `database/sql` — `sqlx.DB` оборачивает `sql.DB` внутри

**Кто использует:**

`sqlx` — стандарт де-факто в Go-проектах. 16k+ звёзд на GitHub. Используется в проектах Cloudflare, HashiCorp и сотнях production-сервисов. Если откроешь вакансию Go-разработчика — с высокой вероятностью увидишь `sqlx` в стеке.

**Документация:**

- **GitHub:** https://github.com/jmoiron/sqlx
- **Illustrated guide:** https://jmoiron.github.io/sqlx/

> В задачах ниже мы **не объясняем** как работает каждый метод `sqlx` — разберись сам по документации. Это часть задания.

---

## Задача 1 — CRUD на коленке

Напиши программу, которая управляет таблицей `bookmarks` — простое хранилище ссылок.

**Шаг 1 — Подготовь базу:**

```sql
CREATE TABLE bookmarks (
    id SERIAL PRIMARY KEY,
    url TEXT NOT NULL,
    title VARCHAR(200) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);
```

**Шаг 2 — Структура:**

```go
type Bookmark struct {
    ID        int    `db:"id"`
    URL       string `db:"url"`
    Title     string `db:"title"`
    CreatedAt string `db:"created_at"`
}
```

**Шаг 3 — Функции:**

Напиши 4 функции, каждая принимает `*sqlx.DB`:

| Функция | Что делает | Подсказка по sqlx |
|---------|-----------|-------------------|
| `AddBookmark(db, url, title)` | `INSERT`, возвращает `id` | `QueryRow` + `RETURNING id` работает так же |
| `GetAll(db) []Bookmark` | `SELECT *`, возвращает слайс | Попробуй `Select` |
| `UpdateTitle(db, id, newTitle)` | `UPDATE` заголовка | Попробуй `NamedExec` или `MustExec` |
| `Delete(db, id)` | `DELETE` по `id` | Проверь `RowsAffected` |

**Шаг 4 — main:**

```go
func main() {
    // 1. Подключись через sqlx.Connect (не sql.Open!)
    // 2. DATABASE_URL из переменной окружения
    // 3. Добавь 3 закладки
    // 4. Выведи все
    // 5. Обнови заголовок первой
    // 6. Удали вторую
    // 7. Выведи все снова
}
```

**Ожидаемый вывод (примерный):**

```
added: id=1
added: id=2
added: id=3

all bookmarks:
  [1] Go docs — https://go.dev/doc
  [2] Postgres tutorial — https://postgresqltutorial.com
  [3] LeetCode — https://leetcode.com

updated id=1 title → "Go official docs"
deleted id=2

all bookmarks:
  [1] Go official docs — https://go.dev/doc
  [3] LeetCode — https://leetcode.com
```

**Требования:**

- Используй `sqlx` + `lib/pq` (или `pgx`)
- Подключение через `sqlx.Connect` — он сразу делает `Ping`, в отличие от `sql.Open`
- Для `GetAll` используй `sqlx.Select` — без ручного `rows.Next()` + `Scan`
- `DATABASE_URL` из `os.Getenv`
- `UpdateTitle` и `Delete` — проверяй `RowsAffected()`. Если 0 — ошибка

---

## Задача 2 — Грязные руки

Ниже — код, который «работает», но содержит **4 проблемы**. Найди и исправь.

```go
package main

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type User struct {
	ID    int    `db:"id"`
	Email string `db:"email"`
}

func main() {
	db, _ := sqlx.Connect("postgres", "postgres://user:pass@localhost:5432/testdb?sslmode=disable")

	var name string
	fmt.Print("enter username: ")
	fmt.Scan(&name)

	query := fmt.Sprintf("SELECT id, email FROM users WHERE name = '%s'", name)

	var users []User
	db.Select(&users, query)

	for _, u := range users {
		fmt.Printf("id=%d email=%s\n", u.ID, u.Email)
	}
}
```

**Что нужно сделать:**

Перепиши код так, чтобы:

1. Ни одна ошибка не была проигнорирована
2. Не было SQL-инъекции
3. Все ресурсы корректно закрывались
4. Пустой результат обрабатывался корректно

**Подсказка:**

Попробуй ввести `' OR '1'='1` в оригинальный код и подумай, что произойдёт.

---

## Задача 3 — Мини-репозиторий

Оберни работу с таблицей `products` в структуру-репозиторий.

**Шаг 1 — Таблица:**

```sql
CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    price NUMERIC(10,2) NOT NULL,
    in_stock BOOLEAN DEFAULT true
);
```

**Шаг 2 — Структура:**

```go
type Product struct {
    ID      int     `db:"id"`
    Name    string  `db:"name"`
    Price   float64 `db:"price"`
    InStock bool    `db:"in_stock"`
}

type ProductRepo struct {
    db *sqlx.DB
}

func NewProductRepo(db *sqlx.DB) *ProductRepo {
    return &ProductRepo{db: db}
}
```

**Шаг 3 — Методы:**

| Метод | Что делает | Подсказка по sqlx |
|-------|-----------|-------------------|
| `Create(p Product) (int, error)` | `INSERT ... RETURNING id` | Можешь попробовать `NamedQuery` |
| `GetByID(id int) (Product, error)` | Одна строка по id | `Get` — вернёт `sql.ErrNoRows` если не найден |
| `ListInStock() ([]Product, error)` | Все где `in_stock=true` | `Select` |
| `UpdatePrice(id int, price float64) error` | Обновить цену | Проверь `RowsAffected` |
| `SoftDelete(id int) error` | `UPDATE in_stock=false` | Не `DELETE` — мягкое удаление |

**Шаг 4 — main:**

```go
func main() {
    // 1. sqlx.Connect
    // 2. Создай ProductRepo
    // 3. Добавь 3 продукта
    // 4. Выведи все in_stock
    // 5. Обнови цену одного
    // 6. "Удали" другой (soft delete)
    // 7. Выведи все in_stock снова — удалённый не должен попасть
}
```

**Ожидаемый вывод (примерный):**

```
created: id=1 (Клавиатура)
created: id=2 (Монитор)
created: id=3 (Мышь)

in stock:
  [1] Клавиатура — 3500.00 ₽
  [2] Монитор — 25000.00 ₽
  [3] Мышь — 1200.00 ₽

updated price: id=1 → 4000.00
soft deleted: id=3

in stock:
  [1] Клавиатура — 4000.00 ₽
  [2] Монитор — 25000.00 ₽
```

**Требования:**

- Вся логика БД — внутри методов `ProductRepo`, `main` не пишет SQL
- Структуры с тегами `db:"..."` — пусть `sqlx` маппит сам
- `GetByID` при отсутствии записи: `fmt.Errorf("product not found: %w", sql.ErrNoRows)` — чтобы вызывающий код мог проверить через `errors.Is`
- Мягкое удаление — `UPDATE`, не `DELETE`
- Попробуй использовать `NamedExec` хотя бы в одном методе — почувствуй разницу с позиционными `$1, $2`

