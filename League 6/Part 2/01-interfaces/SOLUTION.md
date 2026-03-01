# Решения

---

## Задача 1 — Masker

```go
package main

import (
	"fmt"
	"strings"
)

type Masker interface {
	Mask() string
}

type Email string
type Phone string
type CardNumber string

func (e Email) Mask() string {
	parts := strings.SplitN(string(e), "@", 2)
	if len(parts) != 2 {
		return string(e)
	}
	return string(parts[0][0]) + "***@" + parts[1]
}

func (p Phone) Mask() string {
	s := string(p)
	if len(s) < 5 {
		return s
	}
	return s[:2] + "***" + s[len(s)-4:]
}

func (c CardNumber) Mask() string {
	s := string(c)
	if len(s) < 4 {
		return s
	}
	return "****" + s[len(s)-4:]
}

func PrintMasked(m Masker) {
	fmt.Println(m.Mask())
}

func main() {
	PrintMasked(Email("alex@gmail.com"))
	PrintMasked(Phone("+79991234567"))
	PrintMasked(CardNumber("1234567812345678"))
}
```

### Что здесь важно

- Типы `Email`, `Phone`, `CardNumber` — это просто `string` с методами. Интерфейс реализуется неявно (implicit implementation) — нигде не написано `implements`.
- `PrintMasked` не знает о конкретных типах — принимает любой `Masker`.
- Каждый тип сам решает, как маскировать свои данные.

---

## Задача 2 — Дополни код

```go
type AgeInput struct {
	Value int
}

func (a AgeInput) Validate() bool {
	return a.Value > 0 && a.Value <= 130
}

func (a AgeInput) ErrorMessage() string {
	switch {
	case a.Value < 0:
		return "возраст не может быть отрицательным"
	case a.Value == 0:
		return "возраст не может быть нулевым"
	case a.Value > 130:
		return "возраст не может быть больше 130"
	default:
		return ""
	}
}
```

### Что здесь важно

- Задача «наоборот» — интерфейс и потребитель уже написаны, нужно реализовать контракт.
- `ErrorMessage()` вынужден содержать логику, а не просто возвращать константу — один метод, разные ответы в зависимости от состояния.
- `AgeInput` автоматически удовлетворяет `Validator`, потому что имеет оба метода с нужными сигнатурами.

---

## Задача 3 — Stringify

```go
package main

import "fmt"

type Color struct {
	R, G, B uint8
}

func (c Color) String() string {
	return fmt.Sprintf("rgb(%d, %d, %d)", c.R, c.G, c.B)
}

type Point struct {
	X, Y int
}

func Stringify(values []any) []string {
	result := make([]string, 0, len(values))

	for _, v := range values {
		if s, ok := v.(fmt.Stringer); ok {
			result = append(result, s.String())
		} else {
			result = append(result, "unknown")
		}
	}

	return result
}

func main() {
	values := []any{
		Color{255, 0, 128},
		42,
		Point{1, 2},
		Color{0, 0, 0},
		"hello",
	}

	result := Stringify(values)
	for _, s := range result {
		fmt.Println(s)
	}
}
```

### Что здесь важно

- `v.(fmt.Stringer)` — type assertion с проверкой через `ok`. Без `ok` паника при несовпадении.
- `string` в Go **не** реализует `fmt.Stringer` (у него нет метода `String()`), поэтому `"hello"` → `"unknown"`.
- `Stringify` полностью абстрагирована от конкретных типов — она работает только через интерфейс.
- `[]any` — это алиас для `[]interface{}`, появился в Go 1.18.
