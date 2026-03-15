# Решение — Базовый Git

---

## Настройка SSH-ключа (сделать один раз перед началом)

SSH — это способ аутентификации на GitHub без ввода логина и пароля при каждом `push` и `pull`. Вместо пароля GitHub узнаёт тебя по криптографическому ключу.

Работает так: у тебя на машине хранится **приватный ключ** (никому не передавать), а на GitHub загружается **публичный ключ**. При подключении они сверяются — если совпали, GitHub пускает.

**Шаг 1. Сгенерировать ключ**

```bash
ssh-keygen -t ed25519 -C "твой@email.com"
```

Команда спросит куда сохранить (нажми Enter — сохранит в `~/.ssh/id_ed25519`) и предложит задать passphrase (можно оставить пустым, просто Enter).

После генерации появятся два файла:
- `~/.ssh/id_ed25519` — приватный ключ (не показывать никому)
- `~/.ssh/id_ed25519.pub` — публичный ключ (его добавляем на GitHub)

**Шаг 2. Скопировать публичный ключ**

```bash
cat ~/.ssh/id_ed25519.pub
```

Скопируй весь вывод — строка начинается с `ssh-ed25519` и заканчивается твоим email.

**Шаг 3. Добавить ключ на GitHub**

1. Зайди на github.com → **Settings** (аватар справа вверху)
2. Слева → **SSH and GPG keys**
3. Нажми **New SSH key**
4. Title: например `my-laptop`
5. В поле Key вставь скопированный публичный ключ
6. Нажми **Add SSH key**

**Шаг 4. Проверить что всё работает**

```bash
ssh -T git@github.com
# Ответ: Hi <логин>! You've successfully authenticated...
```

**Шаг 5. Клонировать через SSH, а не HTTPS**

На странице репозитория нажми **Code** → вкладка **SSH** → скопируй ссылку вида `git@github.com:логин/репозиторий.git`:

```bash
git clone git@github.com:<логин>/git-practice.git
```

> Если уже клонировал через HTTPS — можно поменять remote на SSH:
> ```bash
> git remote set-url origin git@github.com:<логин>/git-practice.git
> ```

---

## Часть 1 — Первый репозиторий

```bash
git clone git@github.com:<логин>/git-practice.git
cd git-practice

git config --global user.name "Имя Фамилия"
git config --global user.email "email@example.com"
```

Создай файл `about.txt` в редакторе и напиши в нём:

```
Меня зовут Иван. Учу Go чтобы стать бэкенд-разработчиком.
Интересуюсь бэкенд-разработкой.
```

```bash
git add about.txt
git commit -m "add about.txt"
git push origin main
```

---

## Часть 2 — Несколько коммитов и история

**Коммит 1.** Создай `notes.md` с заголовком:

```markdown
# Мои заметки
```

```bash
git add notes.md
git commit -m "add notes.md with title"
```

**Коммит 2.** Добавь в конец `notes.md` секцию Git:

```markdown
# Мои заметки

## Git
- git add добавляет файлы в индекс
- git commit сохраняет снимок изменений
```

```bash
git add notes.md
git commit -m "add Git section to notes"
```

**Коммит 3.** Добавь в конец `notes.md` секцию Go:

```markdown
# Мои заметки

## Git
- git add добавляет файлы в индекс
- git commit сохраняет снимок изменений

## Go
- Go компилируемый статически типизированный язык
- Подходит для написания серверных приложений
```

```bash
git add notes.md
git commit -m "add Go section to notes"
```

Посмотри историю:

```bash
git log --oneline
```

Добавь ответы в конец `about.txt` (подставь реальный хеш из вывода `git log`):

```
Меня зовут Иван. Учу Go чтобы стать бэкенд-разработчиком.
Интересуюсь бэкенд-разработкой.

Коммитов в репозитории: 5
Хеш первого коммита: a1b2c3d
```

```bash
git add about.txt
git commit -m "add git log answers"
git push origin main
```

---

## Часть 3 — Ветки

```bash
git checkout -b feature/contacts
```

Создай `contacts.md`:

```markdown
# Контакты
GitHub: https://github.com/<логин>
```

```bash
git add contacts.md
git commit -m "add contacts.md"
git push origin feature/contacts

git checkout main
git merge feature/contacts
git push origin main
```

---

## Часть 4 — Конфликт

Открой `about.txt` и измени первую строку. В `main` сохрани так:

```
Меня зовут Иван. Go разработчик.
```

```bash
git add about.txt
git commit -m "update about in main"
git checkout -b feature/update-about
```

В ветке `feature/update-about` открой `about.txt` и измени ту же первую строку иначе:

```
Меня зовут Иван. Изучаю бэкенд.
```

```bash
git add about.txt
git commit -m "update about in feature branch"

git checkout main
git merge feature/update-about
# CONFLICT (content): Merge conflict in about.txt
```

Открой `about.txt` — увидишь маркеры конфликта:

```
<<<<<<< HEAD
Меня зовут Иван. Go разработчик.
=======
Меня зовут Иван. Изучаю бэкенд.
>>>>>>> feature/update-about
```

Убери маркеры, оставь итоговый текст:

```
Меня зовут Иван. Go разработчик, изучаю бэкенд.
```

```bash
git add about.txt
git commit -m "resolve merge conflict in about.txt"
git push origin main
```

---

## Часть 5 — Откат изменений

Открой `notes.md` и добавь в конец строку-ошибку:

```markdown
## Go
- Go компилируемый статически типизированный язык
- Подходит для написания серверных приложений
- это ошибочная запись которую нужно отменить
```

```bash
git add notes.md
git commit -m "add wrong note"

git log --oneline          # скопируй хеш последнего коммита
git revert <hash>          # откроется редактор — сохрани сообщение как есть
git push origin main
```

Создай два файла в редакторе:

`needed.txt`:
```
нужный файл
```

`unwanted.txt`:
```
ненужный файл
```

```bash
git add .
git status
# оба файла в staged

git reset unwanted.txt
git status
# needed.txt — staged, unwanted.txt — untracked

git commit -m "add needed.txt"
git push origin main
```

---

## Часть 6 — Pull Request

```bash
git checkout -b feature/summary
```

Добавь в конец `notes.md` итоговую секцию:

```markdown
## Итог
- Git помогает отслеживать изменения в коде
- Ветки позволяют работать над фичами изолированно
- Pull Request — способ предложить изменения и обсудить их
```

```bash
git add notes.md
git commit -m "add summary section to notes"
git push origin feature/summary
```

В интерфейсе GitHub:
1. Нажать **Compare & pull request**
2. Заполнить заголовок и описание
3. Нажать **Create pull request**
4. Посмотреть вкладку **Files changed**
5. Нажать **Merge pull request** → **Confirm merge**

```bash
git checkout main
git pull origin main
```

