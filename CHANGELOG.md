# Changelog

Все заметные изменения в проекте Buffalo будут документироваться в этом файле.

Формат основан на [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
и проект следует [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Планируется
- CLI интерфейс с Cobra (v0.2.0)
- Система конфигурации с Viper (v0.2.0)
- Core builder реализация (v0.3.0)
- Компиляторы для языков (v0.4.0-v0.7.0)
- Система плагинов (v0.8.0)
- Расширенные возможности (v0.9.0-v1.0.0)

---

## [0.1.0] - 2026-01-17

### Added
- **pkg/logger** - Полная система логирования
  - Structured logging с Fields поддержкой
  - 3 форматтера: JSON, Text, Colored (ANSI цвета)
  - 2 типа вывода: Console (stdout/stderr), File (ротация по размеру/возрасту/дням)
  - Уровни: DEBUG, INFO, WARN, ERROR, FATAL
  - 16 тестов, 54.7% покрытие

- **pkg/errors** - Расширенная обработка ошибок
  - Error type с Code/Message/Cause/Stack/Context
  - 26 предопределённых кодов ошибок
  - New/Wrap/Unwrap для цепочек ошибок
  - Совместимость с errors.Is/As
  - 20 тестов, 92.2% покрытие

- **pkg/utils** - Набор утилит
  - file.go: FindFiles, CopyFile, EnsureDir, CleanDir, ReadFile, WriteFile
  - path.go: NormalizePath, JoinPath, ChangeExtension, кроссплатформенность
  - hash.go: MD5, SHA1, SHA256, SHA512 для файлов/строк/данных
  - validation.go: ValidateProtoFile, ValidateProtoPackageName, ValidateOutputDir
  - worker.go: WorkerPool для параллельного выполнения задач
  - 46 тестов, 67.3% покрытие

- **pkg/metrics** - Система метрик
  - Metric types: Counter (Inc/Add), Gauge (Set/Inc/Dec/Add/Sub), Histogram (Observe с buckets)
  - Collector: Register/Get/GetOrCreate, labels, snapshots
  - Exporter: Text, JSON, Prometheus форматы
  - Thread-safe с атомарными операциями
  - Интеграция с pkg/logger
  - 20 тестов, 90.9% покрытие

### Changed
- Обновлён cmd/buffalo/main.go для использования ColoredFormatter
- Добавлены example_test.go для всех компонентов
- README.md обновлён с информацией о v0.1.0

### Technical Details
- 19 исходных файлов в pkg/
- 102 теста, все проходят успешно
- Средний coverage: 76.3%
- Нет внешних зависимостей (только stdlib)
- Кроссплатформенная поддержка (Windows/Linux/macOS)
- Atomic operations для thread-safety

---

## [0.0.0] - 2026-01-17

### Added
- 📋 Полная документация проекта
  - ROADMAP.md: План развития v0.0.0 → v1.0.0
  - ARCHITECTURE.md: Clean Architecture описание
  - PROJECT_STRUCTURE.md: Структура директорий
  - DEVELOPMENT.md: Руководство для разработчиков
  - CONTRIBUTING.md: Правила контрибуции
  - CODE_OF_CONDUCT.md: Кодекс поведения
  - SECURITY.md: Политика безопасности
  - TESTING.md: Стратегия тестирования

- 🏗️ Базовая структура проекта
  - Инициализация Go модуля (github.com/massonsky/buffalo)
  - 28 директорий для всех компонентов
  - internal/version для версионирования (Version/Commit/BuildDate/Platform)
  - cmd/buffalo/main.go - точка входа

- 🔧 Build инфраструктура
  - Makefile с 20+ таргетами (build, test, lint, coverage, install, clean, cross-compile)
  - .github/workflows/ci.yml - CI/CD pipeline (lint, test, build на Ubuntu/Windows/macOS)
  - bin/ для собранных бинарников
  - build/ для build артефактов

- 📦 Git конфигурация
  - .gitignore для Go проектов
  - .gitattributes для line endings
  - LICENSE (MIT)
  - README.md с badges и описанием

### Technical Details
- Go 1.21+ требуется
- Поддержка кросс-компиляции (GOOS/GOARCH)
- Version info встроен в бинарник через ldflags
- Готовность к TDD разработке

---

## [1.0.0] - TBD

### Added
- ✨ Полная поддержка Python компиляции
- ✨ Полная поддержка Go компиляции
- ✨ Полная поддержка Rust компиляции
- ✨ Полная поддержка C++ компиляции
- ✨ Система плагинов
- ✨ Watch mode для автоматической пересборки
- ✨ Инкрементальная сборка с кэшированием
- ✨ Параллельная компиляция
- ✨ Гибкая система конфигурации (YAML/TOML/JSON)
- ✨ CLI интерфейс с полным набором команд
- ✨ Docker образы
- ✨ CI/CD интеграции (GitHub Actions, GitLab CI)
- ✨ Shell completions (bash, zsh, fish)
- ✨ Детальная метрика сборки
- ✨ Dry-run режим
- ✨ Diff режим
- 📚 Полная документация
- 📚 Примеры для всех поддерживаемых языков
- 🧪 Покрытие тестами >85%

### Changed
- N/A (первый релиз)

### Deprecated
- N/A (первый релиз)

### Removed
- N/A (первый релиз)

### Fixed
- N/A (первый релиз)

### Security
- 🔒 Валидация путей для предотвращения path traversal
- 🔒 Безопасное выполнение команд
- 🔒 Sandbox для плагинов

---

## [0.9.0] - TBD (Beta)

### Added
- ✨ Watch mode
- ✨ Dry-run режим
- ✨ Diff режим
- ✨ Metrics и статистика
- 🐳 Docker образ
- 🔧 Shell completions
- 🚀 CI/CD интеграции

### Changed
- ⚡ Оптимизация производительности
- 📖 Улучшена документация

### Fixed
- 🐛 Исправления багов из beta тестирования

---

## [0.8.0] - TBD

### Added
- ✨ Система плагинов
- ✨ Plugin API/SDK
- ✨ Примеры плагинов
- 📚 Документация по разработке плагинов

### Changed
- 🔧 Рефакторинг архитектуры для плагинов

---

## [0.7.0] - TBD

### Added
- ✨ C++ компилятор
- ✨ Поддержка CMake
- ✨ pkg-config генерация
- 📚 C++ примеры

### Fixed
- 🐛 Исправления в предыдущих компиляторах

---

## [0.6.0] - TBD

### Added
- ✨ Rust компилятор
- ✨ Поддержка prost
- ✨ Поддержка tonic
- ✨ Cargo.toml генерация
- 📚 Rust примеры

---

## [0.5.0] - TBD

### Added
- ✨ Go компилятор
- ✨ Поддержка protoc-gen-go
- ✨ Поддержка protoc-gen-go-grpc
- ✨ go.mod интеграция
- 📚 Go примеры

---

## [0.4.0] - TBD

### Added
- ✨ Python компилятор
- ✨ Поддержка grpcio-tools
- ✨ mypy stubs генерация
- ✨ __init__.py генерация
- 📚 Python примеры

---

## [0.3.0] - TBD

### Added
- ✨ Система компиляции (Core Builder)
- ✨ Dependency resolver
- ✨ Build scheduler
- ✨ Параллельная компиляция
- ✨ Инкрементальная сборка
- ✨ Cache manager

### Changed
- ⚡ Оптимизации производительности

---

## [0.2.0] - TBD

### Added
- ✨ CLI интерфейс (cobra + viper)
- ✨ Парсер конфигураций (YAML/TOML/JSON)
- ✨ Валидация конфигураций
- ✨ Hot-reload конфигураций
- 📚 Документация по конфигурации

### Changed
- 🔧 Улучшена обработка ошибок

---

## [0.1.0] - TBD

### Added
- ✨ Кастомный логгер (pkg/logger)
- ✨ Система обработки ошибок (pkg/errors)
- ✨ Базовые утилиты (pkg/utils)
- ✨ Система конфигурации
- 🔧 Makefile для сборки
- 🧪 Unit тесты

### Changed
- 📖 Улучшена документация

---

## [0.0.0] - TBD

### Added
- 🎉 Инициализация проекта
- 📁 Базовая структура директорий
- 📝 README.md
- 📝 CONTRIBUTING.md
- 📝 LICENSE (MIT)
- 🔧 go.mod/go.sum
- 🔧 .gitignore
- 🔧 .editorconfig
- 🔧 .golangci.yml
- 🚀 CI/CD pipeline (GitHub Actions)
- 📚 Начальная документация

---

## Шаблон для будущих версий

## [X.Y.Z] - YYYY-MM-DD

### Added
- Новые функции

### Changed
- Изменения в существующей функциональности

### Deprecated
- Функции, которые будут удалены

### Removed
- Удаленные функции

### Fixed
- Исправления багов

### Security
- Исправления уязвимостей

---

[Unreleased]: https://github.com/yourorg/buffalo/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/yourorg/buffalo/releases/tag/v1.0.0
[0.9.0]: https://github.com/yourorg/buffalo/releases/tag/v0.9.0
[0.8.0]: https://github.com/yourorg/buffalo/releases/tag/v0.8.0
[0.7.0]: https://github.com/yourorg/buffalo/releases/tag/v0.7.0
[0.6.0]: https://github.com/yourorg/buffalo/releases/tag/v0.6.0
[0.5.0]: https://github.com/yourorg/buffalo/releases/tag/v0.5.0
[0.4.0]: https://github.com/yourorg/buffalo/releases/tag/v0.4.0
[0.3.0]: https://github.com/yourorg/buffalo/releases/tag/v0.3.0
[0.2.0]: https://github.com/yourorg/buffalo/releases/tag/v0.2.0
[0.1.0]: https://github.com/yourorg/buffalo/releases/tag/v0.1.0
[0.0.0]: https://github.com/yourorg/buffalo/releases/tag/v0.0.0
