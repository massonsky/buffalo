# Buffalo - Архитектура системы

## Обзор

Buffalo построен на принципах модульности, расширяемости и производительности. Архитектура следует лучшим практикам Go разработки и Domain-Driven Design.

---

## Высокоуровневая архитектура

```
┌─────────────────────────────────────────────────────────┐
│                      CLI Interface                       │
│                    (cobra + viper)                       │
└──────────────────────┬──────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────┐
│                  Configuration Manager                   │
│         (YAML/TOML/JSON + Env + CLI Flags)              │
└──────────────────────┬──────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────┐
│                   Build Orchestrator                     │
│              (Координация всего процесса)                │
└───────┬─────────────┬──────────────┬──────────────┬─────┘
        │             │              │              │
        ▼             ▼              ▼              ▼
┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐
│  Proto   │  │Dependency│  │  Build   │  │  Cache   │
│ Scanner  │  │ Resolver │  │Scheduler │  │ Manager  │
└──────────┘  └──────────┘  └──────────┘  └──────────┘
        │             │              │              │
        └─────────────┴──────────────┴──────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────┐
│                  Compiler Interface                      │
└───────┬─────────────┬──────────────┬──────────────┬─────┘
        │             │              │              │
        ▼             ▼              ▼              ▼
┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐
│  Python  │  │    Go    │  │   Rust   │  │   C++    │
│Compiler  │  │Compiler  │  │Compiler  │  │Compiler  │
└──────────┘  └──────────┘  └──────────┘  └──────────┘
        │             │              │              │
        └─────────────┴──────────────┴──────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────┐
│                   Output Manager                         │
│            (Управление выходными файлами)                │
└─────────────────────────────────────────────────────────┘

           Поддерживающие компоненты:
┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐
│  Logger  │  │  Errors  │  │  Utils   │  │ Metrics  │
└──────────┘  └──────────┘  └──────────┘  └──────────┘
```

---

## Компоненты системы

### 1. CLI Interface (cmd/buffalo)

**Назначение:** Точка входа в приложение, обработка команд пользователя.

**Технологии:**
- `cobra` - CLI framework
- `viper` - Configuration management

**Команды:**
```go
buffalo build [flags]      // Основная команда сборки
buffalo init               // Инициализация нового проекта
buffalo validate           // Валидация proto файлов
buffalo clean              // Очистка генерированных файлов
buffalo watch              // Watch mode
buffalo config             // Управление конфигурацией
buffalo plugin             // Управление плагинами
buffalo version            // Информация о версии
```

**Флаги:**
```
--config, -c          Путь к конфигурационному файлу
--output, -o          Выходная директория
--lang, -l            Целевые языки (python,go,rust,cpp)
--verbose, -v         Уровень детализации логов
--parallel, -p        Количество параллельных процессов
--incremental, -i     Инкрементальная сборка
--watch, -w           Watch mode
--dry-run             Dry run режим
```

---

### 2. Configuration Manager (internal/config)

**Назначение:** Управление конфигурацией из различных источников с учетом приоритетов.

**Приоритет настроек:**
1. CLI флаги (высший приоритет)
2. Переменные окружения
3. Конфигурационный файл
4. Значения по умолчанию

**Структура конфигурации:**

```go
type Config struct {
    Version string         `yaml:"version" json:"version"`
    Global  GlobalConfig   `yaml:"global" json:"global"`
    Languages map[string]LanguageConfig `yaml:"languages" json:"languages"`
    Logging LoggingConfig  `yaml:"logging" json:"logging"`
    Cache   CacheConfig    `yaml:"cache" json:"cache"`
    Plugins []PluginConfig `yaml:"plugins" json:"plugins"`
}

type GlobalConfig struct {
    ProtoPaths   []string `yaml:"proto_path" json:"proto_path"`
    OutputPath   string   `yaml:"output_path" json:"output_path"`
    ImportPaths  []string `yaml:"import_paths" json:"import_paths"`
    TempDir      string   `yaml:"temp_dir" json:"temp_dir"`
    Parallel     int      `yaml:"parallel" json:"parallel"`
    Incremental  bool     `yaml:"incremental" json:"incremental"`
}

type LanguageConfig struct {
    Enabled     bool              `yaml:"enabled" json:"enabled"`
    Output      string            `yaml:"output" json:"output"`
    Plugins     []string          `yaml:"plugins" json:"plugins"`
    Options     map[string]string `yaml:"options" json:"options"`
    Protoc      ProtocConfig      `yaml:"protoc" json:"protoc"`
}

type ProtocConfig struct {
    Path    string   `yaml:"path" json:"path"`
    Version string   `yaml:"version" json:"version"`
    Args    []string `yaml:"args" json:"args"`
}
```

**Функции:**
- Чтение конфигурации из YAML/TOML/JSON
- Валидация конфигурации
- Мержинг конфигураций из разных источников
- Hot-reload (для watch mode)
- Schema validation

---

### 3. Build Orchestrator (internal/orchestrator)

**Назначение:** Координация всего процесса сборки.

**Основной workflow:**

```go
type Orchestrator struct {
    config    *config.Config
    scanner   *scanner.ProtoScanner
    resolver  *resolver.DependencyResolver
    scheduler *scheduler.BuildScheduler
    cache     *cache.Manager
    logger    *logger.Logger
    metrics   *metrics.Collector
}

// Основной метод сборки
func (o *Orchestrator) Build(ctx context.Context) error {
    // 1. Сканирование proto файлов
    files, err := o.scanner.Scan(o.config.Global.ProtoPaths)
    
    // 2. Разрешение зависимостей
    graph, err := o.resolver.Resolve(files)
    
    // 3. Проверка кэша
    tasks := o.cache.FilterCachedFiles(graph)
    
    // 4. Составление плана сборки
    plan, err := o.scheduler.CreatePlan(tasks)
    
    // 5. Параллельная компиляция
    results := o.ExecuteParallel(ctx, plan)
    
    // 6. Обработка результатов
    return o.ProcessResults(results)
}
```

**Функции:**
- Управление жизненным циклом сборки
- Координация компонентов
- Error handling и recovery
- Progress reporting
- Metrics collection

---

### 4. Proto Scanner (internal/scanner)

**Назначение:** Поиск и парсинг .proto файлов.

**Функциональность:**
```go
type ProtoScanner struct {
    paths   []string
    filters []Filter
    parser  *ProtoParser
}

type ProtoFile struct {
    Path         string
    Package      string
    Imports      []string
    Dependencies []string
    Services     []ServiceDescriptor
    Messages     []MessageDescriptor
    Hash         string
    ModTime      time.Time
}

func (s *ProtoScanner) Scan(paths []string) ([]*ProtoFile, error)
func (s *ProtoScanner) Parse(file string) (*ProtoFile, error)
func (s *ProtoScanner) Watch(ctx context.Context) (<-chan Event, error)
```

**Особенности:**
- Рекурсивный поиск .proto файлов
- Парсинг proto синтаксиса (proto2/proto3)
- Извлечение метаданных (package, imports, options)
- File watching для watch mode
- Хэширование для кэша

---

### 5. Dependency Resolver (internal/resolver)

**Назначение:** Построение графа зависимостей между proto файлами.

**Структура:**
```go
type DependencyResolver struct {
    graph *DependencyGraph
}

type DependencyGraph struct {
    Nodes map[string]*Node
    Edges map[string][]string
}

type Node struct {
    File         *ProtoFile
    Dependencies []*Node
    Dependents   []*Node
    Level        int // Уровень в графе
}

func (r *DependencyResolver) Resolve(files []*ProtoFile) (*DependencyGraph, error)
func (r *DependencyResolver) TopologicalSort() ([]*Node, error)
func (r *DependencyResolver) DetectCycles() ([][]string, error)
```

**Алгоритмы:**
- Построение DAG (Directed Acyclic Graph)
- Топологическая сортировка
- Обнаружение циклических зависимостей
- Вычисление уровней компиляции

---

### 6. Build Scheduler (internal/scheduler)

**Назначение:** Планирование и оптимизация порядка компиляции.

**Функциональность:**
```go
type BuildScheduler struct {
    maxParallel int
    optimizer   *Optimizer
}

type BuildPlan struct {
    Stages []BuildStage
    Total  int
}

type BuildStage struct {
    Level int
    Tasks []*BuildTask
}

type BuildTask struct {
    File       *ProtoFile
    Compilers  []Compiler
    Priority   int
    EstimatedDuration time.Duration
}

func (s *BuildScheduler) CreatePlan(graph *DependencyGraph) (*BuildPlan, error)
func (s *BuildScheduler) Optimize(plan *BuildPlan) error
```

**Оптимизации:**
- Параллелизация независимых задач
- Приоритизация критического пути
- Load balancing между воркерами
- Адаптивная подстройка параллелизма

---

### 7. Cache Manager (internal/cache)

**Назначение:** Управление кэшем для инкрементальной сборки.

**Структура:**
```go
type CacheManager struct {
    storage Storage
    hasher  Hasher
}

type CacheEntry struct {
    FileHash    string
    ConfigHash  string
    OutputFiles []string
    Timestamp   time.Time
    Metadata    map[string]interface{}
}

func (c *CacheManager) Get(file *ProtoFile) (*CacheEntry, bool)
func (c *CacheManager) Set(file *ProtoFile, entry *CacheEntry) error
func (c *CacheManager) Invalidate(file *ProtoFile) error
func (c *CacheManager) Clear() error
```

**Стратегии кэширования:**
- Content-based hashing
- Dependency-aware invalidation
- Configurable TTL
- Compression поддержка
- Persistence (файловая система или БД)

---

### 8. Compiler Interface (internal/compiler)

**Назначение:** Унифицированный интерфейс для всех языковых компиляторов.

**Интерфейс:**
```go
type Compiler interface {
    // Метаданные
    Name() string
    Language() string
    Version() string
    
    // Проверки
    Validate(config *config.LanguageConfig) error
    CheckDependencies() error
    
    // Компиляция
    Compile(ctx context.Context, task *BuildTask) (*CompileResult, error)
    
    // Управление
    Setup() error
    Cleanup() error
}

type CompileResult struct {
    Success     bool
    OutputFiles []string
    Errors      []error
    Warnings    []string
    Duration    time.Duration
    Stats       CompileStats
}

type CompileStats struct {
    FilesGenerated int
    BytesWritten   int64
    LinesOfCode    int
}
```

**Базовая реализация:**
```go
type BaseCompiler struct {
    name     string
    language string
    config   *config.LanguageConfig
    logger   *logger.Logger
    executor *CommandExecutor
}

// Общие методы для всех компиляторов
func (b *BaseCompiler) BuildProtocCommand(task *BuildTask) ([]string, error)
func (b *BaseCompiler) ExecuteCommand(ctx context.Context, cmd []string) error
func (b *BaseCompiler) ValidateOutput(expected []string) error
```

---

### 9. Language Compilers

#### 9.1 Python Compiler (internal/compiler/python)

```go
type PythonCompiler struct {
    *BaseCompiler
    grpcTools  GRPCToolsManager
    mypyStubs  bool
}

func (p *PythonCompiler) Compile(ctx context.Context, task *BuildTask) (*CompileResult, error) {
    // 1. Генерация pb2.py файлов
    // 2. Генерация pb2_grpc.py файлов (если grpc enabled)
    // 3. Генерация pyi stubs (если mypy enabled)
    // 4. Генерация __init__.py файлов
    // 5. Настройка import путей
}

func (p *PythonCompiler) GenerateInitFiles(outputDir string) error
func (p *PythonCompiler) GenerateSetupPy(outputDir string) error
```

**Особенности:**
- Поддержка grpcio-tools
- Генерация type stubs для mypy
- Автоматическая структура пакетов
- Поддержка различных Python версий

#### 9.2 Go Compiler (internal/compiler/golang)

```go
type GoCompiler struct {
    *BaseCompiler
    goPackagePrefix string
    moduleMode      bool
}

func (g *GoCompiler) Compile(ctx context.Context, task *BuildTask) (*CompileResult, error) {
    // 1. Генерация .pb.go файлов (protoc-gen-go)
    // 2. Генерация _grpc.pb.go файлов (protoc-gen-go-grpc)
    // 3. Настройка go_package опций
    // 4. Генерация go.mod (если нужно)
}

func (g *GoCompiler) DetermineGoPackage(file *ProtoFile) string
func (g *GoCompiler) GenerateGoMod(outputDir string) error
```

**Особенности:**
- Автоматическая настройка go_package
- Поддержка go modules
- Vendor режим
- Интеграция с buf

#### 9.3 Rust Compiler (internal/compiler/rust)

```go
type RustCompiler struct {
    *BaseCompiler
    useProst bool
    useTonic bool
}

func (r *RustCompiler) Compile(ctx context.Context, task *BuildTask) (*CompileResult, error) {
    // 1. Генерация .rs файлов (prost)
    // 2. Генерация tonic gRPC кода
    // 3. Генерация Cargo.toml
    // 4. Настройка build.rs
}

func (r *RustCompiler) GenerateCargoToml(outputDir string) error
func (r *RustCompiler) GenerateBuildRs(outputDir string) error
```

**Особенности:**
- Поддержка prost (Protocol Buffers)
- Поддержка tonic (gRPC)
- Типобезопасные bindings
- Интеграция с Cargo

#### 9.4 C++ Compiler (internal/compiler/cpp)

```go
type CppCompiler struct {
    *BaseCompiler
    standard    string // C++11, C++14, C++17, C++20
    grpcPlugin  string
}

func (c *CppCompiler) Compile(ctx context.Context, task *BuildTask) (*CompileResult, error) {
    // 1. Генерация .pb.h и .pb.cc файлов
    // 2. Генерация .grpc.pb.h и .grpc.pb.cc файлов
    // 3. Генерация CMakeLists.txt
    // 4. Настройка include путей
}

func (c *CppCompiler) GenerateCMakeLists(outputDir string) error
func (c *CppCompiler) GeneratePkgConfig(outputDir string) error
```

**Особенности:**
- Поддержка различных стандартов C++
- CMake интеграция
- pkg-config файлы
- Кроссплатформенность

---

### 10. Output Manager (internal/output)

**Назначение:** Управление выходными файлами и директориями.

```go
type OutputManager struct {
    baseDir string
    layout  LayoutStrategy
}

type LayoutStrategy interface {
    GetOutputPath(lang string, file *ProtoFile) string
    CreateStructure() error
}

// Стратегии размещения файлов
type FlatLayout struct{}        // Все в одной директории
type LanguageLayout struct{}    // Раздельные директории по языкам
type MirrorLayout struct{}      // Зеркалирование структуры proto
type CustomLayout struct{}      // Кастомная стратегия

func (o *OutputManager) PrepareOutputDir(lang string) error
func (o *OutputManager) WriteFile(path string, content []byte) error
func (o *OutputManager) CleanOutput(lang string) error
```

---

### 11. Plugin System (internal/plugin)

**Назначение:** Расширяемость через плагины.

**Архитектура плагинов:**
```go
type Plugin interface {
    Name() string
    Version() string
    Description() string
    
    Initialize(config map[string]interface{}) error
    Execute(ctx context.Context, data *PluginData) error
    Cleanup() error
}

type PluginData struct {
    ProtoFiles  []*ProtoFile
    CompileResults []*CompileResult
    Config      *config.Config
    Context     map[string]interface{}
}

// Plugin Manager
type PluginManager struct {
    plugins map[string]Plugin
    loader  *PluginLoader
}

func (pm *PluginManager) Load(path string) error
func (pm *PluginManager) Execute(stage PluginStage, data *PluginData) error
```

**Стадии выполнения плагинов:**
- `PreScan` - До сканирования файлов
- `PostScan` - После сканирования
- `PreCompile` - До компиляции
- `PostCompile` - После компиляции
- `PreOutput` - До записи файлов
- `PostOutput` - После записи

**Типы плагинов:**
- **Linters:** Проверка proto файлов
- **Validators:** Валидация кода
- **Transformers:** Трансформация proto файлов
- **Generators:** Дополнительная кодогенерация
- **Compilers:** Новые языки компиляции

---

## Поддерживающие компоненты (pkg/)

### 1. Logger (pkg/logger)

**Структура:**
```go
type Logger struct {
    level      Level
    formatter  Formatter
    outputs    []Output
    fields     Fields
}

type Level int
const (
    DEBUG Level = iota
    INFO
    WARN
    ERROR
    FATAL
)

type Formatter interface {
    Format(entry *Entry) ([]byte, error)
}

// Форматтеры
type JSONFormatter struct{}
type TextFormatter struct{}
type ColoredFormatter struct{}

// Outputs
type ConsoleOutput struct{}
type FileOutput struct {
    path     string
    rotation RotationConfig
}
type SyslogOutput struct{}
```

**Функции:**
```go
func (l *Logger) Debug(msg string, fields ...Field)
func (l *Logger) Info(msg string, fields ...Field)
func (l *Logger) Warn(msg string, fields ...Field)
func (l *Logger) Error(msg string, fields ...Field)
func (l *Logger) Fatal(msg string, fields ...Field)

func (l *Logger) WithFields(fields Fields) *Logger
func (l *Logger) WithContext(ctx context.Context) *Logger
```

**Особенности:**
- Structured logging
- Multiple outputs
- Log rotation
- Context support
- Performance optimization (buffer pool)

---

### 2. Errors (pkg/errors)

**Типы ошибок:**
```go
// Базовый тип ошибки
type Error struct {
    Code       ErrorCode
    Message    string
    Cause      error
    Stack      []uintptr
    Context    map[string]interface{}
    Timestamp  time.Time
}

type ErrorCode string
const (
    ErrConfig          ErrorCode = "CONFIG_ERROR"
    ErrProtoNotFound   ErrorCode = "PROTO_NOT_FOUND"
    ErrCompilation     ErrorCode = "COMPILATION_FAILED"
    ErrDependency      ErrorCode = "DEPENDENCY_ERROR"
    ErrValidation      ErrorCode = "VALIDATION_ERROR"
    ErrIO              ErrorCode = "IO_ERROR"
)

func New(code ErrorCode, message string) *Error
func Wrap(err error, code ErrorCode, message string) *Error
func (e *Error) Error() string
func (e *Error) Unwrap() error
func (e *Error) Is(target error) bool
```

**Функции:**
- Error wrapping
- Stack traces
- Error codes
- Context information
- Error reporting

---

### 3. Utils (pkg/utils)

**Файловые операции:**
```go
func FindFiles(root string, pattern string) ([]string, error)
func CopyFile(src, dst string) error
func CopyDir(src, dst string) error
func EnsureDir(path string) error
func CleanDir(path string) error
func ComputeHash(file string) (string, error)
```

**Работа с путями:**
```go
func NormalizePath(path string) string
func IsAbsolutePath(path string) bool
func JoinPath(elem ...string) string
func GetRelativePath(base, target string) (string, error)
```

**Валидация:**
```go
func ValidateProtoFile(path string) error
func ValidateConfig(config *Config) error
func ValidateOutput(path string) error
```

**Concurrency helpers:**
```go
type WorkerPool struct {
    size    int
    tasks   chan Task
    results chan Result
}

func NewWorkerPool(size int) *WorkerPool
func (wp *WorkerPool) Submit(task Task)
func (wp *WorkerPool) Wait() []Result
```

---

### 4. Metrics (pkg/metrics)

**Сбор метрик:**
```go
type Collector struct {
    metrics map[string]Metric
}

type Metric interface {
    Name() string
    Value() interface{}
    Type() MetricType
}

type MetricType int
const (
    Counter MetricType = iota
    Gauge
    Histogram
    Summary
)

// Метрики сборки
type BuildMetrics struct {
    TotalFiles      int
    CompiledFiles   int
    CachedFiles     int
    FailedFiles     int
    Duration        time.Duration
    CompileDurations map[string]time.Duration
}

func (c *Collector) RecordBuild(metrics *BuildMetrics)
func (c *Collector) Export() ([]byte, error)
```

---

## Паттерны и принципы

### 1. Design Patterns

**Strategy Pattern:**
- Разные стратегии компиляции для языков
- Разные layout стратегии для output
- Разные форматтеры для логов

**Factory Pattern:**
- Создание компиляторов
- Создание плагинов
- Создание конфигураций

**Observer Pattern:**
- Watch mode для файлов
- Events для плагинов
- Progress reporting

**Chain of Responsibility:**
- Обработка ошибок
- Middleware для плагинов
- Validation pipeline

### 2. SOLID Principles

**Single Responsibility:**
- Каждый компонент имеет одну задачу
- Разделение concerns (config, build, compile)

**Open/Closed:**
- Расширяемость через плагины
- Интерфейсы для компонентов

**Liskov Substitution:**
- Все компиляторы реализуют единый интерфейс
- Плагины взаимозаменяемы

**Interface Segregation:**
- Маленькие, специфичные интерфейсы
- Не заставляем реализовывать ненужное

**Dependency Inversion:**
- Зависимости через интерфейсы
- Dependency injection

### 3. Go Best Practices

- Использование context для cancellation
- Error handling без паники
- Structured logging
- Table-driven tests
- Benchmarking критичных путей
- Профилирование производительности
- Documentation через godoc
- Code generation где нужно

---

## Производительность

### Оптимизации:

1. **Параллельная компиляция:**
   - Worker pool с настраиваемым размером
   - Batching задач
   - Load balancing

2. **Кэширование:**
   - Content-based cache
   - Dependency-aware invalidation
   - Compression

3. **Memory management:**
   - Buffer pools для логов
   - Streaming для больших файлов
   - Efficient data structures

4. **IO оптимизации:**
   - Batch file operations
   - Async IO где возможно
   - Efficient path operations

### Целевые метрики:
- Компиляция 100 proto файлов: < 10 секунд
- Memory usage: < 100MB
- Binary size: < 20MB
- Startup time: < 100ms

---

## Безопасность

### Меры безопасности:

1. **Валидация входных данных:**
   - Проверка путей (path traversal)
   - Валидация конфигураций
   - Санитизация command arguments

2. **Sandbox для плагинов:**
   - Ограничение доступа к файловой системе
   - Timeout для выполнения
   - Resource limits

3. **Безопасное выполнение команд:**
   - Escaping arguments
   - Environment isolation
   - Timeout механизмы

4. **Audit logging:**
   - Логирование всех операций
   - Security events
   - Error tracking

---

## Тестирование

### Стратегия тестирования:

1. **Unit Tests:**
   - Покрытие > 85%
   - Table-driven tests
   - Mock interfaces
   - Тестирование error paths

2. **Integration Tests:**
   - End-to-end сборка
   - Тестирование компиляторов
   - Plugin тесты
   - Cross-platform тесты

3. **Performance Tests:**
   - Benchmarks
   - Load testing
   - Memory profiling
   - CPU profiling

4. **Security Tests:**
   - Fuzzing
   - Security scanning
   - Dependency audit

---

## Развертывание

### Distribution:

1. **Binary releases:**
   - GitHub Releases
   - Cross-compilation для всех платформ
   - Checksums и signatures

2. **Package managers:**
   - Homebrew (macOS/Linux)
   - APT (Debian/Ubuntu)
   - RPM (RHEL/Fedora)
   - Chocolatey (Windows)
   - Scoop (Windows)

3. **Container images:**
   - Docker Hub
   - GitHub Container Registry
   - Multi-arch images (amd64, arm64)

4. **CI/CD Integration:**
   - GitHub Actions templates
   - GitLab CI examples
   - Jenkins pipelines

---

## Мониторинг и observability

### Metrics:

- Build success/failure rate
- Build duration
- Cache hit rate
- Compiler performance
- Error rates
- Resource usage

### Logging:

- Structured logs
- Log levels (DEBUG, INFO, WARN, ERROR, FATAL)
- Context propagation
- Correlation IDs

### Tracing:

- Build trace
- Compilation steps
- Plugin execution
- Performance bottlenecks

---

## Заключение

Архитектура Buffalo построена на принципах:
- **Модульности:** Каждый компонент независим и заменяем
- **Расширяемости:** Легко добавлять новые языки и функции
- **Производительности:** Оптимизации на всех уровнях
- **Надежности:** Error handling и recovery
- **Простоты:** Понятный и чистый код

Эта архитектура позволяет Buffalo быть мощным, гибким и простым в использовании инструментом для мультиязычной компиляции protobuf/gRPC файлов.
