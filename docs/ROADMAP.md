# 🦬 Buffalo Roadmap — Идеи для развития

> Этот документ содержит идеи и планы по развитию функциональности Buffalo.
> Статус: 📋 Планируется | 🔧 В разработке | ✅ Готово

---

## ✅ Реализовано (v2.0.0)

### buffalo.validate — Нативная система валидации полей
- PGV-inspired аннотации `[(buffalo.validate.rules)...]`
- Кодогенерация для Go, Python, C++, Rust
- 30+ типов правил (gte, lte, email, uuid, pattern и т.д.)
- Embedded proto через `go:embed`
- CLI: `buffalo validate init | rules | list-protos`

---

## 🎯 Приоритет 1 — Ядро системы

### 1. buffalo.mock — Генерация моков для тестирования
**Статус:** 📋 Планируется

Автоматическая генерация мок-серверов и клиентов из proto-определений сервисов.

```protobuf
service UserService {
  rpc GetUser(GetUserRequest) returns (User)
    [(buffalo.mock) = {scenarios: ["success", "not_found", "error"]}];
}
```

**Возможности:**
- Мок-серверы для gRPC с настраиваемыми сценариями
- Генерация фикстур и тестовых данных
- Интеграция с популярными фреймворками (testify, pytest, googletest)
- Поддержка streaming RPC моков
- Запись и воспроизведение реальных запросов (record/replay)

**CLI:**
```bash
buffalo mock generate --service UserService --lang go
buffalo mock server --port 50051 --scenario happy-path
buffalo mock record --target localhost:50051 --output fixtures/
```

---

### 2. buffalo migrate — Версионирование и миграции proto
**Статус:** 📋 Планируется

Отслеживание изменений в proto файлах, обнаружение breaking changes, автогенерация changelog.

```bash
buffalo migrate create add_user_age_field
buffalo migrate check           # проверка на breaking changes
buffalo migrate changelog       # генерация CHANGELOG
buffalo migrate rollback        # откат к предыдущей версии
```

**Возможности:**
- Семантический diff proto файлов (не текстовый)
- Детекция breaking changes по правилам buf/protolock
- Автогенерация CHANGELOG.md из истории изменений
- Lock-файл с хешами proto для CI/CD
- Политики совместимости (WIRE, SOURCE, FULL)

**Правила breaking changes:**
- Удаление/переименование полей, сообщений, сервисов
- Изменение типов полей
- Изменение номеров полей
- Удаление значений enum
- Изменение `oneof` структуры

---

### 3. buffalo diff — Визуализация изменений proto
**Статус:** 📋 Планируется

```bash
buffalo diff                           # текущие изменения
buffalo diff v1.0.0..v2.0.0           # между версиями
buffalo diff --format html --output report.html
buffalo diff --ci --fail-on breaking   # для CI/CD
```

**Возможности:**
- Семантический diff (поля, типы, сервисы)
- Классификация: breaking / non-breaking / deprecation
- Вывод: terminal (colored), HTML, Markdown, JSON
- Интеграция с GitHub PR comments
- Git-aware (diff между коммитами/тегами/ветками)

---

### 4. buffalo security — Анализ безопасности
**Статус:** 📋 Планируется

```bash
buffalo security scan
buffalo security audit --fix
buffalo security report --format sarif
```

**Возможности:**
- Обнаружение чувствительных данных в proto (PII, credentials)
- Проверка: поля `password`, `token`, `secret` без шифрования
- CVE scanning зависимостей (googleapis, grpc)
- Проверка TLS/mTLS конфигураций
- SARIF-формат для GitHub Security tab
- Авто-добавление `[(buffalo.validate.rules).string = {not_empty: true}]` для required полей

---

## 🎯 Приоритет 2 — Экосистема

### 5. buffalo.openapi — Генерация OpenAPI/Swagger
**Статус:** 📋 Планируется

```yaml
plugins:
  - name: openapi
    options:
      version: "3.1.0"
      servers: ["https://api.example.com"]
      auth: ["bearer", "apikey"]
      gateway: grpc-gateway
```

**Возможности:**
- Proto + HTTP аннотации → OpenAPI 3.1 спецификация
- Swagger UI / Redoc генерация
- gRPC-Gateway интеграция
- Автодокументация из proto comments
- Postman/Insomnia коллекции

---

### 6. buffalo.db — ORM аннотации и генерация моделей БД
**Статус:** 📋 Планируется

```protobuf
message User {
  string id = 1 [(buffalo.db) = {primary_key: true, type: "uuid", auto: true}];
  string email = 2 [(buffalo.db) = {unique: true, index: true, not_null: true}];
  string name = 3 [(buffalo.db) = {max_length: 256}];
  int32 age = 4 [(buffalo.db) = {nullable: true, default: 0}];
  google.protobuf.Timestamp created_at = 5 [(buffalo.db) = {auto_now_add: true}];
  google.protobuf.Timestamp updated_at = 6 [(buffalo.db) = {auto_now: true}];
}
```

**Возможности:**
- SQL DDL генерация (PostgreSQL, MySQL, SQLite, CockroachDB)
- ORM модели: GORM (Go), SQLAlchemy (Python), Diesel (Rust), TypeORM (TS)
- Миграции БД из изменений proto
- Связи: one-to-many, many-to-many через proto options
- Seed data генерация

**CLI:**
```bash
buffalo db generate --dialect postgres --output migrations/
buffalo db migrate up
buffalo db seed --count 100
```

---

### 7. buffalo.graphql — Генерация GraphQL схем
**Статус:** 📋 Планируется

```protobuf
message User {
  string id = 1 [(buffalo.graphql.field) = {type: "ID!"}];
  string email = 2;
  repeated Post posts = 3 [(buffalo.graphql) = {resolver: "UserPosts", dataloader: true}];
}

service UserService {
  rpc GetUser(GetUserRequest) returns (User)
    [(buffalo.graphql) = {query: "user", auth: true}];
  rpc CreateUser(CreateUserRequest) returns (User)
    [(buffalo.graphql) = {mutation: "createUser"}];
  rpc UserUpdates(UserFilter) returns (stream User)
    [(buffalo.graphql) = {subscription: "userUpdates"}];
}
```

**Возможности:**
- Proto → GraphQL SDL
- Query/Mutation/Subscription маппинг из RPC
- DataLoader генерация для N+1 проблемы
- Federation v2 поддержка
- Резолверы на основе gRPC клиентов

---

### 8. buffalo.events — Event-driven архитектура
**Статус:** 📋 Планируется

```protobuf
message UserCreated {
  option (buffalo.event) = {
    topic: "users.created"
    schema_registry: true
    partitioning: "user_id"
    retention: "7d"
  };
  string user_id = 1;
  string email = 2;
  google.protobuf.Timestamp created_at = 3;
}
```

**Возможности:**
- Kafka/RabbitMQ/NATS схемы из proto
- Avro/Protobuf Schema Registry интеграция
- Producer/Consumer кодогенерация
- Event sourcing паттерны
- Dead letter queue конфигурация
- CloudEvents совместимость

---

## 🎯 Приоритет 3 — DevOps & Cloud

### 9. buffalo benchmark — Нагрузочное тестирование gRPC
**Статус:** 📋 Планируется

```bash
buffalo benchmark \
  --service UserService.GetUser \
  --target localhost:50051 \
  --requests 10000 \
  --concurrency 100 \
  --duration 60s \
  --report html
```

**Возможности:**
- Встроенный load testing для gRPC сервисов
- Метрики: latency (p50/p95/p99), throughput, error rate
- Сравнение производительности между версиями
- Прогрев (warmup) и ramp-up
- HTML/JSON отчёты с графиками
- CI/CD интеграция (fail on regression)

---

### 10. buffalo cloud deploy — Деплой в облако
**Статус:** 📋 Планируется

```bash
buffalo deploy --provider gcp --region us-central1
buffalo deploy --provider aws --service ecs
buffalo deploy --provider k8s --namespace production
```

**Возможности:**
- Генерация Kubernetes манифестов (Deployment, Service, Ingress)
- Terraform/Pulumi модули
- Cloud Run, Lambda, ECS шаблоны
- Helm chart генерация
- CI/CD пайплайны (GitHub Actions, GitLab CI, Jenkins)
- Service mesh конфигурация (Istio, Linkerd)

---

### 11. buffalo.observability — Телеметрия из коробки
**Статус:** 📋 Планируется

```protobuf
service UserService {
  option (buffalo.observability) = {
    tracing: true
    metrics: true
    logging: "structured"
  };

  rpc GetUser(GetUserRequest) returns (User)
    [(buffalo.observability.span) = {name: "get_user", attributes: ["user_id"]}];
}
```

**Возможности:**
- OpenTelemetry interceptors генерация
- Prometheus метрики из коробки
- Structured logging middleware
- Distributed tracing
- Health check endpoints
- Grafana dashboard шаблоны

---

## 🎯 Приоритет 4 — DX (Developer Experience)

### 12. buffalo repl — Интерактивная gRPC консоль
**Статус:** 📋 Планируется

```bash
buffalo repl --target localhost:50051
> call UserService.GetUser {"user_id": "123"}
> stream OrderService.WatchOrders {"customer_id": "456"}
> describe UserService
> history
```

**Возможности:**
- Интерактивный gRPC клиент с автодополнением
- Server reflection поддержка
- Сохранение истории запросов
- Переменные и скриптинг
- Streaming поддержка (server/client/bidi)
- Экспорт в curl/grpcurl команды

---

### 13. buffalo convert — Конвертация форматов
**Статус:** 📋 Планируется

```bash
buffalo convert proto2-to-proto3 --input old/ --output new/
buffalo convert json-to-proto --input schema.json --output generated.proto
buffalo convert openapi-to-proto --input swagger.yaml --output protos/
buffalo convert graphql-to-proto --input schema.graphql --output protos/
buffalo convert avro-to-proto --input schema.avsc --output protos/
```

**Возможности:**
- proto2 ↔ proto3 миграция
- JSON Schema → proto
- OpenAPI/Swagger → proto
- GraphQL SDL → proto
- Avro → proto
- TypeScript interfaces → proto
- SQL DDL → proto

---

### 14. buffalo docs — Автодокументация
**Статус:** 📋 Планируется

```bash
buffalo docs generate --format html --output docs/api/
buffalo docs serve --port 8080
buffalo docs publish --provider github-pages
```

**Возможности:**
- HTML/Markdown документация из proto comments
- Интерактивная навигация по сервисам и сообщениям
- Диаграммы зависимостей (Mermaid)
- Примеры запросов/ответов
- Версионированная документация
- Поиск по API

---

### 15. buffalo playground — Онлайн-песочница
**Статус:** 📋 Планируется

```bash
buffalo playground --port 3000
```

**Возможности:**
- Web UI для редактирования proto файлов
- Мгновенный предпросмотр сгенерированного кода
- Валидация в реальном времени
- Шаблоны и примеры
- Совместное редактирование

---

## 💡 Дополнительные идеи

### 16. buffalo.rate_limit — Rate limiting аннотации
```protobuf
rpc GetUser(GetUserRequest) returns (User)
  [(buffalo.rate_limit) = {requests: 100, window: "1m", by: "ip"}];
```

### 17. buffalo.cache — Кэширование ответов
```protobuf
rpc GetUser(GetUserRequest) returns (User)
  [(buffalo.cache) = {ttl: "5m", key: "user:{user_id}", invalidate_on: "UpdateUser"}];
```

### 18. buffalo.retry — Политики повторных попыток
```protobuf
rpc GetUser(GetUserRequest) returns (User)
  [(buffalo.retry) = {max_attempts: 3, backoff: "exponential", codes: [UNAVAILABLE, DEADLINE_EXCEEDED]}];
```

### 19. buffalo.auth — Авторизация на уровне методов
```protobuf
rpc DeleteUser(DeleteUserRequest) returns (Empty)
  [(buffalo.auth) = {roles: ["admin"], scopes: ["users:delete"]}];
```

### 20. buffalo.deprecation — Управление устареванием
```protobuf
message OldUser {
  option (buffalo.deprecation) = {
    since: "2.0.0"
    removal: "3.0.0"
    migration: "Use User instead"
    replacement: "User"
  };
}
```

### 21. buffalo.feature_flags — Feature flags
```protobuf
rpc GetUserV2(GetUserRequest) returns (UserV2)
  [(buffalo.feature) = {flag: "new_user_api", default: false, rollout: 10}];
```

### 22. buffalo.transform — Трансформация данных
```protobuf
message UserDTO {
  string full_name = 1 [(buffalo.transform) = {from: "User", map: "first_name + ' ' + last_name"}];
  string masked_email = 2 [(buffalo.transform) = {from: "User.email", mask: "***@{domain}"}];
}
```

### 23. buffalo.i18n — Интернационализация ошибок
```protobuf
message ValidationError {
  string code = 1 [(buffalo.i18n) = {key: "error.validation", locales: ["en", "ru", "zh"]}];
}
```

### 24. buffalo.circuit_breaker — Circuit breaker
```protobuf
rpc ExternalCall(Request) returns (Response)
  [(buffalo.circuit_breaker) = {threshold: 5, timeout: "30s", half_open_requests: 3}];
```

### 25. buffalo.changelog — Автоматический CHANGELOG
```bash
buffalo changelog generate --from v1.0.0 --to v2.0.0
buffalo changelog ci --format github-release
```

---

## 📊 Матрица приоритетов

| Функция | Сложность | Ценность | Приоритет |
|---------|-----------|----------|-----------|
| buffalo.mock | Средняя | Высокая | 🔴 P1 |
| buffalo migrate | Средняя | Высокая | 🔴 P1 |
| buffalo diff | Низкая | Высокая | 🔴 P1 |
| buffalo security | Средняя | Высокая | 🔴 P1 |
| buffalo.openapi | Средняя | Высокая | 🟡 P2 |
| buffalo.db | Высокая | Высокая | 🟡 P2 |
| buffalo.graphql | Высокая | Средняя | 🟡 P2 |
| buffalo.events | Высокая | Средняя | 🟡 P2 |
| buffalo benchmark | Средняя | Средняя | 🟢 P3 |
| buffalo cloud | Высокая | Средняя | 🟢 P3 |
| buffalo.observability | Средняя | Средняя | 🟢 P3 |
| buffalo repl | Низкая | Средняя | 🔵 P4 |
| buffalo convert | Средняя | Средняя | 🔵 P4 |
| buffalo docs | Средняя | Средняя | 🔵 P4 |
| buffalo playground | Высокая | Низкая | 🔵 P4 |

---

*Последнее обновление: Февраль 2026*
