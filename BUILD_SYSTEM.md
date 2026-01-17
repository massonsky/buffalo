# 🔨 Buffalo Build System Guide

Полное описание системы сборки Buffalo с поддержкой всех платформ.

## 📋 Содержание

- [Обзор](#обзор)
- [Требования](#требования)
- [Установка](#установка)
- [Сборка](#сборка)
- [Тестирование](#тестирование)
- [Docker](#docker)
- [Распространённые задачи](#распространённые-задачи)
- [Решение проблем](#решение-проблем)

---

## 🎯 Обзор

Buffalo имеет полнофункциональную систему сборки с поддержкой:

| Платформа | Основной инструмент | Альтернативы |
|-----------|---------------------|--------------|
| **Linux/macOS** | Make, build.sh | CMake, Go |
| **Windows** | PowerShell (build.ps1) | CMake, Go |
| **Docker** | Docker Compose | Docker CLI |

### Рекомендуемые комбинации

**Linux/macOS (рекомендуется):**
```bash
./build.sh install  # или: make install-system
```

**Windows (рекомендуется):**
```powershell
.\build.ps1 -Target install
```

**Все платформы (универсально):**
```bash
docker build -t buffalo:latest .
docker run --rm -v $(pwd):/workspace buffalo:latest build
```

---

## 📦 Требования

### Минимальные требования

- **Go:** 1.21 или новше
- **protoc:** 3.20 или новше (рекомендуется 3.21+)
- **Git:** для работы версионирования

### Дополнительно для разработки

- **golangci-lint:** для лидирования (рекомендуется)
- **Docker:** для контейнеризации (опционально)
- **CMake:** для альтернативной сборки (опционально)

### Установка зависимостей

**Linux (Ubuntu/Debian):**
```bash
sudo apt-get install -y golang-1.21 protobuf-compiler git
sudo apt-get install -y golangci-lint  # опционально
```

**macOS:**
```bash
brew install go protobuf git
brew install golangci-lint  # опционально
```

**Windows:**
```powershell
# Используя Chocolatey
choco install golang protoc git
choco install golangci-lint  # опционально

# Или Windows Package Manager
winget install GoLang.Go
winget install protocolbuffers.protoc
```

---

## 💾 Установка

### Способ 1: Автоматический скрипт (БЫСТРО)

**Linux/macOS:**
```bash
curl -sSL https://raw.githubusercontent.com/massonsky/buffalo/dev/install.sh | bash
buffalo --version
```

**Windows (PowerShell):**
```powershell
irm https://raw.githubusercontent.com/massonsky/buffalo/dev/install.ps1 | iex
buffalo --version
```

### Способ 2: Из исходников (ГИБКО)

```bash
# Клонирование
git clone https://github.com/massonsky/buffalo.git
cd buffalo

# Linux/macOS - Вариант A (build.sh)
chmod +x build.sh
./build.sh install

# Linux/macOS - Вариант B (Makefile)
sudo make install-system

# Linux/macOS - Вариант C (CMake)
cmake -B build
cmake --build build --target install

# Windows (PowerShell)
.\build.ps1 -Target install
```

### Способ 3: Docker

```bash
# Быстрый старт с Docker
docker build -t buffalo:latest .

# Использование
docker run --rm -v $(pwd):/workspace buffalo:latest buffalo --version

# Через docker-compose
docker-compose pull  # или build
docker-compose run buffalo buffalo --version
```

### Способ 4: Напрямую через Go

```bash
go install github.com/massonsky/buffalo/cmd/buffalo@latest
```

---

## 🔨 Сборка

### Linux/macOS - Основные команды

#### Через build.sh (рекомендуется)

```bash
# Быстрая сборка
./build.sh build              # → bin/buffalo

# Сборка для всех платформ
./build.sh build-all          # → build/buffalo-*-*

# Справка
./build.sh help
```

#### Через Makefile

```bash
# Быстрая сборка
make build                    # → bin/buffalo

# Сборка для всех платформ
make release                  # → build/buffalo-*

# Справка
make help
```

#### Через CMake

```bash
# Инициализация и сборка
cmake -B build
cmake --build build --target build

# Сборка для конкретной платформы
cmake -B build -DCMAKE_SYSTEM_NAME=Linux -DCMAKE_SYSTEM_PROCESSOR=x86_64
cmake --build build --target build
```

#### Через Go (минимально)

```bash
# Простая сборка без метаданных
go build -o bin/buffalo ./cmd/buffalo

# С версией и метаданными (как в build.sh)
go build \
  -ldflags "-X github.com/massonsky/buffalo/internal/version.Version=v0.5.0" \
  -o bin/buffalo \
  ./cmd/buffalo
```

### Windows - Основные команды

#### Через build.ps1 (рекомендуется)

```powershell
# Быстрая сборка
.\build.ps1 build             # → bin\buffalo.exe

# Сборка для всех платформ
.\build.ps1 build-all         # → build\buffalo-*

# Справка
.\build.ps1 help

# С индивидуальными опциями
.\build.ps1 -Target build -Verbose
.\build.ps1 -Target install -InstallPrefix "C:\custom\path"
```

#### Через Go напрямую

```powershell
# Простая сборка
go build -o bin\buffalo.exe .\cmd\buffalo

# С метаданными
$version = git describe --tags --always
go build `
  -ldflags "-X github.com/massonsky/buffalo/internal/version.Version=$version" `
  -o bin\buffalo.exe `
  .\cmd\buffalo
```

### Docker

```bash
# Сборка образа
docker build -t buffalo:latest .
docker build -t buffalo:dev --target builder .

# Запуск build внутри контейнера
docker run --rm \
  -v $(pwd):/workspace \
  buffalo:latest \
  buffalo build

# Использование docker-compose
docker-compose up buffalo   # стандартная конфигурация
docker-compose up buffalo-dev  # с горячей перезагрузкой
```

---

## 🧪 Тестирование

### Linux/macOS

```bash
# Через build.sh
./build.sh test              # базовые тесты
./build.sh test-coverage     # с покрытием (→ coverage.html)
./build.sh check             # все проверки (fmt+vet+lint+test)

# Через Makefile
make test                    # базовые тесты
make test-coverage          # с покрытием
make check                   # все проверки

# Вручную (Go)
go test -v -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### Windows

```powershell
# Через build.ps1
.\build.ps1 -Target test           # базовые тесты
.\build.ps1 -Target test-coverage  # с покрытием
.\build.ps1 -Target check          # все проверки

# Вручную (Go)
go test -v -race -coverprofile=coverage.out .\...
go tool cover -html=coverage.out -o coverage.html
```

### Docker

```bash
# Тесты внутри контейнера
docker run --rm -v $(pwd):/workspace buffalo:latest \
  sh -c "go test -v -race ./..."

# Через docker-compose
docker-compose run buffalo go test -v -race ./...
```

---

## 🐳 Docker

### Структура Docker окружения

**Dockerfile** - Многоступенчатая сборка:
- Stage 1: `builder` - Go 1.23 alpine, компиляция
- Stage 2: `runtime` - Alpine, готовый бинарник

**docker-compose.yml** - Два сервиса:
- `buffalo` - Стандартное использование
- `buffalo-dev` - Разработка с volume mounts

### Примеры использования

```bash
# Сборка и запуск
docker build -t buffalo:latest .
docker run --rm -v $(pwd):/workspace buffalo:latest buffalo build

# Через docker-compose
docker-compose build
docker-compose run buffalo buffalo build

# Development с горячей перезагрузкой
docker-compose -f docker-compose.yml up buffalo-dev

# Запуск конкретной команды
docker run --rm -v $(pwd):/workspace buffalo:latest \
  buffalo build --proto ./protos --lang python,go
```

### Переменные окружения Docker

```bash
# BUFFALO_LOG_LEVEL
docker run -e BUFFALO_LOG_LEVEL=debug \
  -v $(pwd):/workspace \
  buffalo:latest

# Рабочая директория
docker run -w /workspace \
  -v $(pwd):/workspace \
  buffalo:latest
```

---

## 📋 Распространённые задачи

### Установка в систему

```bash
# Linux/macOS - с sudo
sudo make install-system      # → /usr/local/bin/buffalo

# Linux/macOS - в домашнюю папку
INSTALL_PREFIX=~/.local ./build.sh install

# Windows - автоматически в Program Files
.\build.ps1 -Target install

# Windows - в кастомную папку
.\build.ps1 -Target install -InstallPrefix "C:\custom\buffalo"
```

### Проверка версии

```bash
# Через build.sh
./build.sh version

# Через Makefile
make version

# Через build.ps1
.\build.ps1 -Target version

# Через установленный бинарник
buffalo --version
```

### Форматирование и линтинг

```bash
# Linux/macOS
./build.sh fmt               # go fmt
./build.sh vet               # go vet
./build.sh lint              # golangci-lint

# Windows
.\build.ps1 -Target fmt
.\build.ps1 -Target vet
.\build.ps1 -Target lint

# Все вместе (проверки + тесты)
./build.sh check             # Linux/macOS
.\build.ps1 -Target check    # Windows
```

### Сборка для разных платформ

```bash
# Все сразу
./build.sh build-all         # Linux/macOS
.\build.ps1 -Target build-all # Windows

# Конкретная платформа
GOOS=linux GOARCH=amd64 go build -o bin/buffalo-linux ./cmd/buffalo
GOOS=darwin GOARCH=arm64 go build -o bin/buffalo-darwin-arm ./cmd/buffalo
GOOS=windows GOARCH=amd64 go build -o bin/buffalo-windows.exe ./cmd/buffalo
```

### Очистка

```bash
# Linux/macOS
./build.sh clean             # только артефакты сборки
./build.sh clean-all         # + кэши Go

# Windows
.\build.ps1 -Target clean
.\build.ps1 -Target clean-all

# Makefile
make clean
make clean-all
```

### Создание релиза

```bash
# Сборка для всех платформ (автоматически создаёт архивы)
make release                 # → build/buffalo-v0.5.0-*

# Вручную
./build.sh build-all
tar -czf buffalo-linux-amd64.tar.gz -C build buffalo-linux-amd64
```

---

## 🆘 Решение проблем

### «command not found: buffalo»

**Linux/macOS:**
```bash
# Проверить PATH
echo $PATH

# Проверить где установлен
which buffalo

# Переустановить с явным путём
INSTALL_PREFIX=$HOME/.local ./build.sh install
echo 'export PATH=$HOME/.local/bin:$PATH' >> ~/.bashrc
source ~/.bashrc
```

**Windows:**
```powershell
# Проверить PATH
$env:PATH

# Переустановить
.\build.ps1 -Target install
# Перезагрузить PowerShell/CMD после этого
```

### «protoc: command not found»

**Linux:**
```bash
sudo apt-get install protobuf-compiler
protoc --version
```

**macOS:**
```bash
brew install protobuf
protoc --version
```

**Windows:**
```powershell
choco install protoc
# или
winget install protocolbuffers.protoc
```

### Build fails: «go: cannot find package»

```bash
# Обновить модули
go mod download
go mod tidy

# Переустановить зависимости
go clean -modcache
go mod download
```

### Permissions denied (Linux/macOS)

```bash
# Дать права на исполнение скриптам
chmod +x build.sh install.sh

# Для установки в /usr/local
sudo make install-system

# Для установки в домашнюю папку
./build.sh install  # без sudo
```

### Docker build fails

```bash
# Очистить кэш Docker
docker system prune -a

# Перестроить без кэша
docker build --no-cache -t buffalo:latest .

# Проверить logs
docker build -t buffalo:latest . 2>&1 | tail -20
```

### CMake: «Compiler not found»

```bash
# Linux
sudo apt-get install build-essential cmake

# macOS
brew install cmake

# Windows - используйте build.ps1 вместо CMake
# CMake требует Visual Studio Build Tools на Windows
```

### Coverage report отсутствует

```bash
# Linux/macOS
./build.sh test-coverage    # создаёт coverage.html
open coverage.html          # или: xdg-open / firefox

# Windows
.\build.ps1 -Target test-coverage
# Открыть coverage.html в браузере вручную

# Docker
docker run --rm -v $(pwd):/workspace buffalo:latest \
  go test -coverprofile=coverage.out ./...
docker run --rm -v $(pwd):/workspace buffalo:latest \
  go tool cover -html=coverage.out -o coverage.html
```

---

## 📚 Дополнительные ресурсы

- [INSTALL.md](INSTALL.md) - Подробное руководство установки
- [README.md](README.md) - Основная документация проекта
- [Makefile](Makefile) - Все целевые задачи
- [build.sh](build.sh) - Unix-скрипт сборки (с комментариями)
- [build.ps1](build.ps1) - Windows-скрипт сборки (с комментариями)

---

## 🎓 Для контрибьюторов

### Development workflow

```bash
# 1. Форк + клон
git clone git@github.com:yourusername/buffalo.git
cd buffalo

# 2. Создать ветку
git checkout -b feature/my-feature

# 3. Разработка с автоматическими проверками
./build.sh check          # тесты + форматирование + линтинг

# 4. Коммит и пуш
git add .
git commit -m "feat: add my feature"
git push origin feature/my-feature

# 5. Pull Request
# → создать PR в https://github.com/massonsky/buffalo
```

### Development с Docker

```bash
# Развернуть окружение разработки
docker-compose -f docker-compose.yml up -d buffalo-dev

# Редактировать файлы на хосте
# Контейнер автоматически пересобирает при изменениях

# Отследить логи
docker-compose logs -f buffalo-dev

# Остановить
docker-compose down
```

---

## 📝 Лицензия

MIT - см. [LICENSE](LICENSE)

---

**Последнее обновление:** 2024 | **Версия:** v0.5.0
