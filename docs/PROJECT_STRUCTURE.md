# Buffalo - Структура проекта

## Обзор структуры

```
buffalo/
├── cmd/                          # Точки входа приложения
│   └── buffalo/
│       ├── main.go              # Главная точка входа
│       ├── build.go             # Команда build
│       ├── init.go              # Команда init
│       ├── validate.go          # Команда validate
│       ├── clean.go             # Команда clean
│       ├── watch.go             # Команда watch
│       ├── config.go            # Команда config
│       ├── plugin.go            # Команда plugin
│       └── version.go           # Команда version
│
├── internal/                     # Внутренний код (не экспортируется)
│   ├── config/                  # Управление конфигурацией
│   │   ├── config.go           # Основные структуры конфигурации
│   │   ├── loader.go           # Загрузка конфигурации
│   │   ├── validator.go        # Валидация конфигурации
│   │   ├── merger.go           # Мержинг конфигураций
│   │   └── watcher.go          # Hot-reload конфигурации
│   │
│   ├── orchestrator/            # Оркестрация сборки
│   │   ├── orchestrator.go     # Главный оркестратор
│   │   ├── workflow.go         # Workflow управление
│   │   └── executor.go         # Выполнение задач
│   │
│   ├── scanner/                 # Сканирование proto файлов
│   │   ├── scanner.go          # Основной сканер
│   │   ├── parser.go           # Парсер proto файлов
│   │   ├── watcher.go          # File watching
│   │   └── hasher.go           # Хэширование файлов
│   │
│   ├── resolver/                # Разрешение зависимостей
│   │   ├── resolver.go         # Dependency resolver
│   │   ├── graph.go            # Граф зависимостей
│   │   ├── topological.go      # Топологическая сортировка
│   │   └── cycle_detector.go   # Обнаружение циклов
│   │
│   ├── scheduler/               # Планирование сборки
│   │   ├── scheduler.go        # Планировщик задач
│   │   ├── optimizer.go        # Оптимизатор плана
│   │   ├── plan.go             # Структуры плана сборки
│   │   └── priority.go         # Приоритизация задач
│   │
│   ├── cache/                   # Управление кэшем
│   │   ├── cache.go            # Интерфейс кэша
│   │   ├── manager.go          # Менеджер кэша
│   │   ├── storage.go          # Хранилище кэша
│   │   ├── file_storage.go     # Файловое хранилище
│   │   ├── memory_storage.go   # In-memory хранилище
│   │   └── invalidator.go      # Инвалидация кэша
│   │
│   ├── compiler/                # Компиляторы
│   │   ├── compiler.go         # Интерфейс компилятора
│   │   ├── base.go             # Базовый компилятор
│   │   ├── factory.go          # Фабрика компиляторов
│   │   ├── executor.go         # Выполнение команд
│   │   │
│   │   ├── python/             # Python компилятор
│   │   │   ├── compiler.go
│   │   │   ├── grpc_tools.go
│   │   │   ├── mypy_stubs.go
│   │   │   ├── init_gen.go
│   │   │   └── setup_gen.go
│   │   │
│   │   ├── golang/             # Go компилятор
│   │   │   ├── compiler.go
│   │   │   ├── package.go
│   │   │   ├── module.go
│   │   │   └── gomod_gen.go
│   │   │
│   │   ├── rust/               # Rust компилятор
│   │   │   ├── compiler.go
│   │   │   ├── prost.go
│   │   │   ├── tonic.go
│   │   │   ├── cargo_gen.go
│   │   │   └── build_rs_gen.go
│   │   │
│   │   └── cpp/                # C++ компилятор
│   │       ├── compiler.go
│   │       ├── grpc.go
│   │       ├── cmake_gen.go
│   │       └── pkgconfig_gen.go
│   │
│   ├── output/                  # Управление выходом
│   │   ├── manager.go          # Менеджер вывода
│   │   ├── layout.go           # Layout стратегии
│   │   ├── flat_layout.go      # Плоская структура
│   │   ├── language_layout.go  # По языкам
│   │   ├── mirror_layout.go    # Зеркало proto структуры
│   │   └── writer.go           # Запись файлов
│   │
│   ├── plugin/                  # Система плагинов
│   │   ├── plugin.go           # Интерфейс плагина
│   │   ├── manager.go          # Менеджер плагинов
│   │   ├── loader.go           # Загрузка плагинов
│   │   ├── executor.go         # Выполнение плагинов
│   │   └── registry.go         # Реестр плагинов
│   │
│   └── version/                 # Версионирование
│       ├── version.go          # Информация о версии
│       └── build_info.go       # Build информация
│
├── pkg/                         # Публичные библиотеки (экспортируются)
│   ├── logger/                  # Кастомный логгер
│   │   ├── logger.go           # Основной логгер
│   │   ├── level.go            # Уровни логирования
│   │   ├── formatter.go        # Интерфейс форматтера
│   │   ├── json_formatter.go   # JSON форматтер
│   │   ├── text_formatter.go   # Text форматтер
│   │   ├── colored_formatter.go # Цветной форматтер
│   │   ├── output.go           # Интерфейс вывода
│   │   ├── console_output.go   # Консольный вывод
│   │   ├── file_output.go      # Файловый вывод
│   │   ├── rotation.go         # Ротация логов
│   │   └── context.go          # Context support
│   │
│   ├── errors/                  # Обработка ошибок
│   │   ├── errors.go           # Базовые ошибки
│   │   ├── codes.go            # Коды ошибок
│   │   ├── stack.go            # Stack traces
│   │   ├── wrap.go             # Error wrapping
│   │   └── reporter.go         # Error reporting
│   │
│   ├── utils/                   # Утилиты
│   │   ├── file.go             # Файловые операции
│   │   ├── path.go             # Работа с путями
│   │   ├── hash.go             # Хэширование
│   │   ├── validator.go        # Валидация
│   │   ├── worker_pool.go      # Worker pool
│   │   └── exec.go             # Выполнение команд
│   │
│   ├── metrics/                 # Метрики
│   │   ├── collector.go        # Сборщик метрик
│   │   ├── metric.go           # Интерфейс метрики
│   │   ├── counter.go          # Counter метрика
│   │   ├── gauge.go            # Gauge метрика
│   │   ├── histogram.go        # Histogram метрика
│   │   ├── summary.go          # Summary метрика
│   │   └── exporter.go         # Экспорт метрик
│   │
│   └── proto/                   # Protobuf определения
│       ├── config.proto        # Конфигурация
│       ├── build.proto         # Сборка
│       └── plugin.proto        # Плагины
│
├── docs/                        # Документация
│   ├── ROADMAP.md              # Roadmap проекта
│   ├── ARCHITECTURE.md         # Архитектура
│   ├── PROJECT_STRUCTURE.md    # Структура проекта (этот файл)
│   ├── DEVELOPMENT.md          # Руководство по разработке
│   ├── CONFIGURATION.md        # Документация по конфигурации
│   ├── CLI_REFERENCE.md        # Справка по CLI
│   ├── PLUGIN_DEVELOPMENT.md   # Разработка плагинов
│   ├── API_REFERENCE.md        # API документация
│   ├── DEPLOYMENT.md           # Развертывание
│   ├── TROUBLESHOOTING.md      # Решение проблем
│   │
│   ├── tutorials/              # Обучающие материалы
│   │   ├── getting_started.md
│   │   ├── basic_usage.md
│   │   ├── advanced_config.md
│   │   ├── plugin_creation.md
│   │   └── ci_cd_integration.md
│   │
│   └── examples/               # Примеры конфигураций
│       ├── simple_project.yaml
│       ├── multi_language.yaml
│       ├── monorepo.yaml
│       └── with_plugins.yaml
│
├── examples/                    # Примеры использования
│   ├── python/                 # Python примеры
│   │   ├── simple/
│   │   ├── grpc_service/
│   │   └── with_mypy/
│   │
│   ├── go/                     # Go примеры
│   │   ├── simple/
│   │   ├── grpc_service/
│   │   └── with_modules/
│   │
│   ├── rust/                   # Rust примеры
│   │   ├── simple/
│   │   ├── tonic_service/
│   │   └── workspace/
│   │
│   ├── cpp/                    # C++ примеры
│   │   ├── simple/
│   │   ├── grpc_service/
│   │   └── cmake_integration/
│   │
│   └── multi_lang/             # Мультиязычные примеры
│       ├── microservices/
│       └── monorepo/
│
├── tests/                       # Тесты
│   ├── unit/                   # Unit тесты
│   │   ├── config/
│   │   ├── scanner/
│   │   ├── resolver/
│   │   ├── compiler/
│   │   └── ...
│   │
│   ├── integration/            # Интеграционные тесты
│   │   ├── build_test.go
│   │   ├── python_test.go
│   │   ├── go_test.go
│   │   ├── rust_test.go
│   │   ├── cpp_test.go
│   │   └── plugin_test.go
│   │
│   ├── e2e/                    # End-to-end тесты
│   │   ├── simple_build_test.go
│   │   ├── multi_lang_test.go
│   │   ├── watch_mode_test.go
│   │   └── cache_test.go
│   │
│   ├── performance/            # Performance тесты
│   │   ├── benchmark_test.go
│   │   ├── memory_test.go
│   │   └── large_project_test.go
│   │
│   ├── fixtures/               # Тестовые данные
│   │   ├── proto/
│   │   ├── configs/
│   │   └── expected/
│   │
│   └── testdata/               # Go testdata
│
├── configs/                     # Конфигурационные файлы
│   ├── default.yaml            # Дефолтная конфигурация
│   ├── production.yaml         # Production конфигурация
│   ├── development.yaml        # Development конфигурация
│   └── examples/               # Примеры конфигураций
│
├── scripts/                     # Вспомогательные скрипты
│   ├── install.sh              # Установка (Unix)
│   ├── install.ps1             # Установка (Windows)
│   ├── build.sh                # Сборка
│   ├── test.sh                 # Тестирование
│   ├── release.sh              # Создание релиза
│   ├── lint.sh                 # Линтинг
│   └── gen_docs.sh             # Генерация документации
│
├── build/                       # Build артефакты (не в git)
│   ├── bin/                    # Бинарники
│   ├── pkg/                    # Package файлы
│   └── docker/                 # Docker образы
│
├── deployments/                 # Deployment конфигурации
│   ├── docker/
│   │   ├── Dockerfile
│   │   ├── Dockerfile.alpine
│   │   └── docker-compose.yml
│   │
│   ├── kubernetes/
│   │   ├── deployment.yaml
│   │   ├── service.yaml
│   │   └── configmap.yaml
│   │
│   └── ci/
│       ├── github-actions.yml
│       ├── gitlab-ci.yml
│       └── jenkins.groovy
│
├── tools/                       # Инструменты разработки
│   ├── protoc/                 # protoc бинарники
│   ├── linters/                # Линтеры
│   └── generators/             # Code generators
│
├── third_party/                 # Внешние зависимости
│   ├── proto/                  # Сторонние proto файлы
│   │   ├── google/
│   │   └── grpc/
│   └── licenses/               # Лицензии зависимостей
│
├── .github/                     # GitHub специфичное
│   ├── workflows/              # GitHub Actions
│   │   ├── ci.yml
│   │   ├── release.yml
│   │   └── docs.yml
│   ├── ISSUE_TEMPLATE/
│   ├── PULL_REQUEST_TEMPLATE.md
│   └── dependabot.yml
│
├── .vscode/                     # VS Code настройки
│   ├── settings.json
│   ├── launch.json
│   └── tasks.json
│
├── go.mod                       # Go modules
├── go.sum                       # Go modules checksums
├── Makefile                     # Make файл
├── README.md                    # Основная документация
├── CHANGELOG.md                 # История изменений
├── CONTRIBUTING.md              # Руководство для контрибьюторов
├── LICENSE                      # Лицензия
├── .gitignore                   # Git ignore
├── .gitattributes               # Git attributes
├── .editorconfig                # Editor config
├── .golangci.yml                # GolangCI-Lint config
└── codecov.yml                  # Code coverage config
```

---

## Описание основных директорий

### `/cmd`

Точки входа приложения. Каждая команда CLI имеет свой файл. Эта директория содержит минимальный код - только инициализацию и вызов соответствующих функций из `internal`.

**Принципы:**
- Тонкий слой над internal пакетами
- Обработка CLI флагов и аргументов
- Инициализация приложения
- Graceful shutdown

### `/internal`

Внутренний код приложения, не экспортируется. Содержит всю бизнес-логику.

**Принципы:**
- Не экспортируется за пределы модуля
- Основная логика приложения
- Следование принципам SOLID
- Тестируемость

### `/pkg`

Публичные библиотеки, которые могут использоваться другими приложениями.

**Принципы:**
- Экспортируется за пределы модуля
- Общие утилиты
- Независимость от internal
- Хорошо документировано
- Стабильное API

### `/docs`

Вся документация проекта.

**Содержит:**
- Архитектурную документацию
- Руководства пользователя
- API референс
- Туториалы
- Примеры конфигураций

### `/examples`

Рабочие примеры использования buffalo для различных языков и сценариев.

**Принципы:**
- Рабочий код
- Хорошо документирован
- Покрывает разные use cases
- Легко запускается

### `/tests`

Все виды тестов.

**Типы:**
- Unit тесты - тестирование отдельных компонентов
- Integration тесты - тестирование взаимодействия компонентов
- E2E тесты - тестирование всего приложения
- Performance тесты - бенчмарки и профилирование

### `/configs`

Конфигурационные файлы для различных окружений.

**Принципы:**
- Sensible defaults
- Environment-specific configs
- Комментарии и примеры

### `/scripts`

Вспомогательные скрипты для разработки, сборки, тестирования и развертывания.

**Принципы:**
- Кроссплатформенность (где возможно)
- Хорошо документированы
- Идемпотентность
- Error handling

### `/deployments`

Конфигурации для развертывания.

**Содержит:**
- Docker образы
- Kubernetes манифесты
- CI/CD конфигурации

### `/build`

Директория для build артефактов (не в git).

**Содержит:**
- Скомпилированные бинарники
- Package файлы
- Docker образы

### `/tools`

Инструменты разработки и внешние зависимости.

**Содержит:**
- protoc бинарники для разных платформ
- Линтеры
- Code generators
- Другие инструменты

### `/third_party`

Сторонние зависимости, которые нужно включить в репозиторий.

**Содержит:**
- Сторонние proto файлы (google, grpc)
- Лицензии зависимостей

---

## Именование файлов

### Общие правила:

1. **Snake case:** `file_name.go` (стандарт Go)
2. **Тесты:** `file_name_test.go`
3. **Интерфейсы:** В файле с именем интерфейса или `interface.go`
4. **Имплементации:** `<type>_<interface>.go` (например, `file_storage.go`)

### Специальные файлы:

- `main.go` - точка входа
- `doc.go` - документация пакета
- `types.go` - основные типы
- `errors.go` - определения ошибок
- `constants.go` - константы
- `interface.go` - интерфейсы

---

## Структура пакетов

### Принципы организации:

1. **По функциональности:** Группируем по функциональному назначению
2. **Минимизация циклических зависимостей:** Чёткая иерархия зависимостей
3. **Высокая связность:** Связанные вещи в одном пакете
4. **Слабое связывание:** Пакеты независимы друг от друга
5. **Тестируемость:** Легко мокировать и тестировать

### Граф зависимостей:

```
cmd/buffalo
    ↓
internal/orchestrator
    ↓
internal/{config,scanner,resolver,scheduler,cache,compiler,output,plugin}
    ↓
pkg/{logger,errors,utils,metrics}
```

---

## Конфигурационные файлы

### `go.mod`

```go
module github.com/yourorg/buffalo

go 1.21

require (
    github.com/spf13/cobra v1.8.0
    github.com/spf13/viper v1.18.2
    gopkg.in/yaml.v3 v3.0.1
    // ... другие зависимости
)
```

### `Makefile`

Основные команды:

```makefile
.PHONY: help build test lint clean install

help:           ## Показать помощь
build:          ## Собрать бинарник
test:           ## Запустить тесты
test-coverage:  ## Тесты с покрытием
lint:           ## Линтинг кода
clean:          ## Очистить build артефакты
install:        ## Установить бинарник
run:            ## Запустить приложение
docker-build:   ## Собрать Docker образ
docs:           ## Сгенерировать документацию
```

### `.gitignore`

```gitignore
# Build
/build/
/dist/
*.exe
*.dll
*.so
*.dylib

# Test
*.test
*.out
coverage.txt
coverage.html

# IDE
.vscode/
.idea/
*.swp
*.swo

# OS
.DS_Store
Thumbs.db

# Temp
tmp/
temp/
*.tmp

# Logs
*.log
logs/

# Generated
generated/
```

### `.golangci.yml`

Конфигурация для golangci-lint:

```yaml
linters:
  enable:
    - gofmt
    - govet
    - errcheck
    - staticcheck
    - unused
    - gosimple
    - structcheck
    - varcheck
    - ineffassign
    - deadcode
    - typecheck
    - gosec
    - gocyclo
    - gocognit
    - goconst
    - misspell

linters-settings:
  gocyclo:
    min-complexity: 15
  gocognit:
    min-complexity: 20
```

---

## Workflow разработки

### 1. Новая функция:

```bash
# Создать feature branch
git checkout -b feature/my-feature

# Разработка
# ... код ...

# Тесты
make test

# Линтинг
make lint

# Commit
git commit -m "feat: add my feature"

# Push и создать PR
git push origin feature/my-feature
```

### 2. Запуск тестов:

```bash
# Все тесты
make test

# Unit тесты
go test ./internal/...

# Integration тесты
go test ./tests/integration/...

# С покрытием
make test-coverage
```

### 3. Сборка:

```bash
# Локальная сборка
make build

# Cross-compilation
make build-all

# Docker образ
make docker-build
```

---

## Best Practices

### 1. Организация кода:

- Один файл = одна ответственность
- Файлы < 500 строк (в идеале)
- Функции < 50 строк (в идеале)
- Циклическая сложность < 15

### 2. Тестирование:

- Unit тесты для всех публичных функций
- Table-driven tests
- Моки для внешних зависимостей
- Покрытие > 85%

### 3. Документация:

- Godoc комментарии для всех экспортируемых типов и функций
- Примеры использования
- Актуальность документации

### 4. Error handling:

- Всегда обрабатывать ошибки
- Использовать pkg/errors для контекста
- Не паниковать (кроме критических случаев)

### 5. Логирование:

- Использовать structured logging
- Правильные уровни (DEBUG, INFO, WARN, ERROR)
- Не логировать чувствительные данные

---

## Заключение

Эта структура обеспечивает:
- ✅ Чёткую организацию кода
- ✅ Масштабируемость
- ✅ Лёгкость навигации
- ✅ Простоту тестирования
- ✅ Удобство разработки
- ✅ Соответствие Go best practices

Структура следует стандартам сообщества Go и проверенным паттернам организации больших проектов.
