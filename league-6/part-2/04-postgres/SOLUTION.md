# Решение — PostgreSQL

---

## Задача 1 — CRUD на коленке

```go
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type Bookmark struct {
	ID        int    `db:"id"`
	URL       string `db:"url"`
	Title     string `db:"title"`
	CreatedAt string `db:"created_at"`
}

func AddBookmark(db *sqlx.DB, url, title string) (int, error) {
	var id int
	err := db.QueryRow(
		"INSERT INTO bookmarks (url, title) VALUES ($1, $2) RETURNING id",
		url, title,
	).Scan(&id)
	return id, err
}

func GetAll(db *sqlx.DB) ([]Bookmark, error) {
	var bookmarks []Bookmark
	err := db.Select(&bookmarks, "SELECT id, url, title, created_at FROM bookmarks ORDER BY id")
	return bookmarks, err
}

func UpdateTitle(db *sqlx.DB, id int, newTitle string) error {
	res, err := db.Exec("UPDATE bookmarks SET title = $1 WHERE id = $2", newTitle, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("bookmark not found: id=%d", id)
	}
	return nil
}

func Delete(db *sqlx.DB, id int) error {
	res, err := db.Exec("DELETE FROM bookmarks WHERE id = $1", id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("bookmark not found: id=%d", id)
	}
	return nil
}

func main() {
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	db, err := sqlx.Connect("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	id1, _ := AddBookmark(db, "https://go.dev/doc", "Go docs")
	id2, _ := AddBookmark(db, "https://postgresqltutorial.com", "Postgres tutorial")
	id3, _ := AddBookmark(db, "https://leetcode.com", "LeetCode")
	fmt.Printf("added: id=%d\nadded: id=%d\nadded: id=%d\n\n", id1, id2, id3)

	all, _ := GetAll(db)
	fmt.Println("all bookmarks:")
	for _, b := range all {
		fmt.Printf("  [%d] %s — %s\n", b.ID, b.Title, b.URL)
	}

	fmt.Println()
	UpdateTitle(db, id1, "Go official docs")
	fmt.Printf("updated id=%d title → \"Go official docs\"\n", id1)
	Delete(db, id2)
	fmt.Printf("deleted id=%d\n\n", id2)

	all, _ = GetAll(db)
	fmt.Println("all bookmarks:")
	for _, b := range all {
		fmt.Printf("  [%d] %s — %s\n", b.ID, b.Title, b.URL)
	}
}
```

### Что здесь важно

- `sqlx.Connect` = `sql.Open` + `Ping` в одном вызове. Если база недоступна — ошибка сразу, а не при первом запросе.
- `db.Select(&bookmarks, ...)` — заменяет `Query` + цикл `rows.Next()` + `rows.Scan` + `rows.Close` + `rows.Err()`. Одна строка вместо десяти.
- `QueryRow` + `Scan` для `RETURNING id` — тут `sqlx` работает так же, как стандартная библиотека. Не всё нужно менять.
- `db.Exec` для `UPDATE`/`DELETE` — возвращает `sql.Result`, из которого берём `RowsAffected()`.

---

## Задача 2 — Грязные руки

### 4 проблемы:

| # | Строка | Проблема |
|---|--------|----------|
| 1 | `db, _ := sqlx.Connect(...)` | Ошибка подключения проигнорирована — программа упадёт с nil pointer |
| 2 | `fmt.Sprintf("...WHERE name = '%s'", name)` | **SQL-инъекция** — ввод `' OR '1'='1` вернёт всю таблицу |
| 3 | `db.Select(&users, query)` | Ошибка запроса проигнорирована |
| 4 | Нет `defer db.Close()` | Соединение не закрывается при завершении |

### Исправленный код:

```go
package main

import (
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type User struct {
	ID    int    `db:"id"`
	Email string `db:"email"`
}

func main() {
	db, err := sqlx.Connect("postgres", "postgres://user:pass@localhost:5432/testdb?sslmode=disable")
	if err != nil {
		log.Fatal("connect:", err)
	}
	defer db.Close()

	var name string
	fmt.Print("enter username: ")
	fmt.Scan(&name)

	var users []User
	err = db.Select(&users, "SELECT id, email FROM users WHERE name = $1", name)
	if err != nil {
		log.Fatal("query:", err)
	}

	if len(users) == 0 {
		fmt.Println("no users found")
		return
	}

	for _, u := range users {
		fmt.Printf("id=%d email=%s\n", u.ID, u.Email)
	}
}
```

### Что здесь важно

- SQL-инъекция — `$1` вместо `fmt.Sprintf`. Это правило без исключений.
- `sqlx.Connect` уже делает `Ping` — если ошибку проигнорировать, `db` будет `nil` и всё упадёт.
- `db.Select` при пустом результате возвращает пустой слайс и `nil` ошибку — это не ошибка, а нормальная ситуация. Но проверить стоит для UX.

---

## Задача 3 — Мини-репозиторий

```go
package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

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

func (r *ProductRepo) Create(p Product) (int, error) {
	var id int
	err := r.db.QueryRow(
		"INSERT INTO products (name, price, in_stock) VALUES ($1, $2, $3) RETURNING id",
		p.Name, p.Price, p.InStock,
	).Scan(&id)
	return id, err
}

func (r *ProductRepo) GetByID(id int) (Product, error) {
	var p Product
	err := r.db.Get(&p, "SELECT * FROM products WHERE id = $1", id)
	if errors.Is(err, sql.ErrNoRows) {
		return Product{}, fmt.Errorf("product not found: %w", sql.ErrNoRows)
	}
	return p, err
}

func (r *ProductRepo) ListInStock() ([]Product, error) {
	var products []Product
	err := r.db.Select(&products, "SELECT * FROM products WHERE in_stock = true ORDER BY id")
	return products, err
}

func (r *ProductRepo) UpdatePrice(id int, price float64) error {
	res, err := r.db.NamedExec(
		"UPDATE products SET price = :price WHERE id = :id",
		map[string]interface{}{"id": id, "price": price},
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("product not found: id=%d", id)
	}
	return nil
}

func (r *ProductRepo) SoftDelete(id int) error {
	res, err := r.db.Exec("UPDATE products SET in_stock = false WHERE id = $1", id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("product not found: id=%d", id)
	}
	return nil
}

func main() {
	db, err := sqlx.Connect("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	repo := NewProductRepo(db)

	id1, _ := repo.Create(Product{Name: "Клавиатура", Price: 3500.00, InStock: true})
	id2, _ := repo.Create(Product{Name: "Монитор", Price: 25000.00, InStock: true})
	id3, _ := repo.Create(Product{Name: "Мышь", Price: 1200.00, InStock: true})
	fmt.Printf("created: id=%d (Клавиатура)\n", id1)
	fmt.Printf("created: id=%d (Монитор)\n", id2)
	fmt.Printf("created: id=%d (Мышь)\n\n", id3)

	products, _ := repo.ListInStock()
	fmt.Println("in stock:")
	for _, p := range products {
		fmt.Printf("  [%d] %s — %.2f ₽\n", p.ID, p.Name, p.Price)
	}

	fmt.Println()
	repo.UpdatePrice(id1, 4000.00)
	fmt.Printf("updated price: id=%d → 4000.00\n", id1)
	repo.SoftDelete(id3)
	fmt.Printf("soft deleted: id=%d\n\n", id3)

	products, _ = repo.ListInStock()
	fmt.Println("in stock:")
	for _, p := range products {
		fmt.Printf("  [%d] %s — %.2f ₽\n", p.ID, p.Name, p.Price)
	}
}
```

### Что здесь важно

- `db.Get(&p, ...)` — одна строка вместо `QueryRow` + `Scan` с перечислением полей. Маппинг через теги `db:"..."`.
- `db.Select(&products, ...)` — слайс за один вызов. Нет `rows.Next()`, `rows.Close()`, `rows.Err()`.
- `NamedExec` в `UpdatePrice` — `:price`, `:id` вместо `$1`, `$2`. Читабельнее, особенно когда параметров много. Принимает `map` или структуру.
- `sql.ErrNoRows` — `sqlx.Get` возвращает его так же, как стандартный `QueryRow.Scan`. Это не ошибка базы, а штатная ситуация «не найдено».
- **Паттерн Repository** — `main` не знает про SQL. Завтра заменишь PostgreSQL на SQLite — поменяется только реализация `ProductRepo`, а `main` останется тем же.
