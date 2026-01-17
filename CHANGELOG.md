# Changelog

Все заметные изменения в проекте Buffalo будут документироваться в этом файле.

Формат основан на [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
и проект следует [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Планируется
- TypeScript/JavaScript компилятор
- Java/Kotlin компилятор
- Remote cache поддержка
- Web UI для конфигурации

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
