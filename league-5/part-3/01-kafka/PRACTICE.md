# Практика — Kafka: уведомления в социальной сети

Реализуй систему уведомлений для социальной сети через Kafka. Сервис постов публикует события, сервис уведомлений их потребляет и обрабатывает.

---

## Часть 2 — Продюсер: сервис постов

Создай файл `producer/main.go`.

```go
package main

import (
    "encoding/json"
    "fmt"
    "math/rand"
    "time"

    "github.com/segmentio/kafka-go"
)

type PostEvent struct {
    EventID   string    `json:"event_id"`   // уникальный ID события — для идемпотентности
    EventType string    `json:"event_type"` // "post_created", "post_liked", "post_commented"
    PostID    int       `json:"post_id"`
    UserID    int       `json:"user_id"`
    CreatedAt time.Time `json:"created_at"`
}

func newWriter(topic string) *kafka.Writer {
    // создай Writer для топика "post-events"
    // адрес брокера: localhost:9092
    // твоя реализация
}

func publishEvent(writer *kafka.Writer, event PostEvent) error {
    // сериализуй event в JSON
    // опубликуй в Kafka используя EventID как ключ сообщения
    // ключ гарантирует что события одного поста попадут в одну партицию
    // твоя реализация
}

func main() {
    writer := newWriter("post-events")
    defer writer.Close()

    eventTypes := []string{"post_created", "post_liked", "post_commented"}

    for i := 0; i < 20; i++ {
        event := PostEvent{
            EventID:   fmt.Sprintf("evt-%d", i),
            EventType: eventTypes[rand.Intn(len(eventTypes))],
            PostID:    rand.Intn(5) + 1, // посты 1-5
            UserID:    rand.Intn(10) + 1,
            CreatedAt: time.Now(),
        }

        if err := publishEvent(writer, event); err != nil {
            fmt.Println("publish error:", err)
            continue
        }

        fmt.Printf("published: %s post_id=%d user_id=%d\n",
            event.EventType, event.PostID, event.UserID)
        time.Sleep(300 * time.Millisecond)
    }
}
```

**Требования:**
- Использовать `github.com/segmentio/kafka-go`
- Ключ сообщения = `EventID` — для идемпотентности и порядка
- При ошибке публикации — логировать и продолжать

---

## Часть 3 — Консьюмер: сервис уведомлений

Создай файл `consumer/main.go`.

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/segmentio/kafka-go"
)

type PostEvent struct {
    EventID   string    `json:"event_id"`
    EventType string    `json:"event_type"`
    PostID    int       `json:"post_id"`
    UserID    int       `json:"user_id"`
    CreatedAt time.Time `json:"created_at"`
}

// ProcessedEvents хранит уже обработанные EventID — для идемпотентности
type ProcessedEvents struct {
    // твои поля
}

func (p *ProcessedEvents) IsProcessed(eventID string) bool {
    // проверь был ли eventID уже обработан
    // твоя реализация
}

func (p *ProcessedEvents) MarkProcessed(eventID string) {
    // отметь eventID как обработанный
    // твоя реализация
}

func newReader(topic, groupID string) *kafka.Reader {
    // создай Reader для топика "post-events"
    // groupID = "notification-service"
    // адрес брокера: localhost:9092
    // твоя реализация
}

func handleEvent(event PostEvent) {
    // обработай событие в зависимости от типа:
    // "post_created"   → "User %d создал пост %d"
    // "post_liked"     → "User %d лайкнул пост %d"
    // "post_commented" → "User %d прокомментировал пост %d"
    // твоя реализация
}

func main() {
    reader := newReader("post-events", "notification-service")
    defer reader.Close()

    processed := &ProcessedEvents{/* инициализация */}
    ctx := context.Background()

    fmt.Println("notification service started, waiting for events...")

    for {
        msg, err := reader.ReadMessage(ctx)
        if err != nil {
            fmt.Println("read error:", err)
            break
        }

        var event PostEvent
        if err := json.Unmarshal(msg.Value, &event); err != nil {
            fmt.Println("unmarshal error:", err)
            continue
        }

        // идемпотентность: пропускаем дубли
        if processed.IsProcessed(event.EventID) {
            fmt.Printf("duplicate skipped: %s\n", event.EventID)
            continue
        }

        handleEvent(event)
        processed.MarkProcessed(event.EventID)
    }
}
```

**Требования:**
- Использовать consumer group `"notification-service"`
- Реализовать фильтр дублей через `ProcessedEvents`
- При неизвестном типе события — логировать и продолжать

---

## Часть 4 — Запуск через Docker Compose

Создай файл `docker-compose.yml`:

```yaml
version: '3'
services:
  kafka:
    image: confluentinc/cp-kafka:7.5.0
    ports:
      - "9092:9092"
    environment:
      KAFKA_NODE_ID: 1
      KAFKA_PROCESS_ROLES: broker,controller
      KAFKA_LISTENERS: PLAINTEXT://0.0.0.0:9092,CONTROLLER://0.0.0.0:9093
      KAFKA_ADVERTISED_LISTENERS: PLAINTEXT://localhost:9092
      KAFKA_CONTROLLER_QUORUM_VOTERS: 1@kafka:9093
      KAFKA_CONTROLLER_LISTENER_NAMES: CONTROLLER
      KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: 1
      CLUSTER_ID: "MkU3OEVBNTcwNTJENDM2Qk"
```

---

## Часть 5 — Создание топика и проверка

```bash
# запусти Kafka
docker compose up -d

# создай топик с 3 партициями
docker exec -it <container_id> kafka-topics \
  --create \
  --topic post-events \
  --partitions 3 \
  --replication-factor 1 \
  --bootstrap-server localhost:9092

# проверь топик
docker exec -it <container_id> kafka-topics \
  --describe \
  --topic post-events \
  --bootstrap-server localhost:9092

# запусти консьюмер в одном терминале
go run consumer/main.go

# запусти продюсер в другом терминале
go run producer/main.go
```

---

## Итоговая структура

```
kafka-practice/
├── docker-compose.yml
├── producer/
│   ├── go.mod
│   └── main.go
└── consumer/
    ├── go.mod
    └── main.go
```

---

## Итоговая проверка

- Консьюмер получает и обрабатывает все 20 событий
- При повторном запуске продюсера с теми же `EventID` — консьюмер пропускает дубли
- В выводе консьюмера видны все три типа событий с правильными эмодзи
- Если убить и перезапустить консьюмер — он продолжает с того места где остановился (благодаря consumer group)
