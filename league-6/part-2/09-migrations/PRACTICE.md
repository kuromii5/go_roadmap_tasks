# Практика — Миграции

> **Видео:** [Миграции](https://youtu.be/MNyNxloZR0k?t=22785)

---

## Прежде чем начать — goose

Для миграций в Go-проектах стандарт — **[pressly/goose](https://github.com/pressly/goose)**. Есть альтернативы (golang-migrate, atlas), но goose самый популярный в Go-экосистеме: простой CLI, SQL-миграции, поддержка Go-миграций для сложных случаев.

```bash
go install github.com/pressly/goose/v3/cmd/goose@latest
```

```bash
# Создать миграцию
goose -dir migrations create add_users_table sql

# Применить
goose -dir migrations postgres "postgres://user:pass@localhost:5432/mydb?sslmode=disable" up

# Откатить последнюю
goose -dir migrations postgres "..." down
```

**Документация:**

- **GitHub:** https://github.com/pressly/goose
- **CLI reference:** https://pressly.github.io/goose/

---

## Задача 1 — Напиши миграции

Ты проектируешь базу для сервиса заметок. Напиши **3 миграции** в формате goose SQL.

**Миграция 1 — `001_create_users.sql`:**

UP: таблица `users` (id, email unique, name, created_at)
DOWN: дропнуть таблицу

**Миграция 2 — `002_create_notes.sql`:**

UP: таблица `notes` (id, user_id FK, title, body, created_at, updated_at)
DOWN: дропнуть таблицу

**Миграция 3 — `003_add_notes_is_pinned.sql`:**

UP: добавить колонку `is_pinned BOOLEAN DEFAULT false` в `notes`
DOWN: удалить колонку

**Формат файла goose:**

```sql
-- +goose Up
CREATE TABLE ...;

-- +goose Down
DROP TABLE ...;
```

**Требования:**

- Каждая миграция — отдельный файл в папке `migrations/`
- `DOWN` должен корректно откатывать `UP`
- Подумай: можно ли безопасно откатить миграцию 3, если в колонке уже есть данные?

---

## Задача 2 — Что пойдёт не так?

Ниже — последовательность действий разработчика. Для каждого шага ответь: **что будет в базе** и **будет ли ошибка**.

Исходное состояние: пустая база, 3 миграции из задачи 1.

```
1. goose up          — применить все
2. INSERT INTO notes (user_id, title, body) VALUES (1, 'test', 'hello');
3. goose down        — откатить последнюю
4. goose down        — откатить ещё одну
5. goose up          — применить все заново
6. SELECT * FROM notes;
```

**Вопросы:**

1. После шага 3 — что произойдёт с колонкой `is_pinned`? Данные в `notes` сохранятся?
2. После шага 4 — что произойдёт с записью из шага 2?
3. После шага 5 — таблицы пустые или с данными?
4. Разработчик вставил данные между миграциями. Почему это опасно?

---

## Задача 3 — Makefile

Добавь в `Makefile` проекта таргеты для работы с миграциями. Переменные подключения бери из `.env`.

Нужные таргеты:

| Таргет | Что делает |
|--------|-----------|
| `migrate-up` | Применить все миграции |
| `migrate-down` | Откатить последнюю |
| `migrate-status` | Показать статус миграций |
| `migrate-create` | Создать новую миграцию (имя через `NAME=...`) |

Использование:

```bash
make migrate-up
make migrate-down
make migrate-status
make migrate-create NAME=add_index_on_email
```

