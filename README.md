# 🦬 Buffalo

> Кроссплатформенный мультиязычный сборщик protobuf/gRPC файлов

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![CI](https://github.com/yourorg/buffalo/workflows/CI/badge.svg)](https://github.com/yourorg/buffalo/actions)
[![Coverage](https://codecov.io/gh/yourorg/buffalo/branch/main/graph/badge.svg)](https://codecov.io/gh/yourorg/buffalo)
[![Go Report Card](https://goreportcard.com/badge/github.com/yourorg/buffalo)](https://goreportcard.com/report/github.com/yourorg/buffalo)

---

## ✨ Особенности

- 🌍 **Мультиязычность:** Python, Go, Rust, C++ из коробки
- ⚡ **Высокая производительность:** Параллельная компиляция, инкрементальная сборка
- 🔧 **Гибкая настройка:** Конфигурация через YAML/TOML/JSON, CLI флаги, переменные окружения
- 🔌 **Расширяемость:** Система плагинов для кастомизации
- 📦 **Кроссплатформенность:** Windows, Linux, macOS
- 🎯 **Простота использования:** Интуитивный CLI интерфейс
- 🚀 **Watch mode:** Автоматическая пересборка при изменениях
- 💾 **Кэширование:** Умное кэширование для быстрой пересборки
- 📊 **Метрики:** Детальная статистика сборки
- 🐳 **Docker ready:** Готовые Docker образы

---

## 🚀 Быстрый старт

### Установка

#### Homebrew (macOS/Linux)
```bash
brew install buffalo
```

#### Scoop (Windows)
```bash
scoop install buffalo
```

#### Go install
```bash
go install github.com/yourorg/buffalo/cmd/buffalo@latest
```

#### Binary releases
Скачать с [GitHub Releases](https://github.com/yourorg/buffalo/releases)

### Первая сборка

```bash
# Инициализация проекта
buffalo init

# Создать buffalo.yaml конфиг (или редактировать существующий)
# Запустить сборку
buffalo build

# Готово! Сгенерированный код в ./generated
```

---

## 📖 Использование

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
