# Buffalo v0.5.0-dev - Multi-Language Compilers & Versioning

## 🎯 Обзор изменений

Версия v0.5.0-dev добавляет поддержку всех основных языков программирования для компиляции protobuf/gRPC, а также интеллектуальную систему версионирования сгенерированных файлов.

## ✨ Новые возможности

### 1. Компиляторы для всех языков

#### Go Compiler (`internal/compiler/golang/`)
- Поддержка `protoc-gen-go` и `protoc-gen-go-grpc`
- Генерация `.pb.go` и `_grpc.pb.go` файлов
- Настройка Go module path
- Полная поддержка gRPC

**Установка зависимостей:**
```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

**Конфигурация:**
```yaml
languages:
  go:
    enabled: true
    module: github.com/yourorg/yourproject
    generator: protoc-gen-go
```

#### Rust Compiler (`internal/compiler/rust/`)
- Поддержка `rust-protobuf` и `prost`
- Генерация `.rs` файлов
- Интеграция с Cargo (для prost)

**Установка зависимостей:**
```bash
cargo install protobuf-codegen
```

**Конфигурация:**
```yaml
languages:
  rust:
    enabled: true
    generator: prost  # или rust-protobuf
```

#### C++ Compiler (`internal/compiler/cpp/`)
- Поддержка `protoc --cpp_out`
- Генерация `.pb.h`, `.pb.cc`, `.grpc.pb.h`, `.grpc.pb.cc` файлов
- Полная поддержка gRPC через `grpc_cpp_plugin`

**Установка зависимостей:**
Установите gRPC C++ из https://grpc.io/docs/languages/cpp/quickstart/

**Конфигурация:**
```yaml
languages:
  cpp:
    enabled: true
    namespace: myproject
```

### 2. Система версионирования (`internal/versioning/`)

Интеллектуальная система управления версиями сгенерированных файлов:

#### Возможности:
- **Автоматическое обнаружение изменений**: Вычисление SHA256 хеша proto файлов
- **Пропуск неизмененных файлов**: Компиляция только при изменении содержимого
- **Множество стратегий версионирования**:
  - `semantic`: v1, v2, v3 (рекомендуется)
  - `hash`: a1b2c3d4 (короткий хеш содержимого)
  - `timestamp`: 20260117093000
  - `git`: git commit hash (в разработке)

- **Два формата вывода**:
  - `directory`: Создание версионных директорий (`generated/v1/`, `generated/v2/`)
  - `suffix`: Добавление суффикса к файлам (`example_v1.pb.py`)

- **Автоочистка старых версий**: Хранение N последних версий

#### Конфигурация:
```yaml
versioning:
  enabled: true
  strategy: semantic        # hash, timestamp, semantic, git
  output_format: directory  # directory или suffix
  keep_versions: 3          # 0 = хранить все
```

#### Пример использования:

**Первая сборка:**
```bash
buffalo build --lang python
# Создаст: generated/v1/protos/example_pb2.py
```

**Повторная сборка без изменений:**
```bash
buffalo build --lang python
# Выведет: ⏭️  Skipping unchanged file
```

**После изменения proto файла:**
```bash
buffalo build --lang python
# Создаст: generated/v2/protos/example_pb2.py
# Старые версии: generated/v1/ (сохранены)
```

## 🏗️ Архитектурные изменения

### 1. Обновленный `config.Config`
Добавлена секция `VersioningConfig`:
```go
type VersioningConfig struct {
    Enabled      bool
    Strategy     string
    OutputFormat string
    KeepVersions int
}
```

### 2. Обновленный `builder.Executor`
- Интеграция `versioning.Manager`
- Автоматическая инициализация всех компиляторов
- Проверка версий перед компиляцией
- Сохранение состояния версий

### 3. Новый пакет `versioning`
```go
type Manager struct {
    enabled      bool
    strategy     Strategy
    outputFormat OutputFormat
    keepVersions int
    stateDir     string
}
```

## 📝 Примеры конфигурации

### Полная конфигурация с версионированием:
```yaml
project:
  name: "my-grpc-project"
  version: "1.0.0"

proto:
  paths: ["./protos"]
  import_paths: []

output:
  base_dir: "./generated"
  directories:
    python: python
    go: go
    rust: rust
    cpp: cpp

languages:
  python:
    enabled: true
    generator: grpcio-tools
    
  go:
    enabled: true
    module: github.com/myorg/myproject
    
  rust:
    enabled: true
    generator: prost
    
  cpp:
    enabled: true
    namespace: myproject

versioning:
  enabled: true
  strategy: semantic
  output_format: directory
  keep_versions: 3

build:
  workers: 4
  incremental: true
  cache:
    enabled: true
    directory: .buffalo-cache
```

## 🚀 Миграция с v0.4.0

1. **Обновите конфигурацию:**
   Добавьте секцию `versioning` в `buffalo.yaml` (опционально)

2. **Установите зависимости для новых языков:**
   ```bash
   # Go
   go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
   go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
   
   # Rust
   cargo install protobuf-codegen
   
   # C++ - см. документацию gRPC
   ```

3. **Обновите вызов Builder:**
   ```go
   // Старый способ:
   b, err := builder.New(builder.WithLogger(log))
   
   // Новый способ:
   b, err := builder.New(cfg, builder.WithLogger(log))
   ```

## 📊 Метрики и логи

### Новые лог-сообщения:
- `📦 Generating new version`: Создание новой версии
- `⏭️  Skipping unchanged file`: Пропуск неизмененного файла
- `✨ Compiled`: Успешная компиляция с количеством файлов

### Примеры логов:
```
09:51:53 [INFO] 📦 Generating new version file=protos/example.proto version=v1
09:51:54 [INFO] ✨ Compiled file=protos/example.proto generated=2 language=python
```

## 🔧 Внутренние улучшения

1. **Модульная архитектура компиляторов:**
   - Единый интерфейс `compiler.Compiler`
   - Независимая реализация для каждого языка
   - Простота добавления новых языков

2. **Умная обработка путей:**
   - Автоматическое определение `--proto_path`
   - Поддержка относительных и абсолютных путей
   - Корректная работа на Windows и Unix

3. **Улучшенная обработка ошибок:**
   - Детальные сообщения об ошибках
   - Проверка наличия внешних инструментов
   - Рекомендации по установке зависимостей

## 🐛 Известные ограничения

1. **Rust prost:** Требует ручной настройки `build.rs` в Cargo проекте
2. **Git versioning:** Стратегия `git` еще не реализована
3. **Тесты:** Unit тесты для новых компиляторов в разработке

## 🔮 Планы на v0.6.0

- [ ] Поддержка TypeScript (protobuf-ts)
- [ ] Поддержка Java (protoc-gen-java)
- [ ] Поддержка Swift (protoc-gen-swift)
- [ ] Git-based versioning
- [ ] Интеграция с CI/CD
- [ ] Web UI для визуализации версий

## 📚 Дополнительные ресурсы

- [Protocol Buffers Documentation](https://protobuf.dev/)
- [gRPC Documentation](https://grpc.io/)
- [Go Protocol Buffers](https://github.com/protocolbuffers/protobuf-go)
- [Rust prost](https://github.com/tokio-rs/prost)
- [gRPC C++](https://grpc.io/docs/languages/cpp/)
