# Решение — Миграции

---

## Задача 1 — Напиши миграции

### `migrations/001_create_users.sql`

```sql
-- +goose Up
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(200) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- +goose Down
DROP TABLE users;
```

### `migrations/002_create_notes.sql`

```sql
-- +goose Up
CREATE TABLE notes (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id),
    title VARCHAR(300) NOT NULL,
    body TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- +goose Down
DROP TABLE notes;
```

### `migrations/003_add_notes_is_pinned.sql`

```sql
-- +goose Up
ALTER TABLE notes ADD COLUMN is_pinned BOOLEAN DEFAULT false;

-- +goose Down
ALTER TABLE notes DROP COLUMN is_pinned;
```

### Что здесь важно

- Порядок `DOWN` обратный `UP`. `DROP TABLE notes` нельзя выполнить до `DROP TABLE users` — FK не даст.
- Миграция 3 откатывается безопасно — `DROP COLUMN` удалит данные в колонке, но остальные данные строки останутся.
- `-- +goose Up` и `-- +goose Down` — обязательные комментарии-маркеры. Без них goose не поймёт, где что.

---

## Задача 2 — Что пойдёт не так?

| Шаг | Что произойдёт |
|-----|----------------|
| 1. `goose up` | Все 3 миграции применены. Таблицы `users`, `notes` (с `is_pinned`) созданы |
| 2. `INSERT` | Запись добавлена. `is_pinned` = `false` (default) |
| 3. `goose down` | Откат миграции 3. Колонка `is_pinned` удалена. Запись в `notes` **осталась** — удалилась только колонка |
| 4. `goose down` | Откат миграции 2. `DROP TABLE notes` — таблица удалена **вместе с записью**. Данные потеряны |
| 5. `goose up` | Миграции 2 и 3 применены заново. Таблицы пустые — данных нет |
| 6. `SELECT` | Пустой результат |

### Ответы

1. `is_pinned` удалена, остальные данные в `notes` сохранились.
2. `DROP TABLE` — запись уничтожена навсегда.
3. Таблицы пустые. `CREATE TABLE` не восстанавливает данные.
4. **Миграции управляют схемой, не данными.** `DOWN` может дропнуть таблицу — и все данные в ней. В продакшене `down` используют крайне редко и с осторожностью.

---

## Задача 3 — Makefile

```makefile
include .env
export

MIGRATIONS_DIR = migrations
DB_DSN = postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable

.PHONY: migrate-up migrate-down migrate-status migrate-create

migrate-up:
	goose -dir $(MIGRATIONS_DIR) postgres "$(DB_DSN)" up

migrate-down:
	goose -dir $(MIGRATIONS_DIR) postgres "$(DB_DSN)" down

migrate-status:
	goose -dir $(MIGRATIONS_DIR) postgres "$(DB_DSN)" status

migrate-create:
	goose -dir $(MIGRATIONS_DIR) create $(NAME) sql
```

`.env`:

```env
DB_USER=postgres
DB_PASSWORD=secret
DB_HOST=localhost
DB_PORT=5432
DB_NAME=notes_db
```

### Что здесь важно

- `include .env` + `export` — все переменные из `.env` доступны в Make.
- `migrate-create` не требует подключения к базе — только создаёт файл.
- `.PHONY` — говорит Make, что это не файлы, а команды. Без этого Make может решить, что таргет «уже существует» если есть файл с таким именем.
