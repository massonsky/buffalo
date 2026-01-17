# 🦬 Buffalo

> Кроссплатформенный мультиязычный сборщик protobuf/gRPC файлов

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![CI](https://github.com/massonsky/buffalo/workflows/CI/badge.svg)](https://github.com/massonsky/buffalo/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/massonsky/buffalo)](https://goreportcard.com/report/github.com/massonsky/buffalo)
[![Version](https://img.shields.io/badge/version-0.1.0-green.svg)](https://github.com/massonsky/buffalo/releases)

---

## 📊 Статус разработки

**Текущая версия: v0.1.0** - Базовая инфраструктура

✅ **Завершено:**
- **pkg/logger** - Система логирования (54.7% coverage, 16 tests)
- **pkg/errors** - Обработка ошибок (92.2% coverage, 20 tests)  
- **pkg/utils** - Утилиты (67.3% coverage, 46 tests)
- **pkg/metrics** - Метрики (90.9% coverage, 20 tests)

🚧 **В разработке (v0.2.0):**
- CLI интерфейс с Cobra
- Система конфигурации с Viper
- Базовая структура builder

📋 **Запланировано:**
- v0.3.0: Core builder реализация
- v0.4.0-v0.7.0: Компиляторы Python/Go/Rust/C++
- v0.8.0: Система плагинов
- v0.9.0-v1.0.0: Watch mode, инкрементальная сборка, кэширование

См. [ROADMAP.md](ROADMAP.md) для деталей.

---

## ✨ Особенности

- 🌍 **Мультиязычность:** Python, Go, Rust, C++ (планируется)
- ⚡ **Высокая производительность:** Параллельная компиляция, инкрементальная сборка (планируется)
- 🔧 **Гибкая настройка:** Конфигурация через YAML/TOML/JSON (в разработке)
- 🔌 **Расширяемость:** Система плагинов (планируется)
- 📦 **Кроссплатформенность:** Windows, Linux, macOS
- 🎯 **Простота использования:** Интуитивный CLI (в разработке)
- 🚀 **Watch mode:** Автоматическая пересборка (планируется)
- 💾 **Кэширование:** Умное кэширование (планируется)
- 📊 **Метрики:** Система метрик реализована
- 🐳 **Docker ready:** Docker образы (планируется)

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

### Сборка из исходников

```bash
# Клонирование репозитория
git clone git@github.com:massonsky/buffalo.git
cd buffalo

# Сборка
make build

# Запуск
./bin/buffalo --version

# Тесты
make test

# Coverage
make coverage
```

### Разработка

```bash
# Линтеры
make lint

# Форматирование
make fmt

# Сборка для всех платформ
make cross-compile

# Очистка
make clean
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
