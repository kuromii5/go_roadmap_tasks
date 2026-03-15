# Solution — Конкурентность в Go

---

## Часть 1 — Теория

```markdown
1. Горутина управляется планировщиком Go, а не ОС. Стартовый стек ~2КБ против
   ~1МБ у потока ОС. Переключение горутин в user space — без системного вызова.

2. i++ — три шага: прочитать значение из памяти → прибавить 1 → записать обратно.
   Между шагами планировщик может переключить горутину, и другая горутина
   прочитает устаревшее значение.

3. Mutex блокирует всех — и читателей и писателей. RWMutex позволяет читать
   параллельно, блокирует только при записи. RWMutex выгоден когда чтений
   намного больше чем записей.

4. Атомик — уровень железа, одна неделимая инструкция CPU с поддержкой
   протоколов когерентности кэша. Мьютекс — уровень ОС, паркует горутину
   и будит после освобождения. Атомик быстрее, но только для одной переменной.

5. False sharing: две горутины работают с разными переменными, но они лежат
   в одной кэш-линии (64 байта). Ядра синхронизируют кэш-линию целиком,
   хотя данные разные. Паддинг разносит переменные в разные кэш-линии.

6. Закон Амдала: максимальное ускорение = 1 / доля_последовательного_кода.
   При 50% последовательного кода — максимум ×2, сколько бы ядер ни добавить.

7. runtime.NumGoroutine() — метрика числа горутин. Если растёт без предела —
   утечка. В продакшне выводят в Prometheus/Grafana и настраивают алерт.

8. sync.Map оптимизирован для двух сценариев: ключи пишутся один раз и потом
   только читаются, или разные горутины работают с разными ключами (без
   пересечений). В этих случаях sync.Map избегает блокировки при чтении
   через внутренний read-only map. map + RWMutex выгоднее при частых записях
   по одним и тем же ключам — sync.Map тогда проигрывает из-за оверхеда
   на внутреннюю синхронизацию двух map.

9. Обычный map не потокобезопасен на уровне рантайма Go. Параллельная запись
   и чтение вызывают панику "concurrent map read and map write" — это не
   просто гонка данных, а намеренная защита от повреждения внутренней
   структуры хэш-таблицы во время эвакуации данных.

10. map + RWMutex выгоднее sync.Map когда: часто обновляются одни и те же ключи,
    нужны операции которых нет в sync.Map (len, итерация со сложной логикой),
    или нужна атомарность нескольких операций сразу.
```

---

## Часть 2 — Горутины и WaitGroup

```go
package main

import (
    "fmt"
    "sync"
)

func printNumber(n int, wg *sync.WaitGroup) {
    defer wg.Done()
    fmt.Printf("goroutine %d\n", n)
}

func main() {
    var wg sync.WaitGroup

    for i := 1; i <= 5; i++ {
        wg.Add(1)
        go printNumber(i, &wg)
    }

    wg.Wait()
}
```

---

## Часть 3 — Гонка данных и атомики

```go
package main

import (
    "fmt"
    "sync"
    "sync/atomic"
)

func withoutSync(n int) int {
    var counter int
    var wg sync.WaitGroup

    for i := 0; i < n; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            counter++ // гонка данных
        }()
    }

    wg.Wait()
    return counter
}

func withAtomic(n int) int {
    var counter atomic.Int64
    var wg sync.WaitGroup

    for i := 0; i < n; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            counter.Add(1)
        }()
    }

    wg.Wait()
    return int(counter.Load())
}

func main() {
    fmt.Println("without sync:", withoutSync(1000))  // меньше 1000
    fmt.Println("with atomic: ", withAtomic(1000))   // всегда 1000
}
```

---

## Часть 4 — Мьютекс: кошелёк

```go
package main

import (
    "fmt"
    "sync"
)

type Wallet struct {
    mu           sync.Mutex
    balance      int
    transactions int
}

func NewWallet() *Wallet {
    return &Wallet{}
}

func (w *Wallet) Deposit(amount int) {
    w.mu.Lock()
    defer w.mu.Unlock()
    w.balance += amount
    w.transactions++
}

func (w *Wallet) Info() (balance, transactions int) {
    w.mu.Lock()
    defer w.mu.Unlock()
    return w.balance, w.transactions
}

func main() {
    w := NewWallet()
    var wg sync.WaitGroup

    for i := 0; i < 500; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            w.Deposit(10)
        }()
    }

    wg.Wait()
    balance, tx := w.Info()
    fmt.Printf("Balance: %d\nTransactions: %d\n", balance, tx)
}
```

---

## Часть 5 — RWMutex: кэш

```go
package main

import (
    "fmt"
    "sync"
)

type Cache struct {
    mu   sync.RWMutex
    data map[string]string
}

func NewCache() *Cache {
    return &Cache{data: make(map[string]string)}
}

func (c *Cache) Set(key, value string) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.data[key] = value
}

func (c *Cache) Get(key string) (string, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    val, ok := c.data[key]
    return val, ok
}

func main() {
    c := NewCache()
    var wg sync.WaitGroup

    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func(i int) {
            defer wg.Done()
            c.Set(fmt.Sprintf("key%d", i), fmt.Sprintf("val%d", i))
        }(i)
    }

    for i := 0; i < 50; i++ {
        wg.Add(1)
        go func(i int) {
            defer wg.Done()
            c.Get(fmt.Sprintf("key%d", i%10))
        }(i)
    }

    wg.Wait()
    fmt.Println("done")
}
```

---

## Часть 6 — sync.Map

```go
package main

import (
    "fmt"
    "sync"
    "sync/atomic"
)

type PageCounter struct {
    m sync.Map // key: string → value: *atomic.Int64
}

func NewPageCounter() *PageCounter {
    return &PageCounter{}
}

func (p *PageCounter) Visit(page string) {
    // LoadOrStore атомарно возвращает существующий счётчик или сохраняет новый.
    // Несколько горутин могут одновременно создавать счётчик для одной страницы —
    // LoadOrStore гарантирует что только один из них будет сохранён.
    counter := &atomic.Int64{}
    actual, _ := p.m.LoadOrStore(page, counter)
    actual.(*atomic.Int64).Add(1)
}

func (p *PageCounter) Count(page string) int {
    val, ok := p.m.Load(page)
    if !ok {
        return 0
    }
    return int(val.(*atomic.Int64).Load())
}

func (p *PageCounter) Top() {
    p.m.Range(func(key, value any) bool {
        fmt.Printf("%s %d\n", key, value.(*atomic.Int64).Load())
        return true // вернуть false — остановить итерацию
    })
}

func main() {
    pc := NewPageCounter()
    var wg sync.WaitGroup

    pages := []string{"/home", "/about", "/contact", "/home", "/home"}

    for _, page := range pages {
        wg.Add(1)
        go func(p string) {
            defer wg.Done()
            pc.Visit(p)
        }(page)
    }

    wg.Wait()
    pc.Top()
    fmt.Println("/home visits:", pc.Count("/home"))
}
```

> `LoadOrStore` + `atomic.Int64` внутри — идиоматичный способ сделать конкурентный счётчик через `sync.Map`. Просто `Store` после `Load` создаёт гонку: между чтением и записью другая горутина успеет изменить значение.

---

## Часть 7 — Каналы: pipeline

```go
package main

import "fmt"

func generate(nums ...int) <-chan int {
    out := make(chan int)
    go func() {
        for _, n := range nums {
            out <- n
        }
        close(out)
    }()
    return out
}

func square(in <-chan int) <-chan int {
    out := make(chan int)
    go func() {
        for n := range in {
            out <- n * n
        }
        close(out)
    }()
    return out
}

func main() {
    for val := range square(generate(1, 2, 3, 4, 5)) {
        fmt.Println(val)
    }
}
```

---

## Часть 8 — Select и таймаут

```go
package main

import (
    "errors"
    "fmt"
    "math/rand"
    "time"
)

func slowOperation() string {
    time.Sleep(time.Duration(rand.Intn(4)+1) * time.Second)
    return "result"
}

func withTimeout(timeout time.Duration) (string, error) {
    // буфер 1 — горутина не зависнет если таймаут сработал раньше результата
    ch := make(chan string, 1)

    go func() {
        ch <- slowOperation()
    }()

    select {
    case result := <-ch:
        return result, nil
    case <-time.After(timeout):
        return "", errors.New("timeout exceeded")
    }
}

func main() {
    result, err := withTimeout(2 * time.Second)
    if err != nil {
        fmt.Println("error:", err)
    } else {
        fmt.Println("result:", result)
    }
}
```

> Буфер `make(chan string, 1)` важен: если таймаут сработал, горутина всё равно завершится и запишет в канал — без буфера она зависнет навсегда.

---

## Часть 9 — Утечка горутин

```go
package main

import (
    "context"
    "fmt"
    "time"
)

func leaky() <-chan int {
    ch := make(chan int)
    go func() {
        // УТЕЧКА: канал небуферизован. Если вызывающий код не читает из ch
        // (например, не сохранил возвращённый канал или вышел по таймауту),
        // горутина заблокируется на ch <- result навсегда.
        // GC не освободит горутину — она держит ссылку на ch.
        result := heavyWork()
        ch <- result
    }()
    return ch
}

func heavyWork() int {
    time.Sleep(2 * time.Second)
    return 42
}

func fixed(ctx context.Context) <-chan int {
    ch := make(chan int, 1) // буфер 1 — горутина запишет и завершится даже если никто не читает
    go func() {
        result := heavyWork()
        select {
        case ch <- result:
        case <-ctx.Done(): // контекст отменён — выходим, не блокируемся
        }
    }()
    return ch
}

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
    defer cancel()

    ch := fixed(ctx)
    select {
    case val := <-ch:
        fmt.Println("got:", val)
    case <-ctx.Done():
        fmt.Println("cancelled")
    }
}
```

---

## Итоговое задание — Биржа заявок

```go
// order.go
package main

type Side string

const (
    Buy  Side = "buy"
    Sell Side = "sell"
)

type Order struct {
    ID    int
    Side  Side
    Price int
    Qty   int
}

type Trade struct {
    BuyOrderID  int
    SellOrderID int
    Price       int
    Qty         int
}
```

```go
// generator.go
package main

import (
    "context"
    "math/rand"
    "time"
)

func generateOrders(ctx context.Context) <-chan Order {
    out := make(chan Order)
    go func() {
        defer close(out)
        id := 0
        for {
            select {
            case <-ctx.Done():
                return
            case <-time.After(100 * time.Millisecond):
                id++
                side := Buy
                if rand.Intn(2) == 0 {
                    side = Sell
                }
                out <- Order{
                    ID:    id,
                    Side:  side,
                    Price: 90 + rand.Intn(21),
                    Qty:   1 + rand.Intn(10),
                }
            }
        }
    }()
    return out
}
```

```go
// matcher.go
package main

import (
    "context"
    "sync"
)

type OrderBook struct {
    mu    sync.Mutex
    buys  []Order
    sells []Order
}

func (ob *OrderBook) match(order Order) (Trade, bool) {
    ob.mu.Lock()
    defer ob.mu.Unlock()

    if order.Side == Buy {
        for i, sell := range ob.sells {
            if order.Price >= sell.Price {
                ob.sells = append(ob.sells[:i], ob.sells[i+1:]...)
                return Trade{
                    BuyOrderID:  order.ID,
                    SellOrderID: sell.ID,
                    Price:       sell.Price,
                    Qty:         min(order.Qty, sell.Qty),
                }, true
            }
        }
        ob.buys = append(ob.buys, order)
    } else {
        for i, buy := range ob.buys {
            if order.Price <= buy.Price {
                ob.buys = append(ob.buys[:i], ob.buys[i+1:]...)
                return Trade{
                    BuyOrderID:  buy.ID,
                    SellOrderID: order.ID,
                    Price:       buy.Price,
                    Qty:         min(order.Qty, buy.Qty),
                }, true
            }
        }
        ob.sells = append(ob.sells, order)
    }
    return Trade{}, false
}

func matcher(ctx context.Context, orders <-chan Order) <-chan Trade {
    out := make(chan Trade)
    ob := &OrderBook{}
    go func() {
        defer close(out)
        for order := range orders {
            if trade, ok := ob.match(order); ok {
                select {
                case out <- trade:
                case <-ctx.Done():
                    return
                }
            }
        }
    }()
    return out
}

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}
```

```go
// stats.go
package main

import (
    "fmt"
    "sync/atomic"
    "time"
)

type Stats struct {
    trades   atomic.Int64
    volume   atomic.Int64
    priceSum atomic.Int64
}

func runStats(trades <-chan Trade) {
    var s Stats
    ticker := time.NewTicker(2 * time.Second)
    defer ticker.Stop()

    print := func() {
        t := s.trades.Load()
        v := s.volume.Load()
        p := s.priceSum.Load()
        avg := 0.0
        if t > 0 {
            avg = float64(p) / float64(t)
        }
        fmt.Printf("[stats] trades: %d | volume: %d | avg price: %.1f\n", t, v, avg)
    }

    for {
        select {
        case trade, ok := <-trades:
            if !ok {
                print() // итоговая статистика после закрытия канала
                return
            }
            s.trades.Add(1)
            s.volume.Add(int64(trade.Qty))
            s.priceSum.Add(int64(trade.Price))
        case <-ticker.C:
            print()
        }
    }
}
```

```go
// main.go
package main

import (
    "context"
    "fmt"
    "time"
)

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    orders := generateOrders(ctx)
    trades := matcher(ctx, orders)
    runStats(trades)

    fmt.Println("exchange stopped")
}
```