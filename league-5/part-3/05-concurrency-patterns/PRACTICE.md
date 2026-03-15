# Практика — Паттерны конкурентности

---

## Часть 1 — Pipeline

### Простое

Создай файл `pipeline_simple.go`.

Реализуй конвейер из трёх стадий для обработки чисел:

```go
package main

import "fmt"

// generate отправляет числа в канал и закрывает его
func generate(nums ...int) <-chan int {
    // твоя реализация
}

// double удваивает каждое число
func double(in <-chan int) <-chan int {
    // твоя реализация
}

// filterEven пропускает только чётные числа
func filterEven(in <-chan int) <-chan int {
    // твоя реализация
}

func main() {
    for val := range filterEven(double(generate(1, 2, 3, 4, 5, 6, 7, 8, 9, 10))) {
        fmt.Println(val)
    }
}
```

**Требования:**
- Каждая стадия — отдельная горутина
- Каналы закрываются когда стадия завершила работу
- Без `WaitGroup` — только каналы

**Ожидаемый вывод:**
```
4
8
12
16
20
```

---

### Сложное

Создай файл `pipeline_advanced.go`.

Реализуй конвейер обработки заказов интернет-магазина:

```go
package main

import (
    "fmt"
    "math/rand"
    "time"
)

type Order struct {
    ID       int
    UserID   int
    Amount   float64
    Status   string
}

// generateOrders создаёт поток заказов
func generateOrders(count int) <-chan Order {
    // генерирует count заказов со случайными Amount от 10 до 1000
    // твоя реализация
}

// validate проверяет заказ: Amount > 0 и Amount <= 500
// заказы не прошедшие проверку отбрасываются, статус валидных = "validated"
func validate(in <-chan Order) <-chan Order {
    // твоя реализация
}

// applyDiscount применяет скидку 10% для заказов Amount > 300
func applyDiscount(in <-chan Order) <-chan Order {
    // твоя реализация
}

// save имитирует сохранение в БД через time.Sleep(10ms)
// устанавливает статус = "saved"
func save(in <-chan Order) <-chan Order {
    // твоя реализация
}

func main() {
    pipeline := save(applyDiscount(validate(generateOrders(20))))

    total := 0.0
    count := 0
    for order := range pipeline {
        fmt.Printf("order #%d user=%d amount=%.2f status=%s\n",
            order.ID, order.UserID, order.Amount, order.Status)
        total += order.Amount
        count++
    }

    fmt.Printf("\nprocessed: %d orders, total: %.2f\n", count, total)
}
```

**Требования:**
- Каждая стадия — отдельная горутина, закрывает свой канал при завершении
- `validate` отбрасывает невалидные заказы не останавливая конвейер
- В конце вывести количество обработанных заказов и сумму

---

## Часть 2 — Fan-out

### Простое

Создай файл `fanout_simple.go`.

Реализуй fan-out: один канал с задачами раздаётся трём воркерам.

```go
package main

import (
    "fmt"
    "sync"
    "time"
)

// worker читает задачи из jobs, обрабатывает их и печатает результат
// id — номер воркера (1, 2, 3)
func worker(id int, jobs <-chan int, wg *sync.WaitGroup) {
    // для каждой задачи: имитируй работу через time.Sleep(100ms)
    // выведи: "worker %d обработал задачу %d"
    // твоя реализация
}

func main() {
    jobs := make(chan int, 10)
    var wg sync.WaitGroup

    // запусти 3 воркера
    // твоя реализация

    // отправь 9 задач
    for i := 1; i <= 9; i++ {
        jobs <- i
    }
    close(jobs)

    wg.Wait()
    fmt.Println("все задачи выполнены")
}
```

**Ожидаемый вывод** (порядок может отличаться):
```
worker 1 обработал задачу 1
worker 2 обработал задачу 2
worker 3 обработал задачу 3
...
все задачи выполнены
```

---

### Сложное

Создай файл `fanout_advanced.go`.

Реализуй fan-out для параллельного скачивания файлов с результатами:

```go
package main

import (
    "fmt"
    "math/rand"
    "sync"
    "time"
)

type DownloadTask struct {
    ID  int
    URL string
}

type DownloadResult struct {
    TaskID  int
    URL     string
    Size    int   // размер в байтах (случайный)
    Elapsed time.Duration
    Err     error
}

// download имитирует скачивание: sleep от 100 до 500ms
// с вероятностью 20% возвращает ошибку "connection timeout"
func download(task DownloadTask) DownloadResult {
    // твоя реализация
}

// fanOut запускает n воркеров, каждый читает из tasks и пишет в результаты
func fanOut(tasks <-chan DownloadTask, n int) <-chan DownloadResult {
    results := make(chan DownloadResult)
    var wg sync.WaitGroup

    // запусти n воркеров
    // каждый воркер читает из tasks и пишет результат в results
    // когда все воркеры завершат — закрой results
    // твоя реализация

    return results
}

func main() {
    tasks := make(chan DownloadTask, 20)
    urls := []string{
        "https://example.com/file1.zip",
        "https://example.com/file2.zip",
        "https://example.com/file3.zip",
    }

    // отправь 12 задач
    for i := 1; i <= 12; i++ {
        tasks <- DownloadTask{ID: i, URL: urls[i%len(urls)]}
    }
    close(tasks)

    results := fanOut(tasks, 4) // 4 параллельных скачивания

    success, failed := 0, 0
    for r := range results {
        if r.Err != nil {
            fmt.Printf("task %d failed: %v\n", r.TaskID, r.Err)
            failed++
        } else {
            fmt.Printf("task %d done: %d bytes in %v\n", r.TaskID, r.Size, r.Elapsed)
            success++
        }
    }

    fmt.Printf("\nитого: успешно=%d ошибок=%d\n", success, failed)
}
```

**Требования:**
- `fanOut` принимает количество воркеров как параметр
- Воркеры завершаются когда канал `tasks` закрыт
- Канал `results` закрывается когда все воркеры завершили работу

---

## Часть 3 — Fan-in

### Простое

Создай файл `fanin_simple.go`.

Реализуй fan-in: объедини три канала в один.

```go
package main

import (
    "fmt"
    "sync"
    "time"
)

// source имитирует источник данных — отправляет числа с задержкой
func source(name string, nums ...int) <-chan string {
    out := make(chan string)
    go func() {
        defer close(out)
        for _, n := range nums {
            time.Sleep(time.Duration(n*50) * time.Millisecond)
            out <- fmt.Sprintf("%s: %d", name, n)
        }
    }()
    return out
}

// fanIn объединяет несколько каналов в один
func fanIn(channels ...<-chan string) <-chan string {
    // запусти горутину для каждого входного канала
    // каждая горутина читает из своего канала и пишет в общий out
    // когда все горутины завершат — закрой out
    // используй WaitGroup
    // твоя реализация
}

func main() {
    merged := fanIn(
        source("A", 1, 3, 5),
        source("B", 2, 4, 6),
        source("C", 1, 2, 3),
    )

    for msg := range merged {
        fmt.Println(msg)
    }
}
```

**Ожидаемый вывод** (порядок определяется задержками):
```
A: 1
C: 1
B: 2
C: 2
A: 3
...
```

---

### Сложное

Создай файл `fanin_advanced.go`.

Реализуй систему агрегации метрик из нескольких источников:

```go
package main

import (
    "context"
    "fmt"
    "math/rand"
    "time"
)

type Metric struct {
    Source string
    Name   string
    Value  float64
    Time   time.Time
}

// collectMetrics имитирует сбор метрик с одного сервера
// каждые interval отправляет случайную метрику
// останавливается при отмене контекста
func collectMetrics(ctx context.Context, serverName string, interval time.Duration) <-chan Metric {
    // твоя реализация
}

// mergeMetrics объединяет метрики из всех серверов в один канал
func mergeMetrics(ctx context.Context, sources ...<-chan Metric) <-chan Metric {
    // твоя реализация
}

// aggregator читает метрики и считает среднее значение по каждому имени метрики
type Aggregator struct {
    // твои поля
}

func NewAggregator() *Aggregator {
    // твоя реализация
}

func (a *Aggregator) Add(m Metric) {
    // добавь метрику в агрегатор
    // твоя реализация
}

func (a *Aggregator) Report() {
    // выведи среднее значение для каждой метрики
    // формат: "  metric_name: avg=X.XX count=N"
    // твоя реализация
}

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
    defer cancel()

    servers := []string{"server-1", "server-2", "server-3"}
    sources := make([]<-chan Metric, len(servers))
    for i, name := range servers {
        sources[i] = collectMetrics(ctx, name, 200*time.Millisecond)
    }

    merged := mergeMetrics(ctx, sources...)
    agg := NewAggregator()

    for m := range merged {
        fmt.Printf("[%s] %s.%s = %.2f\n", m.Time.Format("15:04:05"), m.Source, m.Name, m.Value)
        agg.Add(m)
    }

    fmt.Println("\n--- итоговый отчёт ---")
    agg.Report()
}
```

**Требования:**
- `collectMetrics` останавливается при отмене контекста и закрывает канал
- `mergeMetrics` закрывает выходной канал когда все источники закончили
- Метрики: `cpu_usage`, `memory_usage`, `request_rate` со случайными значениями

---

## Часть 4 — Worker Pool

### Простое

Создай файл `workerpool_simple.go`.

Реализуй пул воркеров для вычисления факториала:

```go
package main

import (
    "fmt"
    "sync"
)

type Task struct {
    ID    int
    Input int
}

type Result struct {
    TaskID int
    Input  int
    Output int
}

func factorial(n int) int {
    // твоя реализация
}

func workerPool(numWorkers int, tasks <-chan Task, results chan<- Result, wg *sync.WaitGroup) {
    // запусти numWorkers горутин
    // каждая читает задачи из tasks, вычисляет factorial и пишет в results
    // твоя реализация
}

func main() {
    tasks := make(chan Task, 10)
    results := make(chan Result, 10)
    var wg sync.WaitGroup

    workerPool(3, tasks, results, &wg)

    // отправь задачи
    for i := 1; i <= 10; i++ {
        tasks <- Task{ID: i, Input: i}
    }
    close(tasks)

    // закрой results когда все воркеры завершат
    go func() {
        wg.Wait()
        close(results)
    }()

    for r := range results {
        fmt.Printf("task %d: %d! = %d\n", r.TaskID, r.Input, r.Output)
    }
}
```

**Ожидаемый вывод** (порядок может отличаться):
```
task 3: 3! = 6
task 1: 1! = 1
task 2: 2! = 2
...
```

---

### Сложное

Создай файл `workerpool_advanced.go`.

Реализуй пул воркеров с повторными попытками и ограничением по времени:

```go
package main

import (
    "context"
    "errors"
    "fmt"
    "math/rand"
    "sync"
    "time"
)

type Job struct {
    ID      int
    Payload string
    Retries int // сколько раз уже пытались
}

type JobResult struct {
    JobID   int
    Output  string
    Retries int
    Err     error
}

const maxRetries = 3

// process имитирует обработку задачи
// с вероятностью 40% возвращает ошибку "service unavailable"
// успешная обработка занимает от 50 до 200ms
func process(ctx context.Context, job Job) (string, error) {
    // твоя реализация
}

// Pool управляет воркерами, очередью задач и повторными попытками
type Pool struct {
    numWorkers int
    jobs       chan Job
    results    chan JobResult
    wg         sync.WaitGroup
}

func NewPool(numWorkers, queueSize int) *Pool {
    // твоя реализация
}

func (p *Pool) Start(ctx context.Context) {
    // запусти numWorkers горутин
    // каждый воркер:
    //   1. читает задачу из p.jobs
    //   2. вызывает process
    //   3. при ошибке и retries < maxRetries — кладёт задачу обратно в p.jobs с Retries+1
    //   4. иначе — пишет результат в p.results
    // твоя реализация
}

func (p *Pool) Submit(job Job) {
    p.jobs <- job
}

func (p *Pool) Results() <-chan JobResult {
    return p.results
}

func (p *Pool) Wait() {
    // дождись завершения всех воркеров и закрой results
    // твоя реализация
}

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    pool := NewPool(5, 50)
    pool.Start(ctx)

    // отправь 15 задач
    for i := 1; i <= 15; i++ {
        pool.Submit(Job{ID: i, Payload: fmt.Sprintf("data-%d", i)})
    }
    close(pool.jobs)

    go pool.Wait()

    success, failed := 0, 0
    for r := range pool.Results() {
        if r.Err != nil {
            fmt.Printf("job %d failed after %d retries: %v\n", r.JobID, r.Retries, r.Err)
            failed++
        } else {
            fmt.Printf("job %d done (retries=%d): %s\n", r.JobID, r.Retries, r.Output)
            success++
        }
    }

    fmt.Printf("\nитого: успешно=%d провалено=%d\n", success, failed)
}
```

**Требования:**
- При ошибке задача возвращается в очередь если `retries < maxRetries`
- После `maxRetries` неудач — результат с ошибкой идёт в `results`
- Воркеры завершаются когда `jobs` закрыт и очередь пуста
- `go run -race workerpool_advanced.go` — без предупреждений

