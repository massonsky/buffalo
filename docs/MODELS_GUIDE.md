# Buffalo Models — Руководство по генерации моделей

> `buffalo models` — генерация типизированных моделей из `.proto` файлов для Python, Go, Rust и C++ с поддержкой ORM-фреймворков и автоматической Protobuf-конвертацией.

---

## Содержание

- [Обзор](#обзор)
- [Быстрый старт](#быстрый-старт)
- [Команды CLI](#команды-cli)
  - [generate](#buffalo-models-generate)
  - [list](#buffalo-models-list)
  - [inspect](#buffalo-models-inspect)
  - [check-deps](#buffalo-models-check-deps)
- [Режимы генерации](#режимы-генерации)
  - [Режим аннотаций (по умолчанию)](#режим-аннотаций)
  - [Режим from-proto](#режим-from-proto)
- [Языки и ORM](#языки-и-orm)
  - [Python (Pydantic)](#python-pydantic)
  - [Go (None / GORM / SQLX)](#go)
  - [Rust (None / Diesel)](#rust)
  - [C++ (None)](#c)
- [Структура сгенерированного кода](#структура-сгенерированного-кода)
- [Proto-аннотации buffalo.models](#proto-аннотации)
  - [ModelOptions](#modeloptions)
  - [FieldModelOptions](#fieldmodeloptions)
  - [RelationDef](#relationdef)
- [Конвертация Protobuf (from_proto / to_proto)](#конвертация-protobuf)
- [Маппинг типов](#маппинг-типов)
- [Конфигурация buffalo.yaml](#конфигурация-buffaloyaml)
- [Примеры](#примеры)

---

## Обзор

Buffalo Models читает `.proto` файлы, извлекает определения сообщений (message) и генерирует типизированные модели для целевого языка программирования. Поддерживается два режима:

1. **Аннотированный режим** — обрабатывает только сообщения, содержащие аннотацию `buffalo.models.model`.
2. **From-proto режим** (`--from-proto`) — обрабатывает **все** `message` в `.proto` файлах, автоматически извлекая поля, типы и метаданные.

Для каждого языка генерируются:

| Артефакт | Python | Go | Rust | C++ |
|---|---|---|---|---|
| Base model | `base_model.py` | `base_model.go` | `base_model.rs` | `base_model.h` |
| Модели | `<snake_case>.py` | `<snake_case>.go` | `<snake_case>.rs` | `<snake_case>.h` |
| Init/Index | `__init__.py` | `go.mod` | `mod.rs` | `CMakeLists.txt` |

---

## Быстрый старт

```bash
# 1. Сгенерировать Python-модели из всех proto в директории ./protos
buffalo models generate --lang python --from-proto --proto ./protos --output ./gen/models/python

# 2. Сгенерировать Go-модели с GORM тегами
buffalo models generate --lang go --orm gorm --proto ./protos --output ./gen/models/go

# 3. Посмотреть список обнаруженных моделей
buffalo models list --proto ./protos --from-proto

# 4. Проверить зависимости
buffalo models check-deps --lang python --orm pydantic
```

---

## Команды CLI

### `buffalo models generate`

Генерация моделей из proto-файлов.

```
buffalo models generate [flags]
```

**Алиасы:** `gen`, `g`

| Флаг | Сокращение | По умолчанию | Описание |
|---|---|---|---|
| `--lang` | `-l` | — (обязательный) | Целевой язык: `python`, `go`, `rust`, `cpp` |
| `--orm` | `-r` | `None` | ORM/фреймворк: `pydantic`, `gorm`, `sqlx`, `diesel`, `None` |
| `--proto` | `-p` | `.` | Путь к директории с `.proto` файлами |
| `--output` | `-o` | `generated/models/<lang>` | Директория для сгенерированных файлов |
| `--package` | — | — | Имя пакета для сгенерированного кода |
| `--from-proto` | — | `false` | Генерировать из **всех** proto-сообщений, не только аннотированных |
| `--all` | — | `false` | Генерировать для всех языков из `buffalo.yaml` |

**Примеры:**

```bash
# Python Pydantic v2 (по умолчанию)
buffalo models generate --lang python --proto ./api --output ./models/python

# Python с явной версией Pydantic
buffalo models generate --lang python --orm pydantic@2.0 --proto ./api --output ./models/python

# Go GORM
buffalo models generate --lang go --orm gorm --proto ./api --output ./internal/models

# Go SQLX
buffalo models generate --lang go --orm sqlx --proto ./api --output ./internal/models

# Rust Diesel
buffalo models generate --lang rust --orm diesel@2.1 --proto ./api --output ./src/models

# C++ (без ORM)
buffalo models generate --lang cpp --proto ./api --output ./include/models

# Из ВСЕХ proto-сообщений (без аннотаций)
buffalo models generate --lang python --from-proto --proto ./api --output ./models

# Все языки из buffalo.yaml
buffalo models generate --all
```

---

### `buffalo models list`

Список обнаруженных моделей в proto-файлах.

```bash
buffalo models list --proto ./protos
buffalo models list --from-proto --proto ./protos
```

Вывод:

```
  • User  (3 fields)  table=users  "Пользователь"  [./protos/user.proto]
  • Profile  (5 fields)  table=profiles  [./protos/user.proto]
  • Post  (4 fields)  table=posts  [DEPRECATED]  [./protos/blog.proto]
```

---

### `buffalo models inspect`

Детальная информация о конкретной модели.

```bash
buffalo models inspect User --proto ./protos
buffalo models inspect --from-proto Resolution --proto ./api
```

Вывод:

```
Model: User
  Description: Пользователь системы
  Table: users
  Schema: public
  Fields (3):
    email : string [public]
    display_name : ?string
    is_active : bool
```

---

### `buffalo models check-deps`

Проверка зависимостей для выбранного языка и ORM.

```bash
buffalo models check-deps --lang python --orm pydantic
buffalo models check-deps --lang go --orm gorm
buffalo models check-deps --lang rust --orm diesel
```

Вывод (если всё ОК):

```
✓ No dependency issues for python/pydantic
```

---

## Режимы генерации

### Режим аннотаций

По умолчанию Buffalo обрабатывает только сообщения, явно помеченные аннотацией `buffalo.models.model`:

```protobuf
import "buffalo/models/models.proto";

message User {
  option (buffalo.models.model) = {
    name: "User"
    table_name: "users"
    description: "Пользователь"
    generate: ["model"]
  };

  string email = 1 [(buffalo.models.field) = {
    unique: true
    max_length: 255
  }];
}
```

Этот режим даёт полный контроль над тем, какие модели генерируются и с какими опциями.

### Режим from-proto

Флаг `--from-proto` включает режим, в котором **все** `message` из `.proto` файлов становятся моделями — без необходимости добавлять аннотации. Это удобно для:

- Быстрой генерации моделей из существующих proto-схем
- Проектов, где proto является единственным источником истины для типов
- Начального прототипирования

```bash
buffalo models generate --lang python --from-proto --proto ./api --output ./models
```

В этом режиме:

- Имя модели = имя `message` (в PascalCase)
- Имя файла = `snake_case` от имени сообщения
- Поля извлекаются из proto-определения (тип, имя, `repeated`, `optional`)
- `optional` поля автоматически получают `Nullable = true`
- Поддерживаются well-known типы Google (`Timestamp` → `datetime`, `Duration` → `timedelta` и т.д.)
- Кросс-пакетные типы получают автоматический импорт

Режим можно также включить глобально через `buffalo.yaml`:

```yaml
models:
  enabled: true
  generate_models_from_proto: true
```

---

## Языки и ORM

### Python (Pydantic)

Python генерирует модели на основе **Pydantic v2** (по умолчанию). Поддерживается также Pydantic v1.

```bash
buffalo models generate --lang python --output ./models
# или явно:
buffalo models generate --lang python --orm pydantic@2.0 --output ./models
```

**Что генерируется:**

| Файл | Описание |
|---|---|
| `base_model.py` | `BaseModel` (Pydantic) + `ProtoBaseModel` с `from_proto()` / `to_proto()` |
| `<model>.py` | Класс модели, наследующий `ProtoBaseModel` |
| `__init__.py` | Реэкспорт всех моделей + `__all__` |

**Возможности:**

- Автоматический `from_proto()` / `to_proto()` с `@override` декоратором
- Кросс-пакетные импорты для custom-типов (`from .boat_data import BoatData`)
- Well-known type маппинг (`Timestamp` → `datetime`, `Duration` → `timedelta`)
- `Field(...)` с валидацией: `max_length`, `min_length`, `default`, `description`, `examples`
- `ConfigDict` с `json_schema_extra` для табличных метаданных
- Поддержка `Optional`, `List`, `Dict` типов
- Пометки `# [readonly]`, `# [sensitive]`, `# DEPRECATED` в комментариях

**Пример сгенерированного кода:**

```python
# Auto-generated by buffalo-models (pydantic). DO NOT EDIT.
from __future__ import annotations

from typing import Any, ClassVar, Dict, List, Optional, Type, TypeVar

try:
    from typing import Self, override
except ImportError:
    from typing_extensions import Self, override

T = TypeVar("T")

from datetime import datetime

from pydantic import ConfigDict, Field

from .base_model import ProtoBaseModel
from .gps_data import GpsData


class BoatTelemetry(ProtoBaseModel):
    """Телеметрия судна

    Table: boat_telemetry"""

    model_config = ConfigDict(
        json_schema_extra={
            "tablename": "boat_telemetry",
        },
    )

    boat_id: str = Field(default="")
    timestamp: datetime = Field(default=None)
    gps: GpsData = Field(default=None)
    speed_knots: float = Field(default=0.0)
    heading: float = Field(default=0.0)

    proto_class: ClassVar[Type[Any] | None] = None

    @classmethod
    @override
    def from_proto(cls, proto_msg: Any) -> Self:
        """Override-friendly protobuf -> model conversion."""
        return super().from_proto(proto_msg)

    @override
    def to_proto(self, proto_class: Type[T] | None = None) -> Any:
        """Override-friendly model -> protobuf conversion."""
        return super().to_proto(proto_class=proto_class)
```

---

### Go

Go поддерживает три варианта ORM:

| ORM | Команда | Описание |
|---|---|---|
| `None` | `--orm None` (по умолчанию) | Чистые Go-структуры |
| `gorm` | `--orm gorm` | Структуры с `gorm:"..."` тегами |
| `sqlx` | `--orm sqlx` | Структуры с `db:"..."` тегами |

```bash
buffalo models generate --lang go --orm gorm --proto ./api --output ./internal/models
```

**Что генерируется:**

| Файл | Описание |
|---|---|
| `base_model.go` | Базовая структура `BaseModel` с полями ID, timestamps |
| `<model>.go` | Структура модели с JSON/ORM тегами |
| `go.mod` | Файл модуля Go с зависимостями (`protobuf`, `uuid`) |

---

### Rust

Rust поддерживает два варианта:

| ORM | Команда | Описание |
|---|---|---|
| `None` | `--orm None` (по умолчанию) | Структуры с `#[derive(Debug, Clone, Serialize, Deserialize)]` |
| `diesel` | `--orm diesel@2.1` | Структуры с `#[derive(Queryable, Insertable)]` + `#[diesel(table_name = ...)]` |

```bash
buffalo models generate --lang rust --orm diesel --proto ./api --output ./src/models
```

**Что генерируется:**

| Файл | Описание |
|---|---|
| `base_model.rs` | Базовая структура `BaseModel` |
| `<model>.rs` | Структура модели |
| `mod.rs` | Реэкспорт всех модулей |

---

### C++

C++ поддерживает один вариант — чистые структуры:

```bash
buffalo models generate --lang cpp --proto ./api --output ./include/models
```

**Что генерируется:**

| Файл | Описание |
|---|---|
| `base_model.h` | Базовый класс `BaseModel` с nlohmann/json сериализацией |
| `<model>.h` | Заголовочный файл модели |
| `CMakeLists.txt` | CMake конфигурация с C++17, Protobuf и nlohmann_json |

---

## Структура сгенерированного кода

Типичная структура после генерации для Python:

```
generated/models/python/
├── __init__.py           # Реэкспорт всех моделей
├── base_model.py         # BaseModel + ProtoBaseModel
├── user.py               # class User(ProtoBaseModel)
├── profile.py            # class Profile(ProtoBaseModel)
├── post.py               # class Post(ProtoBaseModel)
├── boat_telemetry.py     # class BoatTelemetry(ProtoBaseModel)
└── gps_data.py           # class GpsData(ProtoBaseModel)
```

Для Go:

```
generated/models/go/
├── go.mod                # module + dependencies
├── base_model.go         # type BaseModel struct
├── user.go               # type User struct
├── profile.go            # type Profile struct
└── post.go               # type Post struct
```

---

## Proto-аннотации

### ModelOptions

Полный набор опций уровня модели (сообщения):

```protobuf
message Account {
  option (buffalo.models.model) = {
    name: "AccountModel"        // Имя модели (по умолчанию = имя message)
    table_name: "accounts"      // Имя таблицы
    schema: "iam"               // SQL-схема
    description: "Аккаунт"     // Описание
    tags: ["iam", "billing"]    // Теги
    abstract: false             // Абстрактная модель (не генерируется как отдельный файл)
    extends: "AuditableEntity"  // Наследование
    mixins: ["SoftDeleteMixin"] // Миксины
    soft_delete: true           // Мягкое удаление
    timestamps: true            // Автоматические timestamps
    deprecated: false           // Пометить как устаревшее
    deprecated_message: ""      // Сообщение об устаревании
    generate: ["model"]         // Артефакты для генерации

    // Индексы
    indexes: {
      name: "idx_tenant_status"
      columns: ["tenant_id", "status"]
      unique: false
      type: BTREE
      where: "deleted_at IS NULL"
    }

    // Уникальные ограничения
    uniques: {
      name: "uq_tenant_name"
      columns: ["tenant_id", "name"]
    }

    // Check-ограничения
    checks: {
      name: "chk_status"
      expression: "status IN ('active','suspended')"
    }
  };
}
```

### FieldModelOptions

Полный набор опций уровня поля:

```protobuf
string email = 5 [(buffalo.models.field) = {
  // Идентификация
  alias: "user_email"           // Алиас для сериализации
  description: "Email address"  // Описание

  // Ограничения
  primary_key: true             // Первичный ключ
  auto_increment: true          // Автоинкремент
  auto_generate: true           // Авто-генерация значения
  nullable: true                // Nullable
  unique: true                  // Уникальность
  default_value: "guest"        // Значение по умолчанию
  max_length: 255               // Макс. длина
  min_length: 3                 // Мин. длина
  precision: 18                 // Точность (decimal)
  scale: 6                      // Масштаб (decimal)

  // Типы
  custom_type: "decimal.Decimal" // Кастомный тип
  db_type: "VARCHAR(255)"        // Тип в БД

  // Видимость / поведение
  visibility: PUBLIC             // PUBLIC, PRIVATE, INTERNAL, EXTERNAL
  behavior: READONLY             // READONLY, WRITEONLY, COMPUTED, IMMUTABLE
  sensitive: true                // Чувствительные данные (маскируются)
  deprecated: true               // Устаревшее поле
  deprecated_message: "Use X"    // Сообщение

  // Индексация
  index: true                    // Создать индекс
  index_type: BTREE              // Тип индекса

  // Сериализация
  json_name: "email"             // Имя в JSON
  xml_name: "Email"              // Имя в XML
  omit_empty: true               // Не сериализовать пустое значение
  ignore: false                  // Полностью игнорировать поле

  // Документация
  example: "user@example.com"    // Пример значения
  comment: "Shown in profile"    // Комментарий
  tags: ["identity"]             // Теги
  metadata: { key: "ui.group" value: "identity" }  // Произвольные метаданные
}];
```

### RelationDef

Связи между моделями:

```protobuf
// Belongs To
string owner_id = 6 [(buffalo.models.field) = {
  relation: {
    type: BELONGS_TO
    model: "Owner"
    foreign_key: "owner_id"
    references: "id"
    on_delete: CASCADE
    on_update: NO_ACTION
    eager: true
    inverse_of: "items"
  }
}];

// Has Many
repeated Post posts = 4 [(buffalo.models.field) = {
  relation: {
    type: HAS_MANY
    model: "Post"
    foreign_key: "author_id"
    references: "id"
    on_delete: CASCADE
    inverse_of: "author"
  }
}];

// Many to Many
repeated Role roles = 5 [(buffalo.models.field) = {
  relation: {
    type: MANY_TO_MANY
    model: "Role"
    join_table: "user_roles"
    foreign_key: "user_id"
    references: "id"
    through: "UserRole"
    inverse_of: "users"
  }
}];
```

---

## Конвертация Protobuf

Python-генератор (Pydantic) создаёт модели с встроенной Protobuf-конвертацией.

### BaseModel / ProtoBaseModel

В `base_model.py` генерируется:

- `BaseModel` — базовый Pydantic-класс с полями `id`, `created_at`, `updated_at`, `deleted_at`
- `ProtoBaseModel(BaseModel)` — расширение с методами `from_proto()` и `to_proto()`

### from_proto (classmethod)

Конвертирует Protobuf-сообщение в Pydantic-модель:

```python
from mypackage.models import User
from mypackage.proto import user_pb2

proto_msg = user_pb2.User(email="test@example.com", display_name="Neo")
user = User.from_proto(proto_msg)
print(user.email)  # "test@example.com"
```

### to_proto

Конвертирует Pydantic-модель обратно в Protobuf:

```python
user = User(email="test@example.com", display_name="Neo")
user.proto_class = user_pb2.User  # или: передать явно
proto_msg = user.to_proto()
# или:
proto_msg = user.to_proto(proto_class=user_pb2.User)
```

### @override декоратор

Каждая сгенерированная модель переопределяет `from_proto` / `to_proto` с `@override`, чтобы вы могли добавить кастомную логику в подклассе:

```python
class User(ProtoBaseModel):
    @classmethod
    @override
    def from_proto(cls, proto_msg: Any) -> Self:
        """Override-friendly protobuf -> model conversion."""
        return super().from_proto(proto_msg)

    @override
    def to_proto(self, proto_class: Type[T] | None = None) -> Any:
        """Override-friendly model -> protobuf conversion."""
        return super().to_proto(proto_class=proto_class)
```

---

## Маппинг типов

### Proto → Python

| Proto тип | Python тип |
|---|---|
| `string` | `str` |
| `int32`, `sint32`, `sfixed32` | `int` |
| `int64`, `sint64`, `sfixed64` | `int` |
| `uint32`, `fixed32` | `int` |
| `uint64`, `fixed64` | `int` |
| `float` | `float` |
| `double` | `float` |
| `bool` | `bool` |
| `bytes` | `bytes` |
| `google.protobuf.Timestamp` | `datetime` |
| `google.protobuf.Duration` | `timedelta` |
| `google.protobuf.StringValue` | `Optional[str]` |
| `google.protobuf.Int32Value` | `Optional[int]` |
| `google.protobuf.BoolValue` | `Optional[bool]` |
| `google.protobuf.Struct` | `Dict[str, Any]` |
| `google.protobuf.Any` | `Any` |
| `map<K, V>` | `Dict[K, V]` |
| `repeated T` | `List[T]` |
| `optional T` | `Optional[T]` (Nullable=true) |
| Custom message | `PascalCase` (с авто-импортом) |

### Proto → Go

| Proto тип | Go (None) | Go (GORM) |
|---|---|---|
| `string` | `string` | `string` + `gorm:"..."` |
| `int32` | `int32` | `int32` |
| `int64` | `int64` | `int64` |
| `float` | `float32` | `float32` |
| `double` | `float64` | `float64` |
| `bool` | `bool` | `bool` |
| `bytes` | `[]byte` | `[]byte` |
| `Timestamp` | `time.Time` | `time.Time` |

### Proto → Rust

| Proto тип | Rust |
|---|---|
| `string` | `String` |
| `int32` | `i32` |
| `int64` | `i64` |
| `uint32` | `u32` |
| `uint64` | `u64` |
| `float` | `f32` |
| `double` | `f64` |
| `bool` | `bool` |
| `bytes` | `Vec<u8>` |

### Proto → C++

| Proto тип | C++ |
|---|---|
| `string` | `std::string` |
| `int32` | `int32_t` |
| `int64` | `int64_t` |
| `uint32` | `uint32_t` |
| `float` | `float` |
| `double` | `double` |
| `bool` | `bool` |
| `bytes` | `std::vector<uint8_t>` |

---

## Конфигурация buffalo.yaml

Генерацию можно настроить через `buffalo.yaml` в корне проекта:

```yaml
version: 1.0

# Настройки генерации моделей
models:
  enabled: true
  # Генерировать из ВСЕХ proto-сообщений (не только аннотированных)
  generate_models_from_proto: true

# Языковые настройки
languages:
  python:
    enabled: true
    output: ./generated/python
    options:
      orm: pydantic@2.0
      models_output: ./generated/models/python

  go:
    enabled: true
    output: ./generated/go
    options:
      orm: gorm
      models_output: ./generated/models/go

  rust:
    enabled: false

  cpp:
    enabled: false
```

При использовании `buffalo.yaml` можно генерировать всё одной командой:

```bash
buffalo models generate --all
```

---

## Примеры

### Минимальный пример (аннотации)

```protobuf
syntax = "proto3";
package myapp;

import "buffalo/models/models.proto";

message User {
  option (buffalo.models.model) = {
    name: "User"
    table_name: "users"
    generate: ["model"]
  };

  string id = 1 [(buffalo.models.field) = { primary_key: true }];
  string email = 2 [(buffalo.models.field) = { unique: true, max_length: 255 }];
  string name = 3;
  bool is_active = 4 [(buffalo.models.field) = { default_value: "true" }];
}
```

```bash
buffalo models generate --lang python --orm pydantic --proto . --output ./models
```

### Минимальный пример (from-proto)

Обычный `.proto` файл **без аннотаций**:

```protobuf
syntax = "proto3";
package myapp;

message User {
  string id = 1;
  string email = 2;
  string name = 3;
  bool is_active = 4;
}

message Post {
  string id = 1;
  string title = 2;
  string content = 3;
  string author_id = 4;
  optional string published_at = 5;
}
```

```bash
buffalo models generate --lang python --from-proto --proto . --output ./models
```

Результат — готовые Pydantic-модели `User` и `Post` с `from_proto()` / `to_proto()`.

### Мультиязычная генерация

```bash
# Python
buffalo models generate --lang python --from-proto --proto ./api --output ./gen/python

# Go (GORM)
buffalo models generate --lang go --orm gorm --from-proto --proto ./api --output ./gen/go

# Rust (Diesel)
buffalo models generate --lang rust --orm diesel --from-proto --proto ./api --output ./gen/rust

# C++
buffalo models generate --lang cpp --from-proto --proto ./api --output ./gen/cpp
```

### Пример с наследованием

```protobuf
message AuditableEntity {
  option (buffalo.models.model) = {
    abstract: true
    description: "Базовая модель с аудитом"
  };

  string created_by = 1;
  string updated_by = 2;
}

message Account {
  option (buffalo.models.model) = {
    extends: "AuditableEntity"
    table_name: "accounts"
  };

  string id = 1 [(buffalo.models.field) = { primary_key: true }];
  string name = 2;
}
```

---

## См. также

- [Proto-примеры](../examples/models/) — полные примеры аннотаций
- [Roadmap моделей](MODELS_NEXT_STEPS.md) — планы развития
- [Краткий справочник](readme/MODELS.md) — cheat sheet
