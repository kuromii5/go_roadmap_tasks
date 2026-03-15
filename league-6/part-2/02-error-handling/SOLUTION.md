# Решения

---

## Задача 1 — Цепочка ошибок

```go
package main

import (
	"errors"
	"fmt"
	"strings"
)

var ErrEmptyPath = errors.New("empty path")

func readFile(path string) (string, error) {
	if path == "" {
		return "", ErrEmptyPath
	}
	// Имитация чтения файла
	return "host=localhost;port=8080;broken", nil
}

func parseConfig(data string) (map[string]string, error) {
	result := make(map[string]string)

	entries := strings.Split(data, ";")
	for _, entry := range entries {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid entry: %s", entry)
		}
		result[parts[0]] = parts[1]
	}

	return result, nil
}

func LoadConfig(path string) (map[string]string, error) {
	data, err := readFile(path)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	cfg, err := parseConfig(data)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	return cfg, nil
}

func main() {
	// Кейс 1: пустой путь
	_, err := LoadConfig("")
	if err != nil {
		fmt.Println("errors.Is ErrEmptyPath:", errors.Is(err, ErrEmptyPath))
	}

	// Кейс 2: битый конфиг
	_, err = LoadConfig("valid")
	if err != nil {
		fmt.Println("full chain:", err)
	}
}
```

**Вывод:**

```
errors.Is ErrEmptyPath: true
full chain: load config: invalid entry: broken
```

### Что здесь важно

- `%w` создаёт цепочку — `errors.Is` умеет «разворачивать» обёртки и находить корневую ошибку на любой глубине.
- Если бы использовали `%v` вместо `%w` — ошибка превратилась бы в обычную строку, и `errors.Is` вернул бы `false`.
- Каждый слой добавляет свой контекст (`"load config: parse: ..."`) — при дебаге сразу видно, где именно сломалось.

---

## Задача 2 — Найди и исправь

### 5 ошибок:

| # | Где | Проблема |
|---|-----|----------|
| 1 | `parseAge` | Ошибка от `strconv.Atoi` игнорируется через `_` |
| 2 | `parseAge` | Функция возвращает `int`, не давая вызывающему коду узнать об ошибке |
| 3 | `validateAge` | `panic` вместо возврата `error` — невалидный возраст это не крах программы |
| 4 | `processUser` | `%v` вместо `%w` — ошибка теряет цепочку, `errors.Is` не сработает |
| 5 | `processUser` → `parseAge` | Результат `parseAge("not_a_number")` — тихий `0`, и `validateAge(0)` не ругается, хотя ввод невалиден |

### Исправленный код:

```go
package main

import (
	"fmt"
	"strconv"
)

func parseAge(input string) (int, error) {
	age, err := strconv.Atoi(input)
	if err != nil {
		return 0, fmt.Errorf("parse age: %w", err)
	}
	return age, nil
}

func validateAge(age int) error {
	if age < 0 {
		return fmt.Errorf("invalid age: %d", age)
	}
	if age > 150 {
		return fmt.Errorf("invalid age: %d (max 150)", age)
	}
	return nil
}

func processUser(name string, ageStr string) error {
	if name == "" {
		return fmt.Errorf("empty name")
	}

	age, err := parseAge(ageStr)
	if err != nil {
		return fmt.Errorf("process user: %w", err)
	}

	err = validateAge(age)
	if err != nil {
		return fmt.Errorf("process user: %w", err)
	}

	fmt.Printf("user %s, age %d\n", name, age)
	return nil
}

func main() {
	err := processUser("Alex", "not_a_number")
	if err != nil {
		fmt.Println(err)
	}

	err = processUser("", "25")
	if err != nil {
		fmt.Println(err)
	}

	err = processUser("Bob", "-5")
	if err != nil {
		fmt.Println(err)
	}
}
```

**Вывод:**

```
process user: parse age: strconv.Atoi: parsing "not_a_number": invalid syntax
process user: empty name
process user: invalid age: -5
```

### Что здесь важно

- Игнорирование ошибки через `_` — самый частый антипаттерн. Если функция возвращает `error`, его **надо** обработать.
- `panic` — для ситуаций, когда программа **не может** продолжать работу (нарушен инвариант, баг в логике). Невалидный пользовательский ввод — это **нормальная** ситуация, не паника.
- `%v` превращает ошибку в строку навсегда. `%w` сохраняет цепочку.

---

## Задача 3 — Кастомный тип ошибки

```go
package main

import (
	"errors"
	"fmt"
)

type InsufficientFundsError struct {
	AccountID string
	Balance   float64
	Amount    float64
}

func (e *InsufficientFundsError) Error() string {
	return fmt.Sprintf(
		"account %s: insufficient funds: balance %.2f, requested %.2f",
		e.AccountID, e.Balance, e.Amount,
	)
}

func Transfer(from, to string, amount float64, balances map[string]float64) error {
	balance, exists := balances[from]
	if !exists {
		return fmt.Errorf("account not found: %s", from)
	}

	if balance < amount {
		return fmt.Errorf("transfer: %w", &InsufficientFundsError{
			AccountID: from,
			Balance:   balance,
			Amount:    amount,
		})
	}

	balances[from] -= amount
	balances[to] += amount
	return nil
}

func main() {
	balances := map[string]float64{
		"alice": 100.0,
		"bob":   50.0,
	}

	// Перевод 1: не хватает средств
	err := Transfer("bob", "alice", 75.0, balances)
	if err != nil {
		fmt.Println(err)

		var fundsErr *InsufficientFundsError
		if errors.As(err, &fundsErr) {
			fmt.Printf("  → deficit: %.2f\n", fundsErr.Amount-fundsErr.Balance)
		}
	}

	// Перевод 2: аккаунт не существует
	err = Transfer("charlie", "alice", 10.0, balances)
	if err != nil {
		fmt.Println(err)
	}

	// Перевод 3: успех
	err = Transfer("alice", "bob", 30.0, balances)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Printf(
			"transfer alice → bob: OK (alice: %.2f, bob: %.2f)\n",
			balances["alice"], balances["bob"],
		)
	}
}
```

**Вывод:**

```
transfer: account bob: insufficient funds: balance 50.00, requested 75.00
  → deficit: 25.00
account not found: charlie
transfer alice → bob: OK (alice: 70.00, bob: 80.00)
```

### Что здесь важно

- `InsufficientFundsError` реализует `error` неявно — достаточно метода `Error() string`.
- Используем **указатель** `*InsufficientFundsError` как receiver — иначе `errors.As` не сможет извлечь значение.
- `errors.As` — безопасный способ извлечь кастомную ошибку из цепочки. Работает через любое количество обёрток `%w`.
- Разница с `errors.Is`: `Is` проверяет «это та самая ошибка?», `As` — «есть ли в цепочке ошибка такого типа? если да — дай мне её».
- Прямой type assertion (`err.(*InsufficientFundsError)`) сломается, если ошибка обёрнута — `errors.As` разворачивает цепочку автоматически.
