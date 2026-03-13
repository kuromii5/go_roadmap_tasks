# Solution — Паттерны конкурентности

---

## Часть 1 — Pipeline

### Простое

```go
package main

import "fmt"

func generate(nums ...int) <-chan int {
    out := make(chan int)
    go func() {
        defer close(out)
        for _, n := range nums {
            out <- n
        }
    }()
    return out
}

func double(in <-chan int) <-chan int {
    out := make(chan int)
    go func() {
        defer close(out)
        for n := range in {
            out <- n * 2
        }
    }()
    return out
}

func filterEven(in <-chan int) <-chan int {
    out := make(chan int)
    go func() {
        defer close(out)
        for n := range in {
            if n%2 == 0 {
                out <- n
            }
        }
    }()
    return out
}

func main() {
    for val := range filterEven(double(generate(1, 2, 3, 4, 5, 6, 7, 8, 9, 10))) {
        fmt.Println(val)
    }
}
```

---

### Сложное

```go
package main

import (
    "fmt"
    "math/rand"
    "time"
)

type Order struct {
    ID     int
    UserID int
    Amount float64
    Status string
}

func generateOrders(count int) <-chan Order {
    out := make(chan Order)
    go func() {
        defer close(out)
        for i := 1; i <= count; i++ {
            out <- Order{
                ID:     i,
                UserID: rand.Intn(100) + 1,
                Amount: float64(rand.Intn(990)+10),
            }
        }
    }()
    return out
}

func validate(in <-chan Order) <-chan Order {
    out := make(chan Order)
    go func() {
        defer close(out)
        for order := range in {
            if order.Amount > 0 && order.Amount <= 500 {
                order.Status = "validated"
                out <- order
            }
            // невалидные просто отбрасываем
        }
    }()
    return out
}

func applyDiscount(in <-chan Order) <-chan Order {
    out := make(chan Order)
    go func() {
        defer close(out)
        for order := range in {
            if order.Amount > 300 {
                order.Amount *= 0.9
            }
            out <- order
        }
    }()
    return out
}

func save(in <-chan Order) <-chan Order {
    out := make(chan Order)
    go func() {
        defer close(out)
        for order := range in {
            time.Sleep(10 * time.Millisecond) // имитация записи в БД
            order.Status = "saved"
            out <- order
        }
    }()
    return out
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

---

## Часть 2 — Fan-out

### Простое

```go
package main

import (
    "fmt"
    "sync"
    "time"
)

func worker(id int, jobs <-chan int, wg *sync.WaitGroup) {
    defer wg.Done()
    for job := range jobs {
        time.Sleep(100 * time.Millisecond)
        fmt.Printf("worker %d обработал задачу %d\n", id, job)
    }
}

func main() {
    jobs := make(chan int, 10)
    var wg sync.WaitGroup

    for id := 1; id <= 3; id++ {
        wg.Add(1)
        go worker(id, jobs, &wg)
    }

    for i := 1; i <= 9; i++ {
        jobs <- i
    }
    close(jobs)

    wg.Wait()
    fmt.Println("все задачи выполнены")
}
```

---

### Сложное

```go
package main

import (
    "errors"
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
    Size    int
    Elapsed time.Duration
    Err     error
}

func download(task DownloadTask) DownloadResult {
    start := time.Now()
    time.Sleep(time.Duration(rand.Intn(400)+100) * time.Millisecond)

    if rand.Float32() < 0.2 {
        return DownloadResult{
            TaskID:  task.ID,
            URL:     task.URL,
            Elapsed: time.Since(start),
            Err:     errors.New("connection timeout"),
        }
    }

    return DownloadResult{
        TaskID:  task.ID,
        URL:     task.URL,
        Size:    rand.Intn(1_000_000) + 1000,
        Elapsed: time.Since(start),
    }
}

func fanOut(tasks <-chan DownloadTask, n int) <-chan DownloadResult {
    results := make(chan DownloadResult)
    var wg sync.WaitGroup

    for i := 0; i < n; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for task := range tasks {
                results <- download(task)
            }
        }()
    }

    // закрываем results когда все воркеры завершили
    go func() {
        wg.Wait()
        close(results)
    }()

    return results
}

func main() {
    tasks := make(chan DownloadTask, 20)
    urls := []string{
        "https://example.com/file1.zip",
        "https://example.com/file2.zip",
        "https://example.com/file3.zip",
    }

    for i := 1; i <= 12; i++ {
        tasks <- DownloadTask{ID: i, URL: urls[i%len(urls)]}
    }
    close(tasks)

    results := fanOut(tasks, 4)

    success, failed := 0, 0
    for r := range results {
        if r.Err != nil {
            fmt.Printf("❌ task %d failed: %v\n", r.TaskID, r.Err)
            failed++
        } else {
            fmt.Printf("✅ task %d done: %d bytes in %v\n", r.TaskID, r.Size, r.Elapsed)
            success++
        }
    }

    fmt.Printf("\nитого: успешно=%d ошибок=%d\n", success, failed)
}
```

---

## Часть 3 — Fan-in

### Простое

```go
package main

import (
    "fmt"
    "sync"
    "time"
)

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

func fanIn(channels ...<-chan string) <-chan string {
    out := make(chan string)
    var wg sync.WaitGroup

    for _, ch := range channels {
        wg.Add(1)
        go func(c <-chan string) {
            defer wg.Done()
            for msg := range c {
                out <- msg
            }
        }(ch)
    }

    go func() {
        wg.Wait()
        close(out)
    }()

    return out
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

---

### Сложное

```go
package main

import (
    "context"
    "fmt"
    "math/rand"
    "sync"
    "time"
)

type Metric struct {
    Source string
    Name   string
    Value  float64
    Time   time.Time
}

func collectMetrics(ctx context.Context, serverName string, interval time.Duration) <-chan Metric {
    out := make(chan Metric)
    names := []string{"cpu_usage", "memory_usage", "request_rate"}
    go func() {
        defer close(out)
        for {
            select {
            case <-ctx.Done():
                return
            case <-time.After(interval):
                out <- Metric{
                    Source: serverName,
                    Name:   names[rand.Intn(len(names))],
                    Value:  rand.Float64() * 100,
                    Time:   time.Now(),
                }
            }
        }
    }()
    return out
}

func mergeMetrics(ctx context.Context, sources ...<-chan Metric) <-chan Metric {
    out := make(chan Metric)
    var wg sync.WaitGroup

    for _, src := range sources {
        wg.Add(1)
        go func(s <-chan Metric) {
            defer wg.Done()
            for m := range s {
                select {
                case out <- m:
                case <-ctx.Done():
                    return
                }
            }
        }(src)
    }

    go func() {
        wg.Wait()
        close(out)
    }()

    return out
}

type Aggregator struct {
    sums   map[string]float64
    counts map[string]int
}

func NewAggregator() *Aggregator {
    return &Aggregator{
        sums:   make(map[string]float64),
        counts: make(map[string]int),
    }
}

func (a *Aggregator) Add(m Metric) {
    a.sums[m.Name] += m.Value
    a.counts[m.Name]++
}

func (a *Aggregator) Report() {
    for name, sum := range a.sums {
        avg := sum / float64(a.counts[name])
        fmt.Printf("  %s: avg=%.2f count=%d\n", name, avg, a.counts[name])
    }
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

---

## Часть 4 — Worker Pool

### Простое

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
    if n <= 1 {
        return 1
    }
    return n * factorial(n-1)
}

func workerPool(numWorkers int, tasks <-chan Task, results chan<- Result, wg *sync.WaitGroup) {
    for i := 0; i < numWorkers; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for task := range tasks {
                results <- Result{
                    TaskID: task.ID,
                    Input:  task.Input,
                    Output: factorial(task.Input),
                }
            }
        }()
    }
}

func main() {
    tasks := make(chan Task, 10)
    results := make(chan Result, 10)
    var wg sync.WaitGroup

    workerPool(3, tasks, results, &wg)

    for i := 1; i <= 10; i++ {
        tasks <- Task{ID: i, Input: i}
    }
    close(tasks)

    go func() {
        wg.Wait()
        close(results)
    }()

    for r := range results {
        fmt.Printf("task %d: %d! = %d\n", r.TaskID, r.Input, r.Output)
    }
}
```

---

### Сложное

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
    Retries int
}

type JobResult struct {
    JobID   int
    Output  string
    Retries int
    Err     error
}

const maxRetries = 3

func process(ctx context.Context, job Job) (string, error) {
    select {
    case <-ctx.Done():
        return "", ctx.Err()
    case <-time.After(time.Duration(rand.Intn(150)+50) * time.Millisecond):
    }

    if rand.Float32() < 0.4 {
        return "", errors.New("service unavailable")
    }
    return fmt.Sprintf("processed(%s)", job.Payload), nil
}

type Pool struct {
    numWorkers int
    jobs       chan Job
    results    chan JobResult
    wg         sync.WaitGroup
}

func NewPool(numWorkers, queueSize int) *Pool {
    return &Pool{
        numWorkers: numWorkers,
        jobs:       make(chan Job, queueSize),
        results:    make(chan JobResult, queueSize),
    }
}

func (p *Pool) Start(ctx context.Context) {
    for i := 0; i < p.numWorkers; i++ {
        p.wg.Add(1)
        go func() {
            defer p.wg.Done()
            for job := range p.jobs {
                output, err := process(ctx, job)
                if err != nil && job.Retries < maxRetries {
                    // повторная попытка
                    job.Retries++
                    p.jobs <- job
                    continue
                }
                p.results <- JobResult{
                    JobID:   job.ID,
                    Output:  output,
                    Retries: job.Retries,
                    Err:     err,
                }
            }
        }()
    }
}

func (p *Pool) Submit(job Job) {
    p.jobs <- job
}

func (p *Pool) Results() <-chan JobResult {
    return p.results
}

func (p *Pool) Wait() {
    p.wg.Wait()
    close(p.results)
}

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    pool := NewPool(5, 50)
    pool.Start(ctx)

    for i := 1; i <= 15; i++ {
        pool.Submit(Job{ID: i, Payload: fmt.Sprintf("data-%d", i)})
    }
    close(pool.jobs)

    go pool.Wait()

    success, failed := 0, 0
    for r := range pool.Results() {
        if r.Err != nil {
            fmt.Printf("❌ job %d failed after %d retries: %v\n", r.JobID, r.Retries, r.Err)
            failed++
        } else {
            fmt.Printf("✅ job %d done (retries=%d): %s\n", r.JobID, r.Retries, r.Output)
            success++
        }
    }

    fmt.Printf("\nитого: успешно=%d провалено=%d\n", success, failed)
}
```
