# Решение — Конфигурация

---

## internal/config/config.go

```go
package config

import (
	"log"
	"os"
	"path/filepath"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	AppName     string `yaml:"app_name"     env:"APP_NAME"     env-required:"true"`
	Currency    string `yaml:"currency"     env:"CURRENCY"     env-default:"руб."`
	MaxProducts int    `yaml:"max_products" env:"MAX_PRODUCTS" env-default:"100"`
}

func MustLoad() *Config {
	// Находим папку где лежит бинарник
	executable, err := os.Executable()
	if err != nil {
		log.Fatal("Не удалось определить путь до бинарника: ", err)
	}
	configPath := filepath.Join(filepath.Dir(executable), "config.yml")

	cfg := &Config{}
	if err := cleanenv.ReadConfig(configPath, cfg); err != nil {
		log.Fatal("Не удалось загрузить конфигурацию: ", err)
	}
	return cfg
}
```

**Почему динамический путь, а не `"config.yml"`:**

Захардкоженная строка `"config.yml"` работает только если программа запускается из корня проекта. Запустишь из другой папки — файл не найдётся. `os.Executable()` всегда возвращает путь до самого бинарника, независимо от того откуда он запущен.

**Теги `env-required` и `env-default`:**

`env-required:"true"` — поле обязательное. Если его нет ни в YAML ни в переменных окружения — программа упадёт с ошибкой при старте.

`env-default:"руб."` — если поле не задано нигде, используется это значение.

**Приоритет источников в cleanenv:**

Переменная окружения перекрывает значение из YAML. Это удобно: в `config.yml` хранятся дефолты для разработки, а на сервере через `APP_NAME=...` можно переопределить нужные значения без изменения файла.

---

## cmd/main.go

```go
package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"shop/internal/config"
)

type Product struct {
	Name     string
	Price    float64
	Quantity int
}

func main() {
	cfg := config.MustLoad()

	fmt.Printf("Добро пожаловать в %s\n", cfg.AppName)
	fmt.Println("Команды: list, add <название> <цена> <кол-во>, exit")

	products := []Product{}
	scanner := bufio.NewScanner(os.Stdin)

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
				fmt.Printf("%d. %-12s — %.2f %s x %d шт.\n",
					i+1, p.Name, p.Price, cfg.Currency, p.Quantity)
			}

		case "add":
			if len(parts) < 4 {
				fmt.Println("Использование: add <название> <цена> <кол-во>")
				continue
			}
			if len(products) >= cfg.MaxProducts {
				fmt.Printf("Достигнут лимит товаров (%d)\n", cfg.MaxProducts)
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

		case "exit":
			fmt.Println("Магазин закрыт.")
			return

		default:
			fmt.Println("Неизвестная команда")
		}
	}
}
```