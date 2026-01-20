# Проверка готовности системы к сборке

Buffalo теперь включает функционал автоматической проверки готовности системы к сборке для всех поддерживаемых языков.

## Обзор

Система проверяет наличие всех необходимых инструментов и зависимостей для сборки на основе вашей конфигурации `buffalo.yaml`. Если в конфиге включен Python, Go, Rust или C++, Buffalo автоматически проверит наличие соответствующих компиляторов, генераторов кода и библиотек.

## Основные возможности

### 1. Автоматическая проверка перед сборкой

При запуске `buffalo build` система автоматически проверяет готовность окружения:

```bash
buffalo build
```

**Выход:**
```
🔨 Starting build process
🔍 Checking system readiness...
✅ Protocol Buffers Compiler (protoc): libprotoc 3.21.12
✅ Go Language: go version go1.21.0
✅ protoc-gen-go: protoc-gen-go v1.31.0
✅ protoc-gen-go-grpc: protoc-gen-go-grpc v1.3.0
✅ Система готова к сборке
```

Если критичные компоненты отсутствуют, сборка будет остановлена с подробным сообщением о том, что нужно установить.

### 2. Пропуск проверки системы

Если вы уверены, что все установлено, можно пропустить проверку:

```bash
buffalo build --skip-system-check
```

### 3. Команда `buffalo check`

Быстрая проверка готовности проекта к сборке:

```bash
buffalo check
```

**Выход:**
```
🔍 Checking project configuration...

📄 Configuration File
  ✅ Config file loaded successfully

📦 Proto Files
  ✅ Found 5 proto file(s)

📁 Output Directory
  📂 Output: ./generated
  ✅ Output directory exists

🌐 Languages
  ✅ 2 language(s) enabled:
     • go
     • python

🔧 System Readiness Check
  ✅ 7 из 7 требований установлено

📊 Summary
  ✅ All checks passed! Your project is ready to build.
```

Используйте флаг `-v` для подробного вывода:

```bash
buffalo check -v
```

### 4. Команда `buffalo doctor`

Расширенная диагностика окружения:

```bash
# Проверить все поддерживаемые языки
buffalo doctor

# Проверить только включенные в конфиге языки
buffalo doctor --config-only

# Подробный вывод
buffalo doctor -v
```

**Выход:**
```
🏥 Running Buffalo Doctor - Environment Diagnostic
═══════════════════════════════════════════════════

✅ Buffalo Version: v1.0.0
✅ Operating System: windows/amd64
✅ protoc Compiler: libprotoc 3.21.12
✅ Go Language: go version go1.21.0
✅ protoc-gen-go: Found
✅ protoc-gen-go-grpc: Found
✅ Python Language: Python 3.11.0
✅ grpcio-tools: v1.59.0
⚠️  Rust Language: Not found in PATH

═══════════════════════════════════════════════════
✅ Passed: 8  ⚠️  Warnings: 1  ❌ Failed: 0

⚠️  Some checks have warnings. Some features may be limited.
```

## Проверяемые компоненты

### Общие требования
- **protoc** (Protocol Buffers Compiler) - критично для всех языков

### Go
Если `languages.go.enabled: true`:
- **Go** (компилятор Go) - критично
- **protoc-gen-go** - критично
- **protoc-gen-go-grpc** - критично (если используется gRPC)

### Python
Если `languages.python.enabled: true`:
- **Python 3** - критично
- **grpcio-tools** - критично
- **protobuf** (Python пакет) - критично

### Rust
Если `languages.rust.enabled: true`:
- **rustc** (Rust compiler) - критично
- **cargo** (Rust package manager) - критично
- **prost** (информация, устанавливается через Cargo.toml)

### C++
Если `languages.cpp.enabled: true`:
- **g++**, **clang++** или **cl** (MSVC) - критично (хотя бы один)
- **Protocol Buffers Library** для C++

## Команды установки

Buffalo автоматически предлагает команды установки для недостающих компонентов:

### Windows (с Scoop)
```bash
# Protocol Buffers
scoop install protobuf

# Go
scoop install go
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Python
scoop install python
pip install grpcio-tools protobuf

# C++
scoop install gcc  # или установите Visual Studio
```

### macOS (с Homebrew)
```bash
# Protocol Buffers
brew install protobuf

# Go
brew install go
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Python
brew install python3
pip3 install grpcio-tools protobuf

# Rust
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh

# C++
xcode-select --install  # Clang
brew install gcc        # GCC
```

### Linux (Ubuntu/Debian)
```bash
# Protocol Buffers
sudo apt install -y protobuf-compiler

# Go
wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Python
sudo apt install -y python3 python3-pip
pip3 install grpcio-tools protobuf

# Rust
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh

# C++
sudo apt install -y build-essential libprotobuf-dev
```

## Примеры использования

### Проверка перед первой сборкой
```bash
# 1. Инициализируйте проект
buffalo init

# 2. Проверьте готовность системы
buffalo check -v

# 3. Установите недостающие компоненты (если есть)

# 4. Выполните сборку
buffalo build
```

### CI/CD интеграция
```yaml
# .gitlab-ci.yml
build:
  script:
    - buffalo check  # Проверит готовность системы
    - buffalo build  # Выполнит сборку
```

### Отладка проблем с окружением
```bash
# Полная диагностика
buffalo doctor -v

# Проверка конкретной конфигурации
buffalo doctor --config-only

# Проверка с игнорированием системы (для отладки других проблем)
buffalo build --skip-system-check
```

## Архитектура

Новая функциональность реализована в модуле `internal/system/checker.go`:

- **SystemChecker** - основной класс для проверки готовности системы
- **Requirement** - описание требования к системе
- **CheckResult** - результат проверки требования

Интегрировано в команды:
- `buffalo build` - автоматическая проверка перед сборкой
- `buffalo check` - быстрая проверка готовности проекта
- `buffalo doctor` - расширенная диагностика окружения

## Настройка

В `buffalo.yaml` просто включите нужные языки:

```yaml
languages:
  go:
    enabled: true
    module: "github.com/example/myproject"
    generator: "grpc"
  
  python:
    enabled: true
    package: "myproject"
    generator: "grpc"
  
  rust:
    enabled: false  # Не будет проверяться
  
  cpp:
    enabled: false  # Не будет проверяться
```

Buffalo автоматически проверит только включенные языки.

## FAQ

**Q: Как отключить проверку системы?**
A: Используйте флаг `--skip-system-check` при запуске `buffalo build`.

**Q: Почему проверяется Rust, если он мне не нужен?**
A: Используйте `buffalo doctor --config-only` для проверки только включенных в конфиге языков.

**Q: Можно ли добавить свои проверки?**
A: Да, модуль `internal/system/checker.go` можно расширить, добавив новые `Requirement` объекты.

**Q: Что делать, если проверка ложно показывает ошибку?**
A: Запустите `buffalo doctor -v` для детальной диагностики и проверьте PATH. Также можно временно пропустить проверку с `--skip-system-check`.

## Связанные документы

- [DEVELOPMENT.md](DEVELOPMENT.md) - Руководство по разработке Buffalo
- [CONFIG_GUIDE.md](CONFIG_GUIDE.md) - Подробное описание конфигурации
- [QUICKSTART.md](QUICKSTART.md) - Быстрый старт с Buffalo
