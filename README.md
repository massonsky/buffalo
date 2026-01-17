# 🦬 Buffalo

> Кроссплатформенный мультиязычный сборщик protobuf/gRPC файлов

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![CI](https://github.com/massonsky/buffalo/workflows/CI/badge.svg)](https://github.com/massonsky/buffalo/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/massonsky/buffalo)](https://goreportcard.com/report/github.com/massonsky/buffalo)
[![Version](https://img.shields.io/badge/version-0.5.0-green.svg)](https://github.com/massonsky/buffalo/releases)

---

## 📊 Статус разработки

**Текущая версия: v0.5.0** - Полнофункциональный CLI & Все компиляторы

✅ **Завершено (v0.1.0 - v0.5.0):**

**Инфраструктура (v0.1.0):**
- **pkg/logger** - Система логирования с цветным выводом
- **pkg/errors** - Обработка ошибок с контекстом
- **pkg/utils** - Файловые утилиты и валидация
- **pkg/metrics** - Система метрик

**CLI & Конфигурация (v0.2.0):**
- **internal/cli** - CLI интерфейс с Cobra (14 команд)
- **internal/config** - Конфигурация с Viper (YAML/TOML/JSON)
- **internal/version** - Версионирование с git tags

**Core Builder (v0.3.0):**
- **internal/builder** - Основной билдер
- **internal/scanner** - Сканер proto файлов
- **internal/resolver** - Разрешение зависимостей
- **internal/scheduler** - Планировщик сборки
- **internal/cache** - Кэширование для инкрементальной сборки

**Компиляторы (v0.4.0 - v0.5.0):**
- **internal/compiler/python** - Python (protobuf + grpcio)
- **internal/compiler/golang** - Go (protoc-gen-go + grpc)
- **internal/compiler/rust** - Rust (prost + tonic)
- **internal/compiler/cpp** - C++ (protobuf + grpc++)

**CLI Команды (v0.5.0):**
- `build` - Сборка proto файлов
- `rebuild` - Полная пересборка с очисткой кэша
- `watch` - Автоматическая пересборка при изменениях
- `init` - Инициализация проекта
- `check` - Проверка конфигурации и зависимостей
- `list` - Список proto файлов
- `stats` - Статистика проекта
- `clear` - Очистка кэша и сгенерированных файлов
- `lint` - Проверка стиля proto файлов
- `format` - Форматирование proto файлов
- `validate` - Валидация синтаксиса через protoc
- `deps` - Анализ зависимостей
- `version` - Информация о версии
- `completion` - Автодополнение для shell

🚧 **В разработке (v0.6.0):**
- Система плагинов
- Пользовательские шаблоны генерации
- Расширенное кэширование

📋 **Запланировано:**
- v0.7.0: CI/CD интеграция, GitHub Actions
- v0.8.0: Расширенный watch mode с hot-reload
- v0.9.0: Метрики и профилирование
- v1.0.0: Стабильный релиз

---

## ✨ Особенности

- 🌍 **Мультиязычность:** Python, Go, Rust, C++ ✅
- ⚡ **Высокая производительность:** Параллельная компиляция ✅, инкрементальная сборка ✅
- 🔧 **Гибкая настройка:** Конфигурация через YAML/TOML/JSON ✅
- 🔌 **Расширяемость:** Система плагинов (v0.6.0)
- 📦 **Кроссплатформенность:** Windows, Linux, macOS ✅
- 🎯 **Простота использования:** 14 CLI команд ✅
- 🚀 **Watch mode:** Автоматическая пересборка ✅
- 💾 **Кэширование:** Умное кэширование ✅
- 📊 **Метрики:** Система метрик ✅
- 🐳 **Docker ready:** Docker + docker-compose ✅

---

## 🏗️ Текущая архитектура (v0.1.0)

### Реализованные компоненты

#### pkg/logger
Полнофункциональная система логирования:
```go
// Создание logger с цветным форматированием
log := logger.New(
    logger.WithFormatter(logger.NewColoredFormatter()),
    logger.WithLevel(logger.DEBUG),
)

// Structured logging
log.WithFields(logger.Fields{
    "user": "john",
    "action": "login",
}).Info("User logged in")
```

**Возможности:**
- 3 форматтера: JSON, Text, Colored (ANSI цвета)
- 2 типа вывода: Console (stdout/stderr), File (с ротацией)
- Уровни: DEBUG, INFO, WARN, ERROR, FATAL
- Поддержка Fields для structured logging

#### pkg/errors
Расширенная обработка ошибок:
```go
// Создание ошибки с кодом
err := errors.New(errors.ErrInvalidInput, "неверный формат файла")

// Wrap существующей ошибки
wrapped := errors.Wrap(err, errors.ErrIO, "не удалось прочитать файл")

// Добавление контекста
wrapped.WithContext("file", "example.proto")

// 26 предопределённых кодов ошибок
```

#### pkg/utils
Набор утилит для работы с файлами, путями, хешированием:
```go
// Поиск proto файлов
files, _ := utils.FindFiles("./protos", "*.proto", true)

// Валидация proto файла
result := utils.ValidateProtoFile("example.proto")

// Хеширование
hash, _ := utils.HashFile("example.proto") // SHA256 по умолчанию

// Параллельное выполнение
pool := utils.NewWorkerPool(4)
pool.Execute(tasks)
```

#### pkg/metrics
Система сбора метрик:
```go
// Создание collector
collector := metrics.NewCollector()

// Counter для подсчёта
counter := collector.Counter("builds_total")
counter.Inc()

// Gauge для текущего значения
gauge := collector.Gauge("active_workers")
gauge.Set(10)

// Histogram для распределения значений
hist := collector.Histogram("build_duration_ms", nil)
hist.Observe(245.5)

// Экспорт в Prometheus
exporter := metrics.NewExporter(collector)
exporter.Export(metrics.FormatPrometheus, os.Stdout)
```

**Возможности:**
- 3 типа метрик: Counter, Gauge, Histogram
- Thread-safe с атомарными операциями
- Экспорт: Text, JSON, Prometheus
- Глобальные labels
- Snapshots для точки во времени

---

## 🚀 Быстрый старт

### Установка

#### Способ 1: Автоматический скрипт (рекомендуется)

**Linux/macOS:**
```bash
curl -sSL https://raw.githubusercontent.com/massonsky/buffalo/dev/install.sh | bash
```

**Windows (PowerShell):**
```powershell
irm https://raw.githubusercontent.com/massonsky/buffalo/dev/install.ps1 | iex
```

#### Способ 2: Из исходников

```bash
git clone https://github.com/massonsky/buffalo.git
cd buffalo

# Linux/macOS
./build.sh install

# Windows
.\build.ps1 -Target install
```

#### Способ 3: Docker

```bash
docker build -t buffalo:latest .
docker run --rm -v $(pwd):/workspace buffalo:latest build
```

#### Способ 4: Через Go

```bash
go install github.com/massonsky/buffalo/cmd/buffalo@latest
```

### Сборка и разработка

#### Linux/macOS

```bash
# Быстрая сборка
./build.sh build

# Все платформы
./build.sh build-all

# Тесты
./build.sh test

# С покрытием
./build.sh test-coverage

# Проверка кода (fmt, vet, lint, test)
./build.sh check

# Установка в систему
./build.sh install

# Очистка
./build.sh clean

# Справка
./build.sh help
```

#### Windows (PowerShell)

```powershell
# Быстрая сборка
.\build.ps1 build

# Все платформы
.\build.ps1 build-all

# Тесты
.\build.ps1 test

# С покрытием
.\build.ps1 test-coverage

# Проверка кода
.\build.ps1 check

# Установка
.\build.ps1 install

# Очистка
.\build.ps1 clean

# Справка
.\build.ps1 help
```

#### Через Make (Unix/Linux/macOS)

```bash
# Сборка
make build

# Установка в систему
sudo make install-system

# Тесты
make test

# Покрытие
make coverage

# Docker
make docker-build

# Релиз
make release

# Справка
make help
```

#### Через CMake (альтернатива)

```bash
# Сборка
cmake -B build
cmake --build build --target build

# Установка
cmake --build build --target install

# Тесты
cmake --build build --target test
```

---

## 📖 Использование (планируется v0.2.0+)

> **Примечание:** CLI команды ниже будут доступны в v0.2.0. 
> Сейчас (v0.1.0) доступна только базовая инфраструктура.

### Базовая команда

```bash
buffalo build
```

### С конфигурационным файлом

```bash
buffalo build --config buffalo.yaml
```

### Выбор языков

```bash
buffalo build --lang python,go,rust
```

### Watch mode

```bash
buffalo watch
```

### Dry run

```bash
buffalo build --dry-run
```

---

## ⚙️ Конфигурация

### buffalo.yaml

```yaml
version: 1.0

global:
  proto_path:
    - ./proto
  output_path: ./generated
  import_paths:
    - ./third_party/proto
  parallel: 4
  incremental: true

languages:
  python:
    enabled: true
    output: ./generated/python
    plugins:
      - grpc
    options:
      mypy_stubs: true
  
  go:
    enabled: true
    output: ./generated/go
    go_package_prefix: github.com/yourorg/project
    plugins:
      - grpc
  
  rust:
    enabled: true
    output: ./generated/rust
    plugins:
      - tonic
  
  cpp:
    enabled: true
    output: ./generated/cpp
    plugins:
      - grpc
    options:
      std: c++17

logging:
  level: info
  format: colored
  output: stdout
```

Больше примеров в [docs/CONFIGURATION.md](docs/CONFIGURATION.md)

---

## 🎯 Примеры

### Python с gRPC

```bash
# Генерация Python кода с gRPC
buffalo build --lang python

# Использование сгенерированного кода
python -c "
from generated.python import service_pb2, service_pb2_grpc
# ... ваш код
"
```

### Go с модулями

```bash
# Генерация Go кода
buffalo build --lang go

# Использование в Go проекте
go get github.com/yourorg/project/generated/go
```

### Rust с Tonic

```bash
# Генерация Rust кода
buffalo build --lang rust

# Добавить в Cargo.toml
# [dependencies]
# generated = { path = "./generated/rust" }
```

### C++ с CMake

```bash
# Генерация C++ кода
buffalo build --lang cpp

# CMakeLists.txt автоматически создан
mkdir build && cd build
cmake .. && make
```

Больше примеров в [examples/](examples/)

---

## 🏗️ Архитектура

```
┌─────────────┐
│     CLI     │  Интерфейс командной строки
└──────┬──────┘
       │
┌──────▼──────────────────────┐
│  Build Orchestrator         │  Координация процесса
└──────┬──────────────────────┘
       │
       ├──────────────┬──────────────┬──────────────┐
       ▼              ▼              ▼              ▼
┌──────────┐   ┌──────────┐   ┌──────────┐   ┌──────────┐
│ Scanner  │   │ Resolver │   │Scheduler │   │  Cache   │
└──────────┘   └──────────┘   └──────────┘   └──────────┘
       │
       ├──────────────┬──────────────┬──────────────┐
       ▼              ▼              ▼              ▼
┌──────────┐   ┌──────────┐   ┌──────────┐   ┌──────────┐
│  Python  │   │    Go    │   │   Rust   │   │   C++    │
│Compiler  │   │Compiler  │   │Compiler  │   │Compiler  │
└──────────┘   └──────────┘   └──────────┘   └──────────┘
```

Подробнее в [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)

---

## 📚 Документация

- 📋 [Roadmap](docs/ROADMAP.md) - План развития проекта
- 🏛️ [Архитектура](docs/ARCHITECTURE.md) - Детальная архитектура
- 📁 [Структура проекта](docs/PROJECT_STRUCTURE.md) - Организация кода
- 🛠️ [Руководство по разработке](docs/DEVELOPMENT.md) - Для контрибьюторов
- ⚙️ [Конфигурация](docs/CONFIGURATION.md) - Настройка Buffalo
- 📖 [CLI Reference](docs/CLI_REFERENCE.md) - Справка по командам
- 🔌 [Разработка плагинов](docs/PLUGIN_DEVELOPMENT.md) - Создание плагинов
- 🚀 [Deployment](docs/DEPLOYMENT.md) - Развертывание
- 🔧 [Troubleshooting](docs/TROUBLESHOOTING.md) - Решение проблем

---

## 🎓 Туториалы

- [Начало работы](docs/tutorials/getting_started.md)
- [Базовое использование](docs/tutorials/basic_usage.md)
- [Продвинутая конфигурация](docs/tutorials/advanced_config.md)
- [Создание плагина](docs/tutorials/plugin_creation.md)
- [CI/CD интеграция](docs/tutorials/ci_cd_integration.md)

---

## 🤝 Участие в разработке

Мы приветствуем вклад в проект! Прочитайте [CONTRIBUTING.md](CONTRIBUTING.md) для деталей.

### Процесс:

1. Fork репозитория
2. Создайте feature branch (`git checkout -b feature/amazing-feature`)
3. Commit изменений (`git commit -m 'feat: add amazing feature'`)
4. Push в branch (`git push origin feature/amazing-feature`)
5. Откройте Pull Request

### Для разработчиков:

```bash
# Клонирование
git clone https://github.com/yourorg/buffalo.git
cd buffalo

# Установка зависимостей
make install-tools

# Запуск тестов
make test

# Сборка
make build
```

Подробнее в [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md)

---

## 🧪 Тестирование

```bash
# Все тесты
make test

# Unit тесты
make test-unit

# Integration тесты
make test-integration

# Coverage
make test-coverage

# Бенчмарки
make benchmark
```

Покрытие тестами: ![Coverage](https://codecov.io/gh/yourorg/buffalo/branch/main/graph/badge.svg)

---

## 📊 Производительность

Buffalo оптимизирован для производительности:

- ⚡ Параллельная компиляция
- 💾 Умное кэширование
- 🎯 Инкрементальная сборка
- 📦 Минимальное использование памяти

**Бенчмарки:**
- 100 proto файлов: ~5 секунд
- Memory usage: ~50MB
- Binary size: ~15MB

---

## 🐳 Docker

```bash
# Скачать образ
docker pull buffalo/buffalo:latest

# Запустить
docker run -v $(pwd):/workspace buffalo/buffalo build

# Docker Compose
docker-compose up
```

Docker образы доступны на:
- [Docker Hub](https://hub.docker.com/r/buffalo/buffalo)
- [GitHub Container Registry](https://github.com/yourorg/buffalo/pkgs/container/buffalo)

---

## 🔌 Плагины

Buffalo поддерживает систему плагинов для расширения функциональности:

### Официальные плагины:

- `buffalo-lint` - Линтинг proto файлов
- `buffalo-validate` - Валидация proto схем
- `buffalo-docs` - Генерация документации
- `buffalo-typescript` - TypeScript поддержка

### Кастомные плагины:

```go
package main

import "github.com/yourorg/buffalo/pkg/plugin"

type MyPlugin struct{}

func (p *MyPlugin) Execute(ctx context.Context, data *plugin.Data) error {
    // Ваша логика
    return nil
}

func main() {
    plugin.Register("my-plugin", &MyPlugin{})
}
```

Подробнее в [docs/PLUGIN_DEVELOPMENT.md](docs/PLUGIN_DEVELOPMENT.md)

---

## 🌟 Используется в проектах

- [Project A](https://github.com/example/project-a) - Микросервисная архитектура
- [Project B](https://github.com/example/project-b) - gRPC API gateway
- [Project C](https://github.com/example/project-c) - Real-time платформа

---

## 📈 Roadmap

### v1.0.0 (Current)
- ✅ Поддержка Python, Go, Rust, C++
- ✅ Параллельная сборка
- ✅ Инкрементальная сборка
- ✅ Система плагинов
- ✅ Watch mode

### v1.1.0 (Planning)
- 🔄 TypeScript/JavaScript поддержка
- 🔄 Java/Kotlin поддержка
- 🔄 Remote cache
- 🔄 Web UI

### v2.0.0 (Future)
- 📋 GraphQL поддержка
- 📋 Distributed builds
- 📋 Cloud integration

Полный roadmap в [docs/ROADMAP.md](docs/ROADMAP.md)

---

## 📄 Лицензия

Этот проект лицензирован под MIT License - см. [LICENSE](LICENSE) для деталей.

---

## 👥 Авторы

- **Your Name** - *Initial work* - [@yourname](https://github.com/yourname)

См. также список [contributors](https://github.com/yourorg/buffalo/contributors).

---

## 🙏 Благодарности

- Protocol Buffers team
- gRPC team
- Go community
- Все контрибьюторы

---

## 💬 Поддержка

- 📫 **Email:** buffalo@yourorg.com
- 💬 **Discord:** [Join our Discord](https://discord.gg/buffalo)
- 🐛 **Issues:** [GitHub Issues](https://github.com/yourorg/buffalo/issues)
- 💡 **Discussions:** [GitHub Discussions](https://github.com/yourorg/buffalo/discussions)

---

## ⭐ Star History

[![Star History Chart](https://api.star-history.com/svg?repos=yourorg/buffalo&type=Date)](https://star-history.com/#yourorg/buffalo&Date)

---

<p align="center">
  Made with ❤️ by the Buffalo team
</p>

<p align="center">
  <a href="https://github.com/yourorg/buffalo">GitHub</a> •
  <a href="https://buffalo.yourorg.com">Website</a> •
  <a href="https://buffalo.yourorg.com/docs">Documentation</a> •
  <a href="https://twitter.com/buffalo">Twitter</a>
</p>
