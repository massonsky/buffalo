# 🦬 Buffalo - Multi-Language Protobuf/gRPC Build System

[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.24+-00ADD8.svg)](https://go.dev/)
[![Version](https://img.shields.io/badge/version-1.0.0-green.svg)](https://github.com/massonsky/buffalo/releases)

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

### 🔧 Удобство
- **CLI интерфейс** - интуитивная командная строка
- **Watch mode** - автоматическая пересборка при изменениях
- **Метрики и статистика** - отслеживание производительности
- **Docker support** - контейнеризированные сборки

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
```

#### Docker
```bash
docker pull ghcr.io/massonsky/buffalo:1.0.0
docker run --rm ghcr.io/massonsky/buffalo:1.0.0 buffalo --version
```

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
buffalo check                    # Проверка конфигурации и .proto файлов
buffalo validate                 # Валидация proto файлов
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
FROM ghcr.io/massonsky/buffalo:1.0.0 AS builder

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
    image: ghcr.io/massonsky/buffalo:1.0.0
    volumes:
      - .:/workspace
    working_dir: /workspace
    command: buffalo build
```

---

## 🛠️ Разработка

### Требования

- Go 1.21+
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
      - name: Upload artifacts
        uses: actions/upload-artifact@v3
        with:
          name: generated-code
          path: generated/
```

### Монорепозитории
Buffalo отлично работает с монорепо:
- Одна конфигурация для всего репозитория
- Параллельная сборка независимых модулей
- Эффективное кэширование

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
| Шаблонизация | ✅ | ⚠️ | ❌ | ✅ |
| Метрики | ✅ | ❌ | ❌ | ❌ |
| Простота настройки | ✅ | ✅ | ✅ | ❌ |

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

## 🐛 Известные проблемы

Актуальный список проблем: [GitHub Issues](https://github.com/massonsky/buffalo/issues)

### Обходные пути

#### Проблема с Windows path
```yaml
# Используйте forward slashes в путях
project:
  proto_root: protos  # ✅
  # proto_root: protos\api  # ❌
```

#### Медленная первая сборка
Первая сборка может быть медленной из-за загрузки зависимостей. Последующие сборки будут быстрыми.

---

## 🗺️ Roadmap

### v1.1.0 (Q2 2026)
- [ ] Поддержка TypeScript
- [ ] Улучшенная система шаблонов
- [ ] Remote caching
- [ ] Web UI для управления проектами

### v1.2.0 (Q3 2026)
- [ ] Поддержка Swift и Kotlin
- [ ] Distributed builds
- [ ] Advanced metrics и dashboards
- [ ] AI-powered code optimization

### v2.0.0 (Q4 2026)
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

## 📊 Статистика проекта

![GitHub stars](https://img.shields.io/github/stars/massonsky/buffalo?style=social)
![GitHub forks](https://img.shields.io/github/forks/massonsky/buffalo?style=social)
![GitHub watchers](https://img.shields.io/github/watchers/massonsky/buffalo?style=social)

---

<div align="center">

**[Website](https://buffalo-build.dev)** • 
**[Documentation](https://docs.buffalo-build.dev)** • 
**[Examples](examples/)** • 
**[Changelog](CHANGELOG.md)**

Made with ❤️ by the Buffalo Team

</div>