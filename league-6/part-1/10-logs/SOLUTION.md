# Решение — Логирование

---

## internal/logger/logger.go

```go
package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

func New(level string) *logrus.Logger {
	log := logrus.New()

	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		logLevel = logrus.InfoLevel
	}

	log.SetLevel(logLevel)
	log.SetOutput(os.Stdout)
	log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	return log
}
```

**Почему не используем `logrus.SetLevel` глобально:**

`logrus.New()` создаёт изолированный экземпляр логера. Если использовать глобальный `logrus.SetLevel` — все пакеты в программе будут делить один логер. При разрастании проекта это проблема: нельзя настроить разные уровни для разных частей. Собственный экземпляр передаётся явно туда где нужен.

---

## config.yml

```yaml
app_name: "Консольный магазин"
currency: "руб."
max_products: 100
log_level: "info"
```

---

## internal/config/config.go — добавить поле

```go
type Config struct {
	AppName     string `yaml:"app_name"     env:"APP_NAME"     env-required:"true"`
	Currency    string `yaml:"currency"     env:"CURRENCY"     env-default:"руб."`
	MaxProducts int    `yaml:"max_products" env:"MAX_PRODUCTS" env-default:"100"`
	LogLevel    string `yaml:"log_level"    env:"LOG_LEVEL"    env-default:"info"`
}
```

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
	"shop/internal/logger"
)

type Product struct {
	Name     string
	Price    float64
	Quantity int
}

func main() {
	cfg := config.MustLoad()
	log := logger.New(cfg.LogLevel)

	log.WithField("app", cfg.AppName).Info("Запуск приложения")
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
				log.WithField("limit", cfg.MaxProducts).Warn("Достигнут лимит товаров")
				fmt.Printf("Достигнут лимит товаров (%d)\n", cfg.MaxProducts)
				continue
			}
			price, err1 := strconv.ParseFloat(parts[2], 64)
			qty, err2 := strconv.Atoi(parts[3])
			if err1 != nil {
				log.WithField("input", parts[2]).Error("Неверная цена")
				fmt.Println("Неверный формат цены")
				continue
			}
			if err2 != nil {
				log.WithField("input", parts[3]).Error("Неверное количество")
				fmt.Println("Неверный формат количества")
				continue
			}
			products = append(products, Product{Name: parts[1], Price: price, Quantity: qty})
			log.WithFields(logrus.Fields{
				"name":  parts[1],
				"price": price,
			}).Info("Товар добавлен")
			log.WithField("count", len(products)).Debug("Товаров на складе")
			fmt.Println("Товар добавлен")

		case "exit":
			log.Info("Завершение работы")
			fmt.Println("Магазин закрыт.")
			return

		default:
			fmt.Println("Неизвестная команда")
		}
	}
}
```

---

## Почему разные уровни

**Info** — ключевые события бизнес-логики: запуск, добавление товара, завершение. Эти строки всегда полезны.

**Debug** — промежуточные шаги: сколько товаров сейчас в слайсе. На проде этого не нужно — засоряет логи. В разработке помогает понять что происходит внутри.

**Warn** — что-то нештатное, но программа продолжает работать. Лимит достигнут — это не ошибка, просто предупреждение.

**Error** — неверный ввод, ошибка парсинга. Нужно зафиксировать что пошло не так и с какими данными.

## WithField vs обычный Print

`log.WithField("name", value).Info("текст")` — логирует структурированно: поле отделено от сообщения. Это позволяет потом фильтровать логи по полям в ElasticSearch или любой другой системе сбора логов. Обычный `fmt.Println` такой возможности не даёт.