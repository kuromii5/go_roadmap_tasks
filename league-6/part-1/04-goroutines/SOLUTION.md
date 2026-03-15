# Решение — Консольный магазин

---

## Шаг 1 — Переменные и вывод

```go
package main

import "fmt"

func main() {
	name := "Ноутбук"
	price := 80000.0
	quantity := 5
	inStock := true

	fmt.Println("Товар:", name)
	fmt.Printf("Цена: %.2f руб.\n", price)
	fmt.Printf("На складе: %d шт.\n", quantity)
	fmt.Println("В наличии:", inStock)
}
```

---

## Шаг 2 — Условия

```go
if price > 50000 {
	price = price * 0.9
	fmt.Println("Применена скидка 10%")
} else if price >= 10000 {
	price = price * 0.95
	fmt.Println("Применена скидка 5%")
} else {
	fmt.Println("Скидка не применяется")
}

fmt.Printf("Цена после скидки: %.2f руб.\n", price)
```

---

## Шаг 3 — Цикл

```go
names := []string{"Ноутбук", "Мышь", "Монитор"}
prices := []float64{80000, 1500, 30000}
quantities := []int{5, 3, 1}

total := 0.0
for i := 0; i < len(names); i++ {
	cost := prices[i] * float64(quantities[i])
	total += cost
	fmt.Printf("%d. %-12s — %8.2f руб. x %d шт. = %10.2f руб.\n",
		i+1, names[i], prices[i], quantities[i], cost)
}
fmt.Printf("---\nИтого на складе: %.2f руб.\n", total)
```

---

## Шаг 4 — Функции

```go
func applyDiscount(price float64, percent float64) float64 {
	return price * (1 - percent/100)
}

func totalCost(price float64, quantity int) float64 {
	return price * float64(quantity)
}

func printItem(number int, name string, price float64, quantity int) {
	cost := totalCost(price, quantity)
	fmt.Printf("%d. %-12s — %8.2f руб. x %d шт. = %10.2f руб.\n",
		number, name, price, quantity, cost)
}
```

---

## Шаг 5 — Указатели

```go
func increasePrice(price *float64, amount float64) {
	*price += amount
}

func main() {
	price := 80000.0
	fmt.Printf("Цена до: %.2f\n", price)
	increasePrice(&price, 5000)
	fmt.Printf("Цена после: %.2f\n", price)
}
```

Без указателя функция получила бы копию `price`. Изменение копии не затронуло бы оригинал — цена в `main` осталась бы прежней.

---

## Шаг 6 — Структура

```go
package main

import "fmt"

type Product struct {
	Name     string
	Price    float64
	Quantity int
}

func (p Product) Total() float64 {
	return p.Price * float64(p.Quantity)
}

func (p *Product) ApplyDiscount(percent float64) {
	p.Price = p.Price * (1 - percent/100)
}

func printItem(number int, p Product) {
	fmt.Printf("%d. %-12s — %8.2f руб. x %d шт. = %10.2f руб.\n",
		number, p.Name, p.Price, p.Quantity, p.Total())
}

func main() {
	products := []Product{
		{Name: "Ноутбук", Price: 80000, Quantity: 5},
		{Name: "Мышь", Price: 1500, Quantity: 3},
		{Name: "Монитор", Price: 30000, Quantity: 1},
	}

	products[0].ApplyDiscount(10)

	total := 0.0
	for i, p := range products {
		printItem(i+1, p)
		total += p.Total()
	}
	fmt.Printf("---\nИтого: %.2f руб.\n", total)
}
```

`Total()` — метод по значению: только читает поля, ничего не меняет.
`ApplyDiscount()` — метод по указателю: меняет поле `Price`, нужен доступ к оригиналу.

---

## Шаг 7 — defer

```go
func runShop() {
	defer fmt.Println("\nМагазин закрыт. Спасибо за работу!")

	products := []Product{ ... }

	total := 0.0
	for i, p := range products {
		printItem(i+1, p)
		total += p.Total()
	}
	fmt.Printf("---\nИтого: %.2f руб.\n", total)
}

func main() {
	runShop()
}
```

`defer` выполняется когда функция завершается — после всего остального кода в `runShop`. Сколько бы строк ни было в функции, `defer` всегда последний.

---

## Шаг 8 — Слайсы и мапы

```go
func analyze(products []Product) {
	totals := map[string]float64{}
	for _, p := range products {
		totals[p.Name] = p.Total()
	}

	mostExpensive := products[0]
	maxStock := products[0]
	for _, p := range products[1:] {
		if p.Price > mostExpensive.Price {
			mostExpensive = p
		}
		if p.Quantity > maxStock.Quantity {
			maxStock = p
		}
	}

	fmt.Println("\n--- Аналитика ---")
	for name, total := range totals {
		fmt.Printf("%s: %.2f руб.\n", name, total)
	}
	fmt.Printf("Самый дорогой: %s (%.2f руб.)\n", mostExpensive.Name, mostExpensive.Price)
	fmt.Printf("Больше всего на складе: %s (%d шт.)\n", maxStock.Name, maxStock.Quantity)
}
```

---

## Шаг 9 — Пакеты

`go.mod`:
```
module shop

go 1.21
```

`internal/product/product.go`:
```go
package product

import "fmt"

type Product struct {
	Name     string
	Price    float64
	Quantity int
}

func (p Product) Total() float64 {
	return p.Price * float64(p.Quantity)
}

func (p *Product) ApplyDiscount(percent float64) {
	p.Price = p.Price * (1 - percent/100)
}

func Print(number int, p Product) {
	fmt.Printf("%d. %-12s — %8.2f руб. x %d шт. = %10.2f руб.\n",
		number, p.Name, p.Price, p.Quantity, p.Total())
}

func Analyze(products []Product) {
	totals := map[string]float64{}
	for _, p := range products {
		totals[p.Name] = p.Total()
	}

	mostExpensive := products[0]
	maxStock := products[0]
	for _, p := range products[1:] {
		if p.Price > mostExpensive.Price {
			mostExpensive = p
		}
		if p.Quantity > maxStock.Quantity {
			maxStock = p
		}
	}

	fmt.Println("\n--- Аналитика ---")
	for name, total := range totals {
		fmt.Printf("%s: %.2f руб.\n", name, total)
	}
	fmt.Printf("Самый дорогой: %s\n", mostExpensive.Name)
	fmt.Printf("Больше всего на складе: %s\n", maxStock.Name)
}
```

`cmd/main.go`:
```go
package main

import "shop/internal/product"

func main() {
	products := []product.Product{
		{Name: "Ноутбук", Price: 80000, Quantity: 5},
		{Name: "Мышь", Price: 1500, Quantity: 3},
		{Name: "Монитор", Price: 30000, Quantity: 1},
	}

	for i, p := range products {
		product.Print(i+1, p)
	}
	product.Analyze(products)
}
```

---

## Шаг 10 — Консольный ввод

```go
package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Product struct {
	Name     string
	Price    float64
	Quantity int
}

func (p Product) Total() float64 {
	return p.Price * float64(p.Quantity)
}

func (p *Product) ApplyDiscount(percent float64) {
	p.Price = p.Price * (1 - percent/100)
}

func main() {
	products := []Product{}
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("Команды: list, add <название> <цена> <кол-во>, discount <номер> <процент>, restock <номер> <кол-во>, total, exit")

	for {
		fmt.Print("> ")
		scanner.Scan()
		parts := strings.Fields(scanner.Text())
		if len(parts) == 0 {
			continue
		}

		switch parts[0] {

		case "list":
			if len(products) == 0 {
				fmt.Println("Список пуст")
				continue
			}
			for i, p := range products {
				fmt.Printf("%d. %-12s — %.2f руб. x %d шт.\n", i+1, p.Name, p.Price, p.Quantity)
			}

		case "add":
			if len(parts) < 4 {
				fmt.Println("Использование: add <название> <цена> <кол-во>")
				continue
			}
			price, err1 := strconv.ParseFloat(parts[2], 64)
			qty, err2 := strconv.Atoi(parts[3])
			if err1 != nil || err2 != nil {
				fmt.Println("Неверный формат цены или количества")
				continue
			}
			products = append(products, Product{Name: parts[1], Price: price, Quantity: qty})
			fmt.Println("Товар добавлен")

		case "discount":
			if len(parts) < 3 {
				fmt.Println("Использование: discount <номер> <процент>")
				continue
			}
			n, err1 := strconv.Atoi(parts[1])
			percent, err2 := strconv.ParseFloat(parts[2], 64)
			if err1 != nil || err2 != nil || n < 1 || n > len(products) {
				fmt.Println("Неверный номер или процент")
				continue
			}
			products[n-1].ApplyDiscount(percent)
			fmt.Printf("Новая цена: %.2f руб.\n", products[n-1].Price)

		case "restock":
			if len(parts) < 3 {
				fmt.Println("Использование: restock <номер> <кол-во>")
				continue
			}
			n, err1 := strconv.Atoi(parts[1])
			qty, err2 := strconv.Atoi(parts[2])
			if err1 != nil || err2 != nil || n < 1 || n > len(products) {
				fmt.Println("Неверный номер или количество")
				continue
			}
			products[n-1].Quantity += qty
			fmt.Printf("Новый остаток: %d шт.\n", products[n-1].Quantity)

		case "total":
			sum := 0.0
			for _, p := range products {
				sum += p.Total()
			}
			fmt.Printf("Общая стоимость склада: %.2f руб.\n", sum)

		case "exit":
			fmt.Println("Магазин закрыт.")
			return

		default:
			fmt.Println("Неизвестная команда")
		}
	}
}
```