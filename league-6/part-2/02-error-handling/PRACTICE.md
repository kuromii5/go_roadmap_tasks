# Практика — Error handling

---

## Задача 1 — Цепочка ошибок

Ты пишешь загрузчик конфига для приложения. Конфиг читается из файла, парсится и валидируется. Каждый слой оборачивает ошибку предыдущего.

Реализуй три функции, которые вызываются по цепочке:

```go
func readFile(path string) (string, error)
func parseConfig(data string) (map[string]string, error)
func LoadConfig(path string) (map[string]string, error)
```

**Правила:**

- `readFile` — если `path` пустой, возвращает `ErrEmptyPath` (определи через `errors.New`)
- `parseConfig` — получает строку вида `"key1=value1;key2=value2"`. Если встречает элемент без `=`, возвращает ошибку с `fmt.Errorf` и текстом `"invalid entry: <элемент>"`
- `LoadConfig` — вызывает `readFile`, затем `parseConfig`. Каждую ошибку оборачивает через `%w` с контекстом слоя

В `main`:
1. Вызови `LoadConfig("")` — поймай ошибку и проверь через `errors.Is`, что корневая причина — `ErrEmptyPath`
2. Вызови `LoadConfig("valid")` с данными `"host=localhost;port=8080;broken"` — выведи полную цепочку ошибки

**Ожидаемый вывод (примерный):**

```
errors.Is ErrEmptyPath: true
full chain: load config: parse: invalid entry: broken
```

**Подсказка:**

`readFile` в этой задаче не читает реальный файл — если `path` не пустой, просто возвращай захардкоженную строку `"host=localhost;port=8080;broken"`.

---

## Задача 2 — Найди и исправь

В коде ниже **5 ошибок** в обработке ошибок. Найди и исправь каждую.

Код компилируется и даже «работает» — но обработка ошибок сломана.

```go
package main

import (
	"fmt"
	"strconv"
)

func parseAge(input string) int {
	age, _ := strconv.Atoi(input)
	return age
}

func validateAge(age int) error {
	if age < 0 || age > 150 {
		panic("invalid age")
	}
	return nil
}

func processUser(name string, ageStr string) error {
	if name == "" {
		return fmt.Errorf("empty name")
	}

	age := parseAge(ageStr)

	err := validateAge(age)
	if err != nil {
		return fmt.Errorf("validation failed: %v", err)
	}

	fmt.Printf("user %s, age %d\n", name, age)
	return nil
}

func main() {
	err := processUser("Alex", "not_a_number")
	fmt.Println(err)

	err = processUser("", "25")
	fmt.Println(err)

	err = processUser("Bob", "-5")
	fmt.Println(err)
}
```

**Что нужно сделать:**

Перепиши код так, чтобы:

1. Ни одна ошибка не игнорировалась
2. `panic` использовался только там, где это оправдано (спойлер: здесь — нигде)
3. Ошибки оборачивались через `%w`, а не `%v`
4. Каждая функция корректно прокидывала ошибки наверх
5. `main` мог получить и вывести ошибку для **каждого** вызова

**Ожидаемый вывод после исправления:**

```
process user: parse age: strconv.Atoi: parsing "not_a_number": invalid syntax
process user: empty name
process user: validation: invalid age: -5
```

---

## Задача 3 — Кастомный тип ошибки

Ты пишешь сервис переводов денег. При нехватке средств нужно возвращать не просто текст ошибки, а структурированные данные — чтобы вызывающий код мог извлечь детали.

### Шаг 1 — Создай кастомный тип ошибки

```go
type InsufficientFundsError struct {
    AccountID string
    Balance   float64
    Amount    float64
}
```

Реализуй метод `Error() string`, который возвращает:

```
account <ID>: insufficient funds: balance <Balance>, requested <Amount>
```

### Шаг 2 — Функция перевода

```go
func Transfer(from, to string, amount float64, balances map[string]float64) error
```

- Если аккаунта `from` нет в `balances` — верни ошибку `"account not found: <from>"`
- Если баланс меньше `amount` — верни `InsufficientFundsError` (обёрнутую через `%w`)
- Если всё ок — спиши деньги с `from`, начисли на `to`, верни `nil`

### Шаг 3 — main

```go
func main() {
    balances := map[string]float64{
        "alice": 100.0,
        "bob":   50.0,
    }

    // Перевод 1: bob → alice, 75.0 (не хватает)
    // Перевод 2: charlie → alice, 10.0 (аккаунт не существует)
    // Перевод 3: alice → bob, 30.0 (успех)
}
```

Для **Перевода 1**: используй `errors.As`, чтобы извлечь `InsufficientFundsError` и вывести отдельно `Balance` и `Amount`.

**Ожидаемый вывод:**

```
transfer: account bob: insufficient funds: balance 50.00, requested 75.00
  → deficit: 25.00
account not found: charlie
transfer alice → bob: OK (alice: 70.00, bob: 80.00)
```

**Требования:**

- `InsufficientFundsError` реализует `error` интерфейс
- Используй `errors.As` для извлечения кастомной ошибки — не type assertion напрямую
- Дефицит (`deficit`) вычисляется из полей извлечённой ошибки, а не хардкодится

