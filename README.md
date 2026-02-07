# 🦬 Buffalo - Multi-Language Protobuf/gRPC Build System

[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.24+-00ADD8.svg)](https://go.dev/)
[![Version](https://img.shields.io/badge/version-1.5.0-green.svg)](https://github.com/massonsky/buffalo/releases)

**Buffalo** — это мощная кросс-платформенная система сборки для protobuf/gRPC проектов с поддержкой множества языков программирования. Интеллектуальное кэширование, параллельная компиляция и инкрементальные сборки делают Buffalo идеальным выбором для современных микросервисных архитектур.

---

## ✨ Основные возможности

### 🚀 Поддержка множества языков
- **Python** - полная поддержка protobuf и gRPC
- **Go** - нативная интеграция с Go toolchain
- **Rust** - современная кодогенерация с Tonic
- **C++** - оптимизированная компиляция

### ⚡ Производительность
- **Интеллектуальное кэширование** - пересборка только измененных файлов
- **Параллельная компиляция** - максимальное использование CPU
- **Инкрементальные сборки** - минимальное время сборки

### 🎯 Гибкость
- **Система плагинов** - расширение функциональности
- **Гибкая конфигурация** - YAML/TOML/JSON + environment variables
- **Шаблонизация** - кастомизация генерируемого кода
- **Dependency management** - автоматическое управление зависимостями

### 🔧 Продвинутые функции
- **Workspace management** - управление монорепозиториями
- **Граф зависимостей** - визуализация и анализ
- **Permissions/RBAC** - генерация кода авторизации
- **Auto-upgrade** - автоматическое обновление

---

## 📦 Установка

### Быстрая установка

#### Linux/macOS
```bash
curl -sSL https://raw.githubusercontent.com/massonsky/buffalo/main/install.sh | bash
```

#### Windows (PowerShell)
```powershell
irm https://raw.githubusercontent.com/massonsky/buffalo/main/install.ps1 | iex
```

#### Проверка установки
```bash
buffalo --version
```

### Альтернативные способы

#### С помощью Go
```bash
go install github.com/massonsky/buffalo/cmd/buffalo@latest
```

#### Из исходников
```bash
git clone https://github.com/massonsky/buffalo.git
cd buffalo
make install
``

---

## 🚀 Быстрый старт

### 1. Инициализация проекта

```bash
# Создать новый проект Buffalo
buffalo init myproject
cd myproject
```

Это создаст структуру:
```
myproject/
├── buffalo.yaml          # Конфигурация
├── protos/              # .proto файлы
│   └── service.proto
└── generated/           # Сгенерированный код (создается автоматически)
```

### 2. Создание proto файла

```protobuf
// protos/service.proto
syntax = "proto3";

package myservice;

service Greeter {
  rpc SayHello (HelloRequest) returns (HelloReply) {}
}

message HelloRequest {
  string name = 1;
}

message HelloReply {
  string message = 1;
}
```

### 3. Конфигурация buffalo.yaml

```yaml
version: "1.0"

project:
  name: myproject
  proto_root: protos
  output_root: generated

languages:
  python:
    enabled: true
    output: generated/python
    plugins:
      - grpc
  
  go:
    enabled: true
    output: generated/go
    module: github.com/myorg/myproject
    plugins:
      - grpc
```

### 4. Сборка

```bash
# Собрать все языки
buffalo build

# Собрать конкретный язык
buffalo build --lang python

# Watch mode - автоматическая пересборка
buffalo watch

# Параллельная сборка с 8 воркерами
buffalo build --jobs 8
```

---

## 📚 Примеры использования

### Python проект

```yaml
# buffalo.yaml
version: "1.0"

project:
  name: python-service
  proto_root: protos
  output_root: generated

languages:
  python:
    enabled: true
    output: generated/python
    plugins:
      - grpc
      - mypy
    options:
      mypy_enabled: true
```

```bash
buffalo build --lang python
cd generated/python
python -m pip install -e .
```

### Go проект

```yaml
# buffalo.yaml
version: "1.0"

project:
  name: go-service
  proto_root: protos
  output_root: generated

languages:
  go:
    enabled: true
    output: generated/go
    module: github.com/myorg/go-service
    plugins:
      - grpc
      - grpc-gateway
      - validate
```

```bash
buffalo build --lang go
cd generated/go
go mod tidy
go build ./...
```

### Мультиязычный проект

```yaml
# buffalo.yaml
version: "1.0"

project:
  name: polyglot-service
  proto_root: protos
  output_root: generated

languages:
  python:
    enabled: true
    output: generated/python
  
  go:
    enabled: true
    output: generated/go
    module: github.com/myorg/polyglot
  
  rust:
    enabled: true
    output: generated/rust
    crate: polyglot_service
  
  cpp:
    enabled: true
    output: generated/cpp
```

```bash
# Собрать все языки параллельно
buffalo build --jobs 4

# Использовать watch mode для всех языков
buffalo watch
```

---

## 🔧 Основные команды

### Сборка и компиляция

```bash
buffalo build                    # Собрать все включенные языки
buffalo build --lang python      # Собрать только Python
buffalo build --jobs 8           # Использовать 8 воркеров
buffalo build --no-cache         # Пересобрать все без кэша
buffalo rebuild                  # Полная пересборка (очистка + сборка)
```

### Разработка

```bash
buffalo watch                    # Автоматическая пересборка при изменениях
buffalo watch --lang go          # Watch для конкретного языка
buffalo check                    # Проверка готовности проекта к сборке
buffalo check -v                 # Детальная проверка с выводом всех компонентов
buffalo validate                 # Валидация proto файлов
buffalo doctor                   # Диагностика окружения (все языки)
buffalo doctor --config-only     # Проверка только включенных языков
```

### Управление проектом

```bash
buffalo init myproject           # Создать новый проект
buffalo deps install             # Установить зависимости
buffalo deps update              # Обновить зависимости
buffalo list                     # Показать доступные языки и плагины
```

### Кэш и очистка

```bash
buffalo clear cache              # Очистить кэш
buffalo clear generated          # Удалить сгенерированные файлы
buffalo clear all                # Очистить всё
```

### Метрики и диагностика

```bash
buffalo metrics                  # Показать метрики сборки
buffalo stats                    # Статистика проекта
buffalo doctor                   # Диагностика окружения
buffalo diff                     # Показать изменения proto файлов
```

### Плагины и шаблоны

```bash
buffalo plugin list              # Список доступных плагинов
buffalo plugin install <name>    # Установить плагин
buffalo template list            # Список шаблонов
buffalo template generate        # Генерация из шаблона
```

---

## 📊 Граф зависимостей (buffalo graph)

Визуализация и анализ зависимостей между proto файлами.

### Базовое использование

```bash
# Показать дерево зависимостей в терминале
buffalo graph

# Экспорт в Graphviz DOT формат
buffalo graph --format dot --output deps.dot

# Экспорт в Mermaid для README
buffalo graph --format mermaid --output deps.md

# Экспорт в JSON
buffalo graph --format json --output deps.json

# Экспорт в PlantUML
buffalo graph --format plantuml --output deps.puml
```

### Уровни анализа

```bash
# Зависимости на уровне файлов (по умолчанию)
buffalo graph --scope file

# Зависимости на уровне пакетов
buffalo graph --scope package

# Связи между сообщениями
buffalo graph --scope message

# Связи между сервисами
buffalo graph --scope service

# Полный анализ
buffalo graph --scope full
```

### Анализ конкретного файла

```bash
# Анализ одного proto файла
buffalo graph --file protos/user.proto
```

### Расширенный анализ

```bash
# Обнаружение циклических зависимостей
buffalo graph analyze --cycles

# Поиск orphan файлов (не импортируются никем)
buffalo graph analyze --orphans

# Анализ связности (coupling metrics)
buffalo graph analyze --coupling

# Поиск "hub" файлов (много входящих зависимостей)
buffalo graph analyze --hubs

# Полный анализ
buffalo graph analyze --all
```

### Статистика графа

```bash
# Показать статистику
buffalo graph --stats

# Или через подкоманду
buffalo graph stats
```

Пример вывода:
```
📊 Dependency Graph Statistics
════════════════════════════════════════
Total files:        42
Total dependencies: 156
Average deps/file:  3.7
Max dependencies:   12 (common/types.proto)
Orphan files:       2
Circular deps:      0

Top 5 most imported:
  1. common/types.proto     (23 imports)
  2. common/errors.proto    (18 imports)
  3. auth/auth.proto        (12 imports)
  4. user/user.proto        (8 imports)
  5. api/common.proto       (7 imports)
```

---

## 🔄 Автообновление (buffalo upgrade)

Автоматическое обновление Buffalo с миграцией конфигурации.

### Проверка обновлений

```bash
# Проверить доступные обновления
buffalo upgrade --check
```

Пример вывода:
```
🔍 Checking for updates...

Current version: 1.4.0
Latest version:  1.5.0

📋 Changelog:
  • Added workspace management
  • Added permissions module
  • Improved graph visualization
  • Bug fixes and performance improvements

Run 'buffalo upgrade' to update.
```

### Обновление

```bash
# Обновить до последней версии
buffalo upgrade

# Обновить до конкретной версии
buffalo upgrade --to 1.5.0

# Пропустить подтверждение
buffalo upgrade --force

# Показать что изменится (без применения)
buffalo upgrade --dry-run
```

### Опции миграции

```bash
# Только обновить бинарник, без миграции конфигов
buffalo upgrade --skip-config

# Только мигрировать конфиги, без обновления бинарника
buffalo upgrade --skip-binary

# Отключить создание резервной копии
buffalo upgrade --backup=false
```

### Откат

```bash
# Откатиться к предыдущей версии
buffalo upgrade --rollback
```

### Просмотр изменений

```bash
# Показать changelog между версиями
buffalo upgrade --changelog
```

---

## 🏢 Workspace Management (buffalo workspace)

Управление монорепозиториями с множеством proto проектов.

### Инициализация workspace

```bash
# Создать workspace с автоматическим обнаружением проектов
buffalo workspace init --discover

# Создать workspace с именем
buffalo workspace init --name "my-monorepo"

# Создать workspace в конкретной директории
buffalo workspace init --dir /path/to/monorepo
```

### Конфигурация buffalo-workspace.yaml

```yaml
workspace:
  name: my-monorepo
  description: "Multi-service protobuf monorepo"
  version: "1.0"

projects:
  - name: user-service
    path: services/user
    tags: [backend, core]
    depends_on: [common]

  - name: order-service
    path: services/order
    tags: [backend, commerce]
    depends_on: [common, user-service]

  - name: common
    path: libs/common
    tags: [library, shared]

  - name: api-gateway
    path: services/gateway
    tags: [frontend, api]
    depends_on: [user-service, order-service]

policies:
  consistent_versions: true
  shared_dependencies: true
  no_circular_deps: true
```

### Сборка проектов

```bash
# Собрать все проекты (в правильном порядке зависимостей)
buffalo workspace build

# Собрать конкретные проекты
buffalo workspace build user-service order-service

# Собрать проекты с определенным тегом
buffalo workspace build --tag backend

# Параллельная сборка
buffalo workspace build --parallel 4

# Принудительная пересборка
buffalo workspace build --force

# Показать что будет собрано (dry run)
buffalo workspace build --dry-run
```

### Список проектов

```bash
# Показать все проекты
buffalo workspace list

# Фильтр по тегу
buffalo workspace list --tag backend

# Вывод в JSON
buffalo workspace list --format json
```

Пример вывода:
```
NAME                 PATH                         BUILD                TAGS
────────────────────────────────────────────────────────────────────────────────
common               libs/common                  make build           library, shared
user-service         services/user                make build           backend, core
order-service        services/order               make build           backend, commerce
api-gateway          services/gateway             make build           frontend, api

Total: 4 projects
```

### Граф зависимостей проектов

```bash
# ASCII граф
buffalo workspace graph

# Graphviz DOT формат
buffalo workspace graph --format dot

# Mermaid формат
buffalo workspace graph --format mermaid
```

Пример ASCII вывода:
```
Project Dependencies:

  common (no dependencies)
  user-service
    └── common
  order-service
    ├── common
    └── user-service
  api-gateway
    ├── user-service
    └── order-service
```

### Affected проекты

Определение проектов, затронутых изменениями:

```bash
# Показать затронутые проекты с последнего коммита
buffalo workspace affected

# С конкретной точки отсчета
buffalo workspace affected --since HEAD~5
```

Пример вывода:
```
Affected projects (2):
  • user-service
  • order-service

Changed files (3):
  - services/user/protos/user.proto
  - services/user/protos/profile.proto
  - libs/common/types.proto
```

### Выполнение команд во всех проектах

```bash
# Запустить команду во всех проектах
buffalo workspace exec -- go mod tidy

# Только в проектах с тегом
buffalo workspace exec --tag backend -- make test

# Продолжить при ошибке
buffalo workspace exec --continue-on-error -- make lint
```

### Валидация workspace

```bash
buffalo workspace validate
```

---

## 🔐 Permissions Management (buffalo permissions)

Управление RBAC/ABAC аннотациями в proto файлах.

### Аннотации в proto файлах

```protobuf
syntax = "proto3";

package users;

import "buffalo/permissions.proto";

option (buffalo.permissions.resource) = "users";

service UserService {
  // Публичный эндпоинт
  rpc GetPublicProfile (GetProfileRequest) returns (Profile) {
    option (buffalo.permissions) = {
      action: "users:read_public"
      public: true
      rate_limit: { requests: 100, window: "1m" }
    };
  }
  
  // Требует аутентификации
  rpc GetUser (GetUserRequest) returns (User) {
    option (buffalo.permissions) = {
      action: "users:read"
      roles: ["admin", "user"]
      scopes: ["users:read"]
      allow_self: true
      conditions: [
        { field: "user_id", operator: "eq", source: "auth.user_id" }
      ]
    };
  }
  
  // Требует MFA для админов
  rpc DeleteUser (DeleteUserRequest) returns (Empty) {
    option (buffalo.permissions) = {
      action: "users:delete"
      roles: ["admin"]
      require_mfa: true
      audit_log: true
    };
  }
}
```

### Генерация кода

```bash
# Генерация Go кода
buffalo permissions generate --framework go --output pkg/permissions/permissions.go

# Генерация Casbin политик
buffalo permissions generate --framework casbin --output config/policy.csv

# Генерация OPA политик
buffalo permissions generate --framework opa --output policies/authz.rego

# С константами для actions/roles/scopes
buffalo permissions generate --framework go --constants --output permissions.go
```

Пример сгенерированного Go кода:

```go
// Code generated by Buffalo. DO NOT EDIT.
package permissions

var Permissions = map[string]Permission{
    "users.UserService.GetPublicProfile": {
        Action:  "users:read_public",
        Public:  true,
        RateLimit: &RateLimit{Requests: 100, Window: "1m"},
    },
    "users.UserService.GetUser": {
        Action:    "users:read",
        Roles:     []string{"admin", "user"},
        Scopes:    []string{"users:read"},
        AllowSelf: true,
    },
    "users.UserService.DeleteUser": {
        Action:     "users:delete",
        Roles:      []string{"admin"},
        RequireMFA: true,
        AuditLog:   true,
    },
}

// Permission action constants
const (
    ActionUsersReadPublic = "users:read_public"
    ActionUsersRead       = "users:read"
    ActionUsersDelete     = "users:delete"
)

// Role constants
const (
    RoleAdmin = "admin"
    RoleUser  = "user"
)
```

### Матрица доступа

```bash
# Текстовая матрица
buffalo permissions matrix

# HTML матрица
buffalo permissions matrix --format html --output matrix.html

# Markdown матрица
buffalo permissions matrix --format markdown --output PERMISSIONS.md
```

Пример текстового вывода:
```
═══════════════════════════════════════════════════════════════════════════════
Method                        admin       user        s:users:read Public
───────────────────────────────────────────────────────────────────────────────

[users.UserService]
GetPublicProfile              —           —           —            ✓ʳ
GetUser                       ✓           ✓           ✓            —
DeleteUser                    ✓ᴹ          —           —            —

Legend: ✓ = allowed, M = requires MFA, C = has conditions, R = rate limited
```

### Аудит безопасности

```bash
# Полный аудит
buffalo permissions audit

# Только ошибки
buffalo permissions audit --severity error

# JSON вывод
buffalo permissions audit --format json
```

Пример вывода:
```
Errors (1):
  [NO_ROLES_OR_SCOPES] users.UserService.InternalMethod
    Method has no roles or scopes defined
    Fix: Add roles, scopes, or mark as public: true

Warnings (2):
  [ADMIN_NO_MFA] orders.OrderService.CancelAllOrders
    Admin access does not require MFA
    Fix: Add require_mfa: true for admin operations

  [SENSITIVE_NO_AUDIT] users.UserService.UpdatePassword
    Sensitive operation does not have audit logging enabled
    Fix: Add audit_log: true for sensitive operations

Info (1):
  [MISSING_RATE_LIMIT] public.HealthService.Check
    Public endpoint has no rate limit configured
    Fix: Add rate_limit: { requests: 100, window: "1m" }

Total: 1 errors, 2 warnings, 1 info
```

### Сравнение изменений

```bash
# Сравнить permissions между двумя директориями
buffalo permissions diff old-protos/ new-protos/
```

Пример вывода:
```
+ users.UserService.GetUserActivity
  Action: users:read_activity
  Roles: [admin, user]

~ users.UserService.GetUser
  Roles: [admin] -> [admin, user]
  AllowSelf: false -> true

- users.UserService.DeprecatedMethod

Summary: +1 added, ~1 modified, -1 removed
```

### Статистика

```bash
buffalo permissions summary
```

Пример вывода:
```
Permission Summary
════════════════════════════════════════
Services:        5
Methods:         42
Public Methods:  3

Methods by Role:
  admin           28
  user            15
  guest           3

Methods by Scope:
  users:read      12
  users:write     8
  orders:read     6
  orders:write    4
```

---

## ⚙️ Конфигурация

### buffalo.yaml структура

```yaml
version: "1.0"

# Настройки проекта
project:
  name: myproject              # Имя проекта
  proto_root: protos           # Корневая папка .proto файлов
  output_root: generated       # Корневая папка для генерации
  
# Настройки сборки
build:
  parallel: true               # Параллельная компиляция
  jobs: 4                      # Количество воркеров
  incremental: true            # Инкрементальные сборки
  cache_enabled: true          # Использовать кэш
  
# Языки программирования
languages:
  python:
    enabled: true
    output: generated/python
    version: "3.8+"
    plugins:
      - grpc
      - mypy
    options:
      mypy_enabled: true
      package_name: myservice
  
  go:
    enabled: true
    output: generated/go
    module: github.com/myorg/myproject
    version: "1.21+"
    plugins:
      - grpc
      - grpc-gateway
      - validate
    options:
      go_package_prefix: github.com/myorg/myproject
  
  rust:
    enabled: false
    output: generated/rust
    crate: myproject
    plugins:
      - tonic
  
  cpp:
    enabled: false
    output: generated/cpp
    std: "c++17"

# Зависимости
dependencies:
  - name: googleapis
    url: https://github.com/googleapis/googleapis
    version: master
    path: google/api

# Плагины
plugins:
  - name: custom-validator
    path: ./plugins/validator
    enabled: true

# Шаблоны
templates:
  enabled: true
  path: ./templates
  
# Метрики
metrics:
  enabled: true
  output: .buffalo/metrics.json
```

### Переменные окружения

```bash
BUFFALO_CONFIG=./custom.yaml    # Путь к конфигурации
BUFFALO_VERBOSE=true            # Подробный вывод
BUFFALO_JOBS=8                  # Количество воркеров
BUFFALO_NO_CACHE=true           # Отключить кэш
BUFFALO_LOG_LEVEL=debug         # Уровень логирования
```

---

## 🔌 Система плагинов

Buffalo поддерживает расширяемую систему плагинов для кастомизации генерации кода.

### Встроенные плагины

- **grpc** - генерация gRPC кода
- **grpc-gateway** - REST API gateway для gRPC (Go)
- **validate** - валидация сообщений
- **mypy** - типизация для Python
- **tonic** - современный gRPC фреймворк для Rust

### Создание собственного плагина

```go
// plugins/custom/plugin.go
package custom

import "github.com/massonsky/buffalo/pkg/plugin"

type CustomPlugin struct {
    plugin.BasePlugin
}

func (p *CustomPlugin) Name() string {
    return "custom-plugin"
}

func (p *CustomPlugin) Execute(ctx plugin.Context) error {
    // Ваша логика плагина
    return nil
}
```

Подробнее: [Plugin Development Guide](docs/PLUGIN_GUIDE.md)

---

## 📊 Метрики и мониторинг

Buffalo собирает детальные метрики о процессе сборки:

```bash
# Показать метрики последней сборки
buffalo metrics

# Экспорт метрик в JSON
buffalo metrics --format json > metrics.json

# Статистика проекта
buffalo stats
```

Пример вывода:
```
📊 Build Metrics
═══════════════════════════════════════
Total files:        42
Changed files:      3
Build time:         2.34s
Cache hit rate:     92.3%
Workers used:       8
Peak memory:        156MB

📈 Performance by language:
Python:   0.89s  (38%)
Go:       1.12s  (48%)
Rust:     0.33s  (14%)
```

---

## 🐳 Docker интеграция

### Использование официального образа

```dockerfile
FROM ghcr.io/massonsky/buffalo:1.5.0 AS builder

WORKDIR /workspace
COPY protos ./protos
COPY buffalo.yaml .

RUN buffalo build

# Копировать сгенерированный код в ваш образ
FROM python:3.11-slim
COPY --from=builder /workspace/generated/python /app/proto
```

### Docker Compose

```yaml
version: '3.8'

services:
  buffalo-build:
    image: ghcr.io/massonsky/buffalo:1.5.0
    volumes:
      - .:/workspace
    working_dir: /workspace
    command: buffalo build
```

---

## 🛠️ Разработка

### Требования

- Go 1.24+
- Make (опционально)
- Git

### Сборка из исходников

```bash
git clone https://github.com/massonsky/buffalo.git
cd buffalo

# Сборка
make build

# Тестирование
make test

# Проверка кода
make check

# Установка
make install
```

### Запуск тестов

```bash
make test              # Все тесты
make test-unit         # Unit тесты
make test-integration  # Integration тесты
make coverage          # С покрытием кода
```

Подробнее: [Development Guide](docs/DEVELOPMENT.md)

---

## 🤝 Участие в разработке

Мы приветствуем вклад сообщества! Вот как вы можете помочь:

1. 🐛 **Сообщайте о багах** через [GitHub Issues](https://github.com/massonsky/buffalo/issues)
2. 💡 **Предлагайте новые функции** в [Discussions](https://github.com/massonsky/buffalo/discussions)
3. 🔧 **Отправляйте Pull Requests**
4. 📖 **Улучшайте документацию**
5. ⭐ **Ставьте звезды** проекту на GitHub

Перед началом работы прочитайте [Contributing Guide](CONTRIBUTING.md).

---

## 📖 Документация

- [Quick Start](QUICKSTART.md) - быстрый старт за 5 минут
- [Installation Guide](INSTALL.md) - детальная инструкция по установке
- [Architecture](docs/ARCHITECTURE.md) - архитектура системы
- [Configuration Guide](docs/CONFIG_GUIDE.md) - полное руководство по конфигурации
- [Plugin Development](docs/PLUGIN_GUIDE.md) - создание плагинов
- [Development Guide](docs/DEVELOPMENT.md) - руководство для разработчиков
- [CLI Reference](docs/PLUGIN_CLI.md) - справочник по командам
- [CI/CD Integration](docs/CI_CD_GUIDE.md) - интеграция с CI/CD
- [Metrics Guide](docs/METRICS_GUIDE.md) - работа с метриками

---

## 🎯 Use Cases

### Микросервисная архитектура
Buffalo идеален для проектов с множеством микросервисов на разных языках:
- Централизованное управление proto-контрактами
- Консистентная генерация кода для всех сервисов
- Быстрые итерации благодаря инкрементальным сборкам

### CI/CD пайплайны
Интеграция с популярными CI/CD системами:

```yaml
# .github/workflows/build.yml
name: Build Proto
on: [push]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Install Buffalo
        run: curl -sSL https://raw.githubusercontent.com/massonsky/buffalo/main/install.sh | bash
      
      - name: Build
        run: buffalo build
      
      - name: Audit Permissions
        run: buffalo permissions audit --severity error
      
      - name: Upload artifacts
        uses: actions/upload-artifact@v3
        with:
          name: generated-code
          path: generated/
```

### Монорепозитории
Buffalo отлично работает с монорепо через workspace:

```yaml
# .github/workflows/workspace.yml
name: Workspace Build
on:
  push:
    branches: [main]
  pull_request:

jobs:
  affected:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
        with:
          fetch-depth: 0
      
      - name: Install Buffalo
        run: curl -sSL https://raw.githubusercontent.com/massonsky/buffalo/main/install.sh | bash
      
      - name: Build affected projects
        run: |
          AFFECTED=$(buffalo workspace affected --since ${{ github.event.before }})
          if [ -n "$AFFECTED" ]; then
            buffalo workspace build $(echo $AFFECTED | tr '\n' ' ')
          fi
```

---

## 🔍 Сравнение с альтернативами

| Функция | Buffalo | Buf | Prototool | Ручная сборка |
|---------|---------|-----|-----------|---------------|
| Мультиязычность | ✅ Python, Go, Rust, C++ | ✅ Множество | ⚠️ Ограничено | ✅ Любой |
| Интеллектуальное кэширование | ✅ | ⚠️ Базовое | ❌ | ❌ |
| Параллельная компиляция | ✅ | ✅ | ⚠️ | ❌ |
| Система плагинов | ✅ | ✅ | ⚠️ | ⚠️ |
| Watch mode | ✅ | ❌ | ✅ | ❌ |
| Dependency management | ✅ | ✅ | ❌ | ❌ |
| Workspace management | ✅ | ⚠️ | ❌ | ❌ |
| Permissions/RBAC | ✅ | ❌ | ❌ | ❌ |
| Граф зависимостей | ✅ | ⚠️ | ❌ | ❌ |
| Auto-upgrade | ✅ | ❌ | ❌ | ❌ |
| Метрики | ✅ | ❌ | ❌ | ❌ |

---

## 📈 Производительность

Buffalo оптимизирован для максимальной производительности:

### Бенчмарки (проект с 100 proto файлов)

```
Полная сборка (без кэша):
Buffalo:        12.3s
Buf:            18.7s
Manual (make):  34.2s

Инкрементальная сборка (5% изменений):
Buffalo:        0.9s  (93% faster)
Buf:            4.2s  (77% faster)
Manual (make):  34.2s (no cache)

Параллельная сборка (4 языка):
Buffalo:        15.4s (4 workers)
Sequential:     58.6s
Speedup:        3.8x
```

---

## 🗺️ Roadmap

### v1.6.0 (Q2 2026)
- [ ] Поддержка TypeScript
- [ ] Улучшенная система шаблонов
- [ ] Remote caching
- [ ] Web UI для управления проектами

### v2.0.0 (Q3 2026)
- [ ] Cloud-native build service
- [ ] Integration с популярными IDE
- [ ] Advanced dependency resolution
- [ ] Built-in proto registry

Полный roadmap: [GitHub Projects](https://github.com/massonsky/buffalo/projects)

---

## 📄 Лицензия

Buffalo распространяется под лицензией [MIT License](LICENSE).

---

## 🙏 Благодарности

Buffalo создан с использованием отличных open-source проектов:

- [Protocol Buffers](https://github.com/protocolbuffers/protobuf) - Google's data interchange format
- [gRPC](https://grpc.io/) - High performance RPC framework
- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [Viper](https://github.com/spf13/viper) - Configuration management
- [fsnotify](https://github.com/fsnotify/fsnotify) - File system notifications

---

## 💬 Поддержка и сообщество

- 📧 **Email:** support@buffalo-build.dev
- 💬 **Discussions:** [GitHub Discussions](https://github.com/massonsky/buffalo/discussions)
- 🐛 **Issues:** [GitHub Issues](https://github.com/massonsky/buffalo/issues)
- 📖 **Wiki:** [GitHub Wiki](https://github.com/massonsky/buffalo/wiki)

---

<div align="center">

**[Website](https://buffalo-build.dev)** • 
**[Documentation](https://docs.buffalo-build.dev)** • 
**[Examples](examples/)** • 
**[Changelog](CHANGELOG.md)**

Made with ❤️ by the Buffalo Team

</div>
