# Решение — NoSQL (основы)

---

## Задача 1 — Выбери базу

| # | Сценарий | Выбор | Тип | Почему |
|---|----------|-------|-----|--------|
| 1 | Сессии | NoSQL | Key-Value (Redis) | Токен → данные, быстрый доступ по ключу, TTL из коробки |
| 2 | Банковские переводы | SQL (PostgreSQL) | — | ACID-транзакции обязательны, consistency критична |
| 3 | Лента с комментариями | NoSQL | Document (MongoDB) | Пост + комментарии + реакции — естественный вложенный документ |
| 4 | Друзья друзей | NoSQL | Graph (Neo4j) | Обход связей — нативная операция графовой базы, JOIN-ы на SQL будут убийственны |
| 5 | Логи API | NoSQL | Column (ClickHouse) | Запись огромных объёмов, колоночное хранение для агрегаций |
| 6 | Каталог товаров | NoSQL | Document (MongoDB) | Разные категории — разные поля. Гибкая схема вместо 50 nullable-колонок |
| 7 | Корзина | NoSQL | Key-Value (Redis) | Временные данные, привязка к сессии, быстрый доступ |
| 8 | Аналитика кликов | NoSQL | Column (ClickHouse) | Агрегации по большим объёмам — колоночные базы на порядки быстрее |

### Нюансы

- Пункт 3 — если нужна сложная аналитика по комментариям отдельно (топ авторов комментариев, поиск по тексту), SQL может быть удобнее. Зависит от приоритетов.
- Пункт 6 — SQL тоже умеет (`JSONB` в PostgreSQL), но если большинство запросов — чтение карточки товара целиком, document store натуральнее.

---

## Задача 2 — SQL vs Document

### SQL-схема

```sql
CREATE TABLE authors (
    id SERIAL PRIMARY KEY,
    nickname VARCHAR(100) NOT NULL
);

CREATE TABLE posts (
    id SERIAL PRIMARY KEY,
    author_id INT REFERENCES authors(id),
    title VARCHAR(300) NOT NULL,
    body TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE comments (
    id SERIAL PRIMARY KEY,
    post_id INT REFERENCES posts(id),
    author_id INT REFERENCES authors(id),
    body TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE tags (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) UNIQUE NOT NULL
);

CREATE TABLE post_tags (
    post_id INT REFERENCES posts(id),
    tag_id INT REFERENCES tags(id),
    PRIMARY KEY (post_id, tag_id)
);
```

### Document-схема

```json
{
  "_id": "post_1",
  "title": "Middleware в Go",
  "body": "...",
  "author": {
    "id": "author_1",
    "nickname": "alex"
  },
  "tags": ["go", "backend", "middleware"],
  "comments": [
    {
      "author": {"id": "author_2", "nickname": "ivan"},
      "body": "Круто!",
      "created_at": "2025-03-01T10:00:00Z"
    }
  ],
  "created_at": "2025-03-01T09:00:00Z"
}
```

### Ответы

| # | Вопрос | Ответ |
|---|--------|-------|
| 1 | Смена никнейма | **SQL проще** — один `UPDATE authors SET nickname = ...`. В document-store никнейм дублируется в каждом посте и комментарии — нужно обновлять везде (денормализация) |
| 2 | Пост + комментарии | **Document проще** — один запрос, всё в одном документе. SQL — `JOIN` двух таблиц |
| 3 | Все посты по тегу | **SQL эффективнее** — `JOIN post_tags` по индексу. В document — нужен индекс на массив `tags`, менее эффективно при большом объёме |
| 4 | Новое поле | **Document проще** — просто начинаешь писать поле, старые документы без него продолжают работать. SQL — `ALTER TABLE`, миграция |
| 5 | Гарантия записи | **SQL надёжнее** — транзакции из коробки. В document-store при параллельной записи в один документ возможны конфликты |

---

## Задача 3 — Eventual vs Strong Consistency

| # | Ситуация | Ответ | Почему |
|---|----------|-------|--------|
| 1 | Лайки YouTube | Eventual | Показать 1.4M вместо 1.4M+1 на секунду — никто не заметит |
| 2 | Баланс банка | Strong | Два списания одновременно → уход в минус. Недопустимо |
| 3 | Лента Telegram | Eventual | Сообщение появится у подписчиков с задержкой в секунду — ок |
| 4 | Последний товар | Strong | Два покупателя «купят» один и тот же последний товар → oversell |
| 5 | Просмотры статьи | Eventual | Аналогично лайкам — неточность в ±10 просмотров не критична |
| 6 | Место в кино | Strong | Два человека не могут сесть на одно кресло. Нужна гарантия |
| 7 | Статус заказа | Eventual | Статус «готовится» отобразится с задержкой в пару секунд — допустимо |

### Ключевой принцип

Вопрос всегда один: **что произойдёт, если два пользователя на секунду увидят устаревшие данные?** Если ответ «ничего страшного» — eventual. Если «потеря денег, двойная продажа, конфликт ресурсов» — strong.
