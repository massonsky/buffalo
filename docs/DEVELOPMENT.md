# Buffalo Development Guide

## Быстрый старт для разработчиков

### Требования

#### Обязательные:
- **Go:** >= 1.21
- **Git:** >= 2.0
- **Make:** для удобства разработки

#### Опциональные (для работы с примерами):
- **Python:** >= 3.8
- **Rust:** >= 1.70
- **C++:** >= C++17 компилятор
- **protoc:** >= 3.20

---

## Настройка окружения разработки

### 1. Клонирование репозитория

```bash
git clone https://github.com/yourorg/buffalo.git
cd buffalo
```

### 2. Установка зависимостей

```bash
# Go модули
go mod download

# Инструменты разработки
make install-tools
```

### 3. Настройка IDE (VS Code)

Рекомендуемые расширения:
- Go (golang.go)
- YAML (redhat.vscode-yaml)
- Protocol Buffers (zxh404.vscode-proto3)
- Better Comments (aaron-bond.better-comments)

```bash
# Установить расширения
code --install-extension golang.go
code --install-extension redhat.vscode-yaml
code --install-extension zxh404.vscode-proto3
```

---

## Основные команды разработки

### Сборка

```bash
# Локальная сборка
make build

# Сборка для всех платформ
make build-all

# Установка в $GOPATH/bin
make install

# Сборка с отладочной информацией
make build-debug
```

### Тестирование

```bash
# Все тесты
make test

# Unit тесты
make test-unit

# Integration тесты
make test-integration

# E2E тесты
make test-e2e

# Тесты с покрытием
make test-coverage

# Открыть coverage report в браузере
make coverage-html

# Бенчмарки
make benchmark
```

### Качество кода

```bash
# Линтинг
make lint

# Форматирование
make fmt

# Vet проверки
make vet

# Все проверки (fmt + vet + lint)
make check

# Исправить автофиксимые проблемы
make fix
```

### Запуск

```bash
# Запуск из исходников
go run cmd/buffalo/main.go build --help

# Запуск установленного бинарника
buffalo build --help

# Запуск с конкретным конфигом
buffalo build --config examples/configs/simple.yaml

# Debug режим
buffalo build --verbose --dry-run
```

---

## Структура коммитов

Мы используем [Conventional Commits](https://www.conventionalcommits.org/):

### Формат:

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Типы:

- `feat:` - новая функциональность
- `fix:` - исправление бага
- `docs:` - изменения в документации
- `style:` - форматирование, отсутствующие точки с запятой и т.д.
- `refactor:` - рефакторинг кода
- `perf:` - улучшение производительности
- `test:` - добавление или исправление тестов
- `build:` - изменения в системе сборки
- `ci:` - изменения в CI конфигурации
- `chore:` - прочие изменения

### Примеры:

```bash
# Новая функция
git commit -m "feat(compiler): add rust compiler support"

# Исправление бага
git commit -m "fix(scanner): fix proto file detection on Windows"

# Документация
git commit -m "docs: update installation instructions"

# Breaking change
git commit -m "feat(config)!: change config file format to YAML

BREAKING CHANGE: Config files now use YAML instead of JSON."
```

---

## Процесс разработки

### 1. Создание feature branch

```bash
# Обновить main
git checkout main
git pull origin main

# Создать feature branch
git checkout -b feature/my-awesome-feature

# Или для багфикса
git checkout -b fix/bug-description
```

### 2. Разработка

```bash
# Писать код
# Писать тесты
# Обновлять документацию

# Проверять код
make check
make test
```

### 3. Коммит изменений

```bash
# Добавить файлы
git add .

# Коммит
git commit -m "feat: add my awesome feature"

# Если нужно, амендить
git commit --amend
```

### 4. Push и Pull Request

```bash
# Push в remote
git push origin feature/my-awesome-feature

# Создать PR на GitHub
# Заполнить описание по шаблону
# Запросить review
```

### 5. Code Review

- Ответить на комментарии
- Внести изменения если требуется
- Push изменений
- Дождаться approve

### 6. Merge

После approve:
- Squash commits (опционально)
- Merge в main
- Удалить feature branch

---

## Написание тестов

### Unit тесты

```go
// internal/scanner/scanner_test.go
package scanner

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestScanner_Scan(t *testing.T) {
    tests := []struct {
        name    string
        paths   []string
        want    int
        wantErr bool
    }{
        {
            name:    "scan valid proto files",
            paths:   []string{"testdata/proto"},
            want:    5,
            wantErr: false,
        },
        {
            name:    "scan empty directory",
            paths:   []string{"testdata/empty"},
            want:    0,
            wantErr: false,
        },
        {
            name:    "scan non-existent path",
            paths:   []string{"testdata/nonexistent"},
            want:    0,
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            s := NewScanner()
            files, err := s.Scan(tt.paths)
            
            if tt.wantErr {
                require.Error(t, err)
                return
            }
            
            require.NoError(t, err)
            assert.Len(t, files, tt.want)
        })
    }
}
```

### Integration тесты

```go
// tests/integration/python_test.go
package integration

import (
    "context"
    "testing"
    "github.com/stretchr/testify/suite"
)

type PythonCompilerSuite struct {
    suite.Suite
    compiler *python.Compiler
    tempDir  string
}

func (s *PythonCompilerSuite) SetupTest() {
    // Подготовка окружения для каждого теста
    s.tempDir = s.T().TempDir()
    s.compiler = python.NewCompiler(config)
}

func (s *PythonCompilerSuite) TestCompileSimpleProto() {
    // Тест компиляции простого proto файла
    result, err := s.compiler.Compile(context.Background(), task)
    s.Require().NoError(err)
    s.True(result.Success)
    s.NotEmpty(result.OutputFiles)
}

func TestPythonCompiler(t *testing.T) {
    suite.Run(t, new(PythonCompilerSuite))
}
```

### Бенчмарки

```go
func BenchmarkScanner_Scan(b *testing.B) {
    scanner := NewScanner()
    paths := []string{"testdata/proto"}
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := scanner.Scan(paths)
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

---

## Логирование

### Использование логгера

```go
import "github.com/yourorg/buffalo/pkg/logger"

// Создание логгера
log := logger.New(
    logger.WithLevel(logger.INFO),
    logger.WithFormatter(logger.NewJSONFormatter()),
)

// Простое логирование
log.Info("starting build")
log.Error("compilation failed", logger.String("file", "test.proto"))

// С контекстом
log.WithFields(logger.Fields{
    "component": "compiler",
    "language": "python",
}).Debug("executing protoc command")

// С Context
ctx := context.Background()
log.WithContext(ctx).Info("processing request")
```

### Уровни логирования

- **DEBUG:** Детальная отладочная информация
- **INFO:** Информационные сообщения
- **WARN:** Предупреждения (не критичные)
- **ERROR:** Ошибки (требуют внимания)
- **FATAL:** Критические ошибки (завершают приложение)

---

## Обработка ошибок

### Создание ошибок

```go
import "github.com/yourorg/buffalo/pkg/errors"

// Новая ошибка
err := errors.New(
    errors.ErrProtoNotFound,
    "proto file not found: %s", filePath,
)

// Wrap существующей ошибки
err = errors.Wrap(
    originalErr,
    errors.ErrCompilation,
    "failed to compile proto file",
)

// С контекстом
err = errors.WithContext(err, map[string]interface{}{
    "file": filePath,
    "line": lineNum,
})
```

### Проверка ошибок

```go
if err != nil {
    // Проверка типа ошибки
    if errors.Is(err, errors.ErrProtoNotFound) {
        // Обработка конкретной ошибки
    }
    
    // Получение деталей
    if bufErr, ok := err.(*errors.Error); ok {
        log.Error("error occurred",
            logger.String("code", string(bufErr.Code)),
            logger.Any("context", bufErr.Context),
        )
    }
    
    return err
}
```

---

## Работа с конфигурацией

### Загрузка конфигурации

```go
import "github.com/yourorg/buffalo/internal/config"

// Загрузка из файла
cfg, err := config.Load("buffalo.yaml")
if err != nil {
    return err
}

// С учетом переменных окружения
cfg, err := config.LoadWithEnv("buffalo.yaml")

// С CLI флагами
cfg, err := config.LoadWithFlags("buffalo.yaml", cliFlags)
```

### Валидация конфигурации

```go
// Валидация
if err := cfg.Validate(); err != nil {
    return fmt.Errorf("invalid config: %w", err)
}

// Применение defaults
cfg = cfg.WithDefaults()
```

---

## Добавление нового компилятора

### 1. Создать структуру

```go
// internal/compiler/mylang/compiler.go
package mylang

import (
    "context"
    "github.com/yourorg/buffalo/internal/compiler"
)

type Compiler struct {
    *compiler.BaseCompiler
    // Специфичные для языка поля
}

func NewCompiler(config *config.LanguageConfig) *Compiler {
    return &Compiler{
        BaseCompiler: compiler.NewBase("mylang", config),
    }
}
```

### 2. Реализовать интерфейс

```go
func (c *Compiler) Compile(ctx context.Context, task *compiler.BuildTask) (*compiler.CompileResult, error) {
    // 1. Подготовка команды
    cmd := c.buildCommand(task)
    
    // 2. Выполнение
    if err := c.Execute(ctx, cmd); err != nil {
        return nil, err
    }
    
    // 3. Валидация результата
    files, err := c.validateOutput(task.OutputPath)
    if err != nil {
        return nil, err
    }
    
    return &compiler.CompileResult{
        Success:     true,
        OutputFiles: files,
    }, nil
}

func (c *Compiler) Validate(config *config.LanguageConfig) error {
    // Валидация конфигурации
    return nil
}
```

### 3. Зарегистрировать

```go
// internal/compiler/factory.go
func init() {
    Register("mylang", mylang.NewCompiler)
}
```

### 4. Добавить тесты

```go
// internal/compiler/mylang/compiler_test.go
func TestMyLangCompiler_Compile(t *testing.T) {
    // Тесты компилятора
}
```

---

## Профилирование

### CPU профилирование

```bash
# Запустить с профилированием
go test -cpuprofile=cpu.prof -bench=.

# Анализ
go tool pprof cpu.prof
```

### Memory профилирование

```bash
# Запустить с профилированием
go test -memprofile=mem.prof -bench=.

# Анализ
go tool pprof mem.prof
```

### В коде

```go
import _ "net/http/pprof"

func main() {
    go func() {
        log.Println(http.ListenAndServe("localhost:6060", nil))
    }()
    
    // ... остальной код
}
```

Затем открыть http://localhost:6060/debug/pprof/

---

## Отладка

### Delve debugger

```bash
# Установка
go install github.com/go-delve/delve/cmd/dlv@latest

# Запуск
dlv debug cmd/buffalo/main.go -- build --config test.yaml

# В debug сессии
(dlv) break main.main
(dlv) continue
(dlv) print variable
(dlv) next
(dlv) step
```

### VS Code debugging

В `.vscode/launch.json`:

```json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Debug Buffalo",
            "type": "go",
            "request": "launch",
            "mode": "debug",
            "program": "${workspaceFolder}/cmd/buffalo",
            "args": ["build", "--config", "test.yaml", "--verbose"]
        }
    ]
}
```

---

## CI/CD

### GitHub Actions

Автоматически запускается при:
- Push в main
- Pull request
- Создании тега

**Проверки:**
- Линтинг
- Тесты (все платформы)
- Сборка (все платформы)
- Coverage отчет
- Security scan

### Локальный запуск CI

```bash
# Симуляция CI локально
make ci
```

---

## Релиз процесс

### 1. Подготовка

```bash
# Обновить версию
vim internal/version/version.go

# Обновить CHANGELOG
vim CHANGELOG.md

# Коммит
git commit -am "chore: bump version to v1.0.0"
```

### 2. Создание тега

```bash
# Создать тег
git tag -a v1.0.0 -m "Release v1.0.0"

# Push тег
git push origin v1.0.0
```

### 3. GitHub Actions автоматически:

- Соберет бинарники для всех платформ
- Создаст Docker образы
- Опубликует GitHub Release
- Обновит документацию

---

## Полезные ресурсы

### Go:
- [Effective Go](https://golang.org/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)

### Protobuf/gRPC:
- [Protocol Buffers Documentation](https://developers.google.com/protocol-buffers)
- [gRPC Documentation](https://grpc.io/docs/)

### Тестирование:
- [testify](https://github.com/stretchr/testify)
- [gomock](https://github.com/golang/mock)

---

## FAQ

**Q: Как добавить новую команду CLI?**
A: Создай новый файл в `cmd/buffalo/`, определи cobra команду, зарегистрируй в root command.

**Q: Как добавить новую метрику?**
A: Используй `pkg/metrics.Collector`, добавь новую метрику и вызывай `Record*` методы.

**Q: Как изменить формат конфигурации?**
A: Обнови структуры в `internal/config/`, обнови валидацию, обнови документацию, добавь миграцию.

**Q: Где смотреть примеры кода?**
A: Смотри `examples/` директорию и существующие компиляторы.

---

## Контакты

- **Issues:** https://github.com/yourorg/buffalo/issues
- **Discussions:** https://github.com/yourorg/buffalo/discussions
- **Discord:** https://discord.gg/buffalo
- **Email:** buffalo@yourorg.com
