# Практика — Конкурентность в Go

---

## Часть 1 — Теория

Создай файл `theory.md` и ответь на вопросы:

1. Чем горутина отличается от потока ОС?
2. Почему операция `i++` не атомарна? Из каких трёх шагов она состоит?
3. В чём разница между `Mutex` и `RWMutex`? Когда использовать каждый?
4. Чем атомик отличается от мьютекса — на каком уровне работает каждый?
5. Что такое false sharing и почему паддинг помогает?
6. Что такое закон Амдала? Если 50% кода последовательны — какой максимальный прирост даст распараллеливание?
7. Как обнаружить утечку горутин в продакшне?
8. Чем `sync.Map` отличается от `map` с мьютексом? В каких сценариях `sync.Map` выгоднее?
9. Почему нельзя использовать обычный `map` из нескольких горутин без синхронизации?
10. В каком сценарии `map` + `RWMutex` выгоднее чем `sync.Map`?

---

## Часть 2 — Горутины и WaitGroup

Создай файл `goroutines.go`.

Запусти 5 горутин, каждая печатает своё число. Дождись завершения всех через `WaitGroup`.

```go
package main

import (
    "fmt"
    "sync"
)

func printNumber(n int, wg *sync.WaitGroup) {
    // твоя реализация
}

func main() {
    // запусти 5 горутин и дождись их завершения
}
```

**Требования:**
- Не использовать `time.Sleep` для синхронизации
- `WaitGroup` передавать по указателю

**Ожидаемый вывод** (порядок может отличаться):
```
goroutine 1
goroutine 2
goroutine 3
goroutine 4
goroutine 5
```

---

## Часть 3 — Гонка данных и атомики

Создай файл `race.go`.

```go
package main

import (
    "fmt"
    "sync"
    "sync/atomic"
)

func withoutSync(n int) int {
    // запусти n горутин, каждая прибавляет 1 к общему счётчику
    // без синхронизации — результат будет неверным
    // твоя реализация
}

func withAtomic(n int) int {
    // то же самое, но через atomic.Int64
    // твоя реализация
}

func main() {
    fmt.Println("without sync:", withoutSync(1000))  // непредсказуемо, меньше 1000
    fmt.Println("with atomic: ", withAtomic(1000))   // всегда 1000
}
```

**Требования:**
1. Реализуй `withoutSync` — убедись что результат неверный
2. Запусти с флагом `-race`, сохрани вывод в `theory.md`
3. Реализуй `withAtomic` через `atomic.Int64`

---

## Часть 4 — Мьютекс: кошелёк

Создай файл `wallet.go`.

```go
package main

import (
    "fmt"
    "sync"
)

type Wallet struct {
    // твои поля
}

func NewWallet() *Wallet {
    // твоя реализация
}

func (w *Wallet) Deposit(amount int) {
    // твоя реализация — защити оба поля одним мьютексом
}

func (w *Wallet) Info() (balance, transactions int) {
    // твоя реализация
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

**Требования:**
- `balance` и `transactions` должны меняться атомарно — защити одним `Mutex`
- `go run -race wallet.go` — без предупреждений

**Ожидаемый результат:**
```
Balance: 5000
Transactions: 500
```

---

## Часть 5 — RWMutex: кэш

Создай файл `cache.go`.

```go
package main

import (
    "fmt"
    "sync"
)

type Cache struct {
    // твои поля
}

func NewCache() *Cache {
    // твоя реализация
}

func (c *Cache) Set(key, value string) {
    // твоя реализация — используй Lock
}

func (c *Cache) Get(key string) (string, bool) {
    // твоя реализация — используй RLock
}

func main() {
    c := NewCache()
    var wg sync.WaitGroup

    // 10 писателей
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func(i int) {
            defer wg.Done()
            c.Set(fmt.Sprintf("key%d", i), fmt.Sprintf("val%d", i))
        }(i)
    }

    // 50 читателей
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

**Требования:**
- Чтение не должно блокировать другие чтения
- `go run -race cache.go` — без предупреждений

---

## Часть 6 — sync.Map

Создай файл `syncmap.go`.

Реализуй счётчик посещений страниц сайта. Несколько горутин одновременно пишут и читают данные — одни страницы пишутся часто, другие пишутся один раз и потом только читаются.

```go
package main

import (
    "fmt"
    "sync"
)

type PageCounter struct {
    // твои поля — используй sync.Map
}

func NewPageCounter() *PageCounter {
    // твоя реализация
}

func (p *PageCounter) Visit(page string) {
    // атомарно увеличь счётчик посещений страницы на 1
    // подсказка: используй Load + Store в цикле или atomic внутри sync.Map
    // твоя реализация
}

func (p *PageCounter) Count(page string) int {
    // верни количество посещений страницы
    // твоя реализация
}

func (p *PageCounter) Top() {
    // выведи все страницы и их счётчики через Range
    // твоя реализация
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

**Требования:**
- Использовать `sync.Map`, не `map` с мьютексом
- `go run -race syncmap.go` — без предупреждений
- `Visit` должен корректно работать при параллельных вызовах для одной страницы

**Ожидаемый вывод** (порядок строк может отличаться):
```
/about 1
/contact 1
/home 3
/home visits: 3
```

---

## Часть 7 — Каналы: pipeline

Создай файл `pipeline.go`.

```go
package main

import "fmt"

func generate(nums ...int) <-chan int {
    // отправляет числа в канал, закрывает его когда все отправлены
    // твоя реализация
}

func square(in <-chan int) <-chan int {
    // читает из in, возводит в квадрат, отправляет в новый канал
    // твоя реализация
}

func main() {
    for val := range square(generate(1, 2, 3, 4, 5)) {
        fmt.Println(val)
    }
}
```

**Требования:**
- Каждая стадия — отдельная горутина
- Каналы закрываются когда данные закончились
- Без `WaitGroup` и `Mutex` — только каналы

**Ожидаемый вывод:**
```
1
4
9
16
25
```

---

## Часть 8 — Select и таймаут

Создай файл `timeout.go`.

```go
package main

import (
    "errors"
    "fmt"
    "math/rand"
    "time"
)

func slowOperation() string {
    // имитирует работу от 1 до 4 секунд
    time.Sleep(time.Duration(rand.Intn(4)+1) * time.Second)
    return "result"
}

func withTimeout(timeout time.Duration) (string, error) {
    // запусти slowOperation в горутине
    // используй select: либо получи результат, либо таймаут
    // твоя реализация
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

**Требования:**
- Использовать `select` с `time.After`
- При таймауте вернуть ошибку `"timeout exceeded"`
- Проверь оба сценария — запусти несколько раз

---

## Часть 9 — Утечка горутин

Создай файл `leak.go`.

**Шаг 1.** Перед тобой функция с утечкой — разберись почему она течёт и добавь комментарий:

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
        // TODO: объясни в комментарии почему эта горутина течёт
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
    ch := make(chan int)
    go func() {
        // исправь утечку через ctx.Done()
        // твоя реализация
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

**Требования:**
- В `leaky` добавь комментарий с объяснением утечки
- В `fixed` исправь утечку через `ctx.Done()`
- Ответь в `theory.md`: как обнаружить утечку горутин в продакшне?

---

## Итоговое задание — Биржа заявок

Реализуй упрощённую биржу, которая принимает заявки на покупку и продажу и матчит их между собой.

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

import "context"

func generateOrders(ctx context.Context) <-chan Order {
    // каждые 100мс генерирует случайную заявку:
    // - сторона: случайно buy или sell
    // - цена: от 90 до 110
    // - количество: от 1 до 10
    // останавливается при отмене контекста и закрывает канал
    // твоя реализация
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
    // твои поля: очереди на покупку и продажу, мьютекс
}

func matcher(ctx context.Context, orders <-chan Order) <-chan Trade {
    // читает заявки из канала
    // если buy.Price >= минимальной цены продажи — создаёт Trade
    // если sell.Price <= максимальной цены покупки — создаёт Trade
    // иначе добавляет заявку в очередь
    // закрывает канал trades когда завершает работу
    // твоя реализация
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
    // твои поля: атомарные счётчики — количество сделок, объём, сумма цен
}

func runStats(trades <-chan Trade) {
    // читает сделки и обновляет счётчики через atomic
    // каждые 2 секунды выводит:
    // [stats] trades: 12 | volume: 87 | avg price: 99.3
    // завершается когда канал trades закрыт
    // твоя реализация
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

**Требования:**
- `go run -race .` — без предупреждений о гонке данных
- Генератор останавливается при отмене контекста и закрывает канал
- Матчер закрывает канал `trades` когда завершает работу
- Сборщик завершается когда канал `trades` закрыт
- Итоговая статистика выводится после остановки

**Итоговая структура:**
```
exchange/
├── main.go
├── order.go
├── generator.go
├── matcher.go
└── stats.go
```

---

## Итоговая проверка

К концу задания у тебя должны быть файлы:

```
concurrency-practice/
├── theory.md
├── goroutines.go
├── race.go
├── wallet.go
├── cache.go
├── syncmap.go
├── pipeline.go
├── timeout.go
├── leak.go
└── exchange/
    ├── main.go
    ├── order.go
    ├── generator.go
    ├── matcher.go
    └── stats.go
```

- Все файлы проходят `go run -race` без предупреждений
- В `theory.md` даны ответы на все 10 вопросов и вывод `-race` из части 3
- В `leak.go` есть комментарий с объяснением утечки