# Contributing to Buffalo

Спасибо за интерес к Buffalo! Мы рады любому вкладу.

## Кодекс поведения

Участвуя в этом проекте, вы соглашаетесь соблюдать наш [Code of Conduct](CODE_OF_CONDUCT.md).

## Как помочь проекту

### Сообщения об ошибках

Найденные баги сообщайте через [GitHub Issues](https://github.com/yourorg/buffalo/issues).

**Хороший баг-репорт включает:**
- Краткое описание проблемы
- Шаги для воспроизведения
- Ожидаемое поведение
- Фактическое поведение
- Версия Buffalo, ОС, версия Go
- Логи или скриншоты (если применимо)

**Шаблон:**
```markdown
**Описание:**
Краткое описание проблемы

**Шаги для воспроизведения:**
1. Запустить buffalo build
2. ...

**Ожидаемое поведение:**
Что должно произойти

**Фактическое поведение:**
Что произошло на самом деле

**Окружение:**
- Buffalo version: v1.0.0
- OS: Ubuntu 22.04
- Go version: 1.21.5
```

### Предложения новых функций

Предложения принимаются через [GitHub Discussions](https://github.com/yourorg/buffalo/discussions).

**Хорошее предложение включает:**
- Описание проблемы, которую решает
- Предлагаемое решение
- Альтернативы (если есть)
- Примеры использования

### Pull Requests

1. **Fork** репозитория
2. **Создайте** feature branch (`git checkout -b feature/amazing-feature`)
3. **Сделайте** изменения
4. **Добавьте** тесты
5. **Убедитесь** что тесты проходят (`make test`)
6. **Запустите** линтер (`make lint`)
7. **Commit** изменений (`git commit -m 'feat: add amazing feature'`)
8. **Push** в branch (`git push origin feature/amazing-feature`)
9. **Откройте** Pull Request

## Стиль кода

### Go

Мы следуем стандартному стилю Go:

- `gofmt` для форматирования
- `go vet` для проверки
- `golangci-lint` для дополнительных проверок

```bash
# Форматирование
make fmt

# Проверки
make lint
```

### Соглашения об именовании

- **Файлы:** `snake_case.go`
- **Пакеты:** `lowercase` (один word, без underscore)
- **Типы:** `PascalCase`
- **Функции/методы:** `PascalCase` (экспортируемые), `camelCase` (приватные)
- **Переменные:** `camelCase`
- **Константы:** `PascalCase` или `SCREAMING_SNAKE_CASE`

### Комментарии

```go
// Package scanner provides proto file scanning functionality.
package scanner

// Scanner scans directories for proto files.
type Scanner struct {
    paths []string
}

// NewScanner creates a new Scanner instance.
func NewScanner(paths []string) *Scanner {
    return &Scanner{paths: paths}
}

// Scan scans all configured paths and returns found proto files.
// Returns an error if any path is inaccessible.
func (s *Scanner) Scan() ([]*ProtoFile, error) {
    // Implementation
}
```

### Обработка ошибок

```go
// ✅ Правильно
if err != nil {
    return fmt.Errorf("failed to scan directory: %w", err)
}

// ❌ Неправильно
if err != nil {
    panic(err) // Не использовать panic для обычных ошибок
}
```

## Тесты

### Требования

- Unit тесты для всех новых функций
- Integration тесты для комплексной функциональности
- Покрытие тестами > 80% для нового кода

### Написание тестов

```go
func TestScanner_Scan(t *testing.T) {
    tests := []struct {
        name    string
        paths   []string
        want    int
        wantErr bool
    }{
        {
            name:    "valid paths",
            paths:   []string{"testdata/proto"},
            want:    5,
            wantErr: false,
        },
        {
            name:    "invalid path",
            paths:   []string{"nonexistent"},
            want:    0,
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            s := NewScanner(tt.paths)
            got, err := s.Scan()
            
            if (err != nil) != tt.wantErr {
                t.Errorf("Scan() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            
            if len(got) != tt.want {
                t.Errorf("Scan() got %d files, want %d", len(got), tt.want)
            }
        })
    }
}
```

### Запуск тестов

```bash
# Все тесты
make test

# Конкретный пакет
go test ./internal/scanner/...

# С покрытием
make test-coverage

# С race detector
go test -race ./...
```

## Коммиты

### Conventional Commits

Мы используем [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Типы

- `feat:` - новая функциональность
- `fix:` - исправление бага
- `docs:` - изменения в документации
- `style:` - форматирование, отсутствие изменений в коде
- `refactor:` - рефакторинг кода
- `perf:` - улучшение производительности
- `test:` - добавление или исправление тестов
- `build:` - изменения в системе сборки
- `ci:` - изменения в CI конфигурации
- `chore:` - прочие изменения

### Примеры

```bash
feat(compiler): add rust compiler support

fix(scanner): fix proto file detection on Windows
Fixes #123

docs: update installation instructions

refactor(cache): simplify cache invalidation logic

test(resolver): add tests for cycle detection

chore: bump dependencies
```

### Breaking Changes

```bash
feat(config)!: change config file format

BREAKING CHANGE: Config files now use YAML instead of JSON.
Migration guide: docs/migrations/v1-to-v2.md
```

## Документация

### Обновление документации

При добавлении новой функциональности:

1. Обновите соответствующий файл в `/docs`
2. Добавьте примеры в `/examples`
3. Обновите `README.md` если нужно
4. Добавьте запись в `CHANGELOG.md`

### Godoc

Все экспортируемые типы и функции должны иметь комментарии:

```go
// Scanner scans directories for proto files.
// It supports recursive scanning and filtering by patterns.
type Scanner struct {
    // fields
}
```

## Pull Request процесс

### Перед созданием PR

- [ ] Код следует стилю проекта
- [ ] Добавлены тесты
- [ ] Все тесты проходят
- [ ] Линтер не выдает ошибок
- [ ] Документация обновлена
- [ ] CHANGELOG.md обновлен

### Описание PR

Используйте шаблон:

```markdown
## Описание
Краткое описание изменений

## Мотивация и контекст
Почему эти изменения необходимы? Какую проблему решают?

Closes #(issue)

## Тип изменений
- [ ] Bug fix (non-breaking change)
- [ ] New feature (non-breaking change)
- [ ] Breaking change
- [ ] Documentation update

## Как протестировано?
Опишите тесты, которые вы провели

## Checklist
- [ ] Код следует стилю проекта
- [ ] Проведен self-review
- [ ] Добавлены комментарии в сложных местах
- [ ] Документация обновлена
- [ ] Нет предупреждений от линтера
- [ ] Добавлены тесты
- [ ] Все тесты проходят
```

### Code Review

- Отвечайте на комментарии
- Вносите запрошенные изменения
- Обновляйте PR после изменений
- Будьте открыты к обратной связи

## Настройка окружения разработки

```bash
# Клонирование
git clone https://github.com/yourorg/buffalo.git
cd buffalo

# Установка зависимостей
go mod download

# Установка инструментов
make install-tools

# Проверка настройки
make check

# Запуск тестов
make test
```

## Структура проекта

См. [docs/PROJECT_STRUCTURE.md](docs/PROJECT_STRUCTURE.md)

## Добавление нового компилятора

См. [docs/DEVELOPMENT.md#adding-new-compiler](docs/DEVELOPMENT.md#adding-new-compiler)

## Создание плагина

См. [docs/PLUGIN_DEVELOPMENT.md](docs/PLUGIN_DEVELOPMENT.md)

## Релиз процесс

Релизы делаются maintainers:

1. Обновление версии в `internal/version/version.go`
2. Обновление `CHANGELOG.md`
3. Создание git тега
4. Push тега запускает CI/CD для релиза

## Вопросы?

- 💬 [GitHub Discussions](https://github.com/yourorg/buffalo/discussions)
- 💬 [Discord](https://discord.gg/buffalo)
- 📫 buffalo@yourorg.com

## Лицензия

Внося вклад в Buffalo, вы соглашаетесь с тем, что ваш вклад будет лицензирован под MIT License.
