# 🦬 buffalo.db — ORM аннотации, SQL DDL, миграции БД

> **Версия:** 1.21.0  
> **Статус:** 📋 Планируется  
> **Дата:** Февраль 2026  

---

## 📌 Обзор

`buffalo.db` — система декларативных proto-аннотаций для генерации:
- **Типизированных моделей** (data classes, structs, dataclasses)
- **ORM-моделей** с привязкой к конкретному фреймворку (pydantic, sqlalchemy, gorm, diesel и т.д.)
- **SQL DDL** (CREATE TABLE, ALTER TABLE, CREATE INDEX)
- **Миграций БД** (версионированные дельты схемы)
- **Базовых классов**, от которых наследуются все генерируемые модели

Все модели наследуются от генерируемого `BaseModel`, что обеспечивает единую точку расширения.

---

## 🏗️ Архитектура

### Диаграмма компонентов

```
┌─────────────────────────────────────────────────────────────────────┐
│                         buffalo.yaml                                │
│  languages:                                                         │
│    python:                                                          │
│      orm: true                                                      │
│      orm_plugin: "pydantic@2.0"                                     │
└──────────────┬──────────────────────────────────────────────────────┘
               │
               ▼
┌──────────────────────────┐     ┌────────────────────────────────┐
│   Proto с аннотациями    │────▶│   internal/db/parser.go        │
│   [(buffalo.db.*) ...]   │     │   Regex-парсинг аннотаций      │
└──────────────────────────┘     └────────────┬───────────────────┘
                                              │
                                              ▼
                                 ┌────────────────────────────────┐
                                 │   internal/db/types.go         │
                                 │   TableDef, ColumnDef,         │
                                 │   IndexDef, RelationDef        │
                                 └────────────┬───────────────────┘
                                              │
                              ┌───────────────┼───────────────┐
                              ▼               ▼               ▼
                    ┌─────────────┐  ┌─────────────┐  ┌───────────┐
                    │  codegen.go │  │  ddl.go     │  │ migrate.go│
                    │  ORM модели │  │  SQL DDL    │  │ Миграции  │
                    └─────────────┘  └─────────────┘  └───────────┘
                          │
          ┌───────────────┼───────────────┐
          ▼               ▼               ▼
    ┌──────────┐   ┌──────────┐   ┌──────────┐
    │  Python  │   │    Go    │   │   Rust   │
    │  codegen │   │  codegen │   │  codegen │
    └──────────┘   └──────────┘   └──────────┘
```

---

## ⚙️ Конфигурация (buffalo.yaml)

### Формат поля `orm_plugin`

```
orm_plugin: "PLUGIN_NAME[@VERSION]"
```

| Значение | Описание |
|----------|----------|
| `default` | Синоним `None` (чистый язык, без зависимостей) |
| `None` | Генерация кода средствами стандартной библиотеки |
| `pydantic` | Последняя версия pydantic (должна быть установлена) |
| `pydantic@2.0` | Конкретная версия pydantic (проверяется при сборке) |
| `sqlalchemy` | SQLAlchemy ORM модели |
| `sqlalchemy@2.0` | Конкретная версия SQLAlchemy |
| `gorm` | Go ORM (для Go) |
| `diesel` | Diesel (для Rust) |

### Пример конфигурации

```yaml
project:
  name: my-service
  version: 1.21.0

languages:
  python:
    enabled: true
    package: myservice_pb
    orm: true
    orm_plugin: "pydantic@2.0"   # или "None", "sqlalchemy@2.0"

  go:
    enabled: true
    module: github.com/myorg/myservice
    orm: true
    orm_plugin: "gorm"           # или "None", "ent"

  rust:
    enabled: true
    orm: true
    orm_plugin: "diesel"         # или "None", "sea-orm"

  cpp:
    enabled: true
    orm: true
    orm_plugin: "None"           # C++ — только чистые структуры

# Настройки DB-генерации
db:
  # Генерация SQL DDL
  ddl:
    enabled: true
    dialect: "postgresql"        # postgresql, mysql, sqlite
    output: "./migrations/ddl"
  
  # Генерация миграций
  migrations:
    enabled: true
    format: "sql"                # sql, go-migrate, alembic
    output: "./migrations"
    naming: "timestamp"          # timestamp, sequential
  
  # Базовая модель
  base_model:
    # Поля, добавляемые в каждую модель
    fields:
      - name: "id"
        type: "uuid"
        primary_key: true
        auto_generate: true
      - name: "created_at"
        type: "timestamp"
        auto_now_add: true
      - name: "updated_at"
        type: "timestamp"
        auto_now: true
      - name: "deleted_at"
        type: "timestamp"
        nullable: true
        comment: "soft delete"
```

### Матрица ORM-плагинов по языкам

| Язык | `None` / `default` | Плагин 1 | Плагин 2 | Плагин 3 |
|------|-------------------|----------|----------|----------|
| **Python** | `dataclass` + type hints | `pydantic` (BaseModel) | `sqlalchemy` (DeclarativeBase) | `tortoise` |
| **Go** | plain struct + tags | `gorm` (gorm.Model) | `ent` | `sqlx` (db tags) |
| **Rust** | plain struct + derive | `diesel` (Queryable, Insertable) | `sea-orm` (Entity) | — |
| **C++** | plain struct | — | — | — |

---

## 📝 Proto аннотации

### Embedded proto файл

```
internal/embedded/proto/buffalo/db/db.proto
```

Extension numbers: **53000–53099** (не пересекаются с validate: 51xxx, permissions: 52xxx).

### Определение аннотаций

```protobuf
syntax = "proto3";
package buffalo.db;

import "google/protobuf/descriptor.proto";

// ═══════════════════════════════════════════
//  Message-level: таблица
// ═══════════════════════════════════════════

message TableOptions {
  string name = 1;              // имя таблицы (по умолчанию = snake_case от message)
  string schema = 2;            // схема БД (например "public")
  string engine = 3;            // движок (InnoDB, etc.)
  string comment = 4;           // комментарий к таблице
  repeated IndexDef indexes = 5;
  repeated UniqueDef uniques = 6;
  repeated CheckDef checks = 7;
  bool soft_delete = 8;         // автоматический soft delete
  string table_prefix = 9;     // префикс таблицы
}

message IndexDef {
  string name = 1;
  repeated string columns = 2;
  bool unique = 3;
  string type = 4;              // btree, hash, gin, gist
  string where = 5;            // partial index condition
}

message UniqueDef {
  string name = 1;
  repeated string columns = 2;
}

message CheckDef {
  string name = 1;
  string expression = 2;
}

extend google.protobuf.MessageOptions {
  optional TableOptions table = 53000;
}

// ═══════════════════════════════════════════
//  Field-level: колонка
// ═══════════════════════════════════════════

message ColumnOptions {
  string name = 1;              // имя колонки (по умолчанию = snake_case поля)
  string db_type = 2;          // явный SQL тип: VARCHAR(255), JSONB, etc.
  bool primary_key = 3;
  bool auto_increment = 4;
  bool nullable = 5;
  string default_value = 6;    // SQL default: NOW(), gen_random_uuid()
  bool unique = 7;
  string comment = 8;
  int32 size = 9;              // размер для VARCHAR, CHAR
  int32 precision = 10;        // для DECIMAL
  int32 scale = 11;            // для DECIMAL
  string collation = 12;
  bool index = 13;             // создать одиночный индекс
  string index_type = 14;     // btree, hash, gin, gist
  bool ignore = 15;           // не создавать колонку в БД
  string sequence = 16;       // имя sequence
}

extend google.protobuf.FieldOptions {
  optional ColumnOptions column = 53001;
}

// ═══════════════════════════════════════════
//  Field-level: отношения (FK, relations)
// ═══════════════════════════════════════════

message RelationOptions {
  string type = 1;             // belongs_to, has_one, has_many, many_to_many
  string model = 2;           // целевой message/таблица
  string foreign_key = 3;     // FK поле
  string references = 4;      // ссылка на поле (по умолчанию "id")
  string join_table = 5;      // для many_to_many
  string on_delete = 6;       // CASCADE, SET NULL, RESTRICT, NO ACTION
  string on_update = 7;       // CASCADE, SET NULL, RESTRICT, NO ACTION
  bool eager = 8;             // автоматическая загрузка
  string through = 9;         // промежуточная модель для m2m
}

extend google.protobuf.FieldOptions {
  optional RelationOptions relation = 53002;
}
```

### Пример использования

```protobuf
syntax = "proto3";
package myservice;

import "buffalo/db/db.proto";
import "buffalo/validate/validate.proto";

// ═══════════════════════════════════════════
//  User — таблица пользователей
// ═══════════════════════════════════════════
message User {
  option (buffalo.db.table) = {
    name: "users"
    schema: "public"
    comment: "Пользователи системы"
    soft_delete: true
    indexes: [
      { name: "idx_users_email", columns: ["email"], unique: true },
      { name: "idx_users_status", columns: ["status", "created_at"] }
    ]
    checks: [
      { name: "chk_age", expression: "age >= 0 AND age <= 200" }
    ]
  };

  string id = 1 [
    (buffalo.db.column) = {
      primary_key: true
      db_type: "UUID"
      default_value: "gen_random_uuid()"
    }
  ];

  string email = 2 [
    (buffalo.db.column) = { unique: true, size: 255 },
    (buffalo.validate.rules).string = { email: true, not_empty: true }
  ];

  string name = 3 [
    (buffalo.db.column) = { size: 100, nullable: false }
  ];

  int32 age = 4 [
    (buffalo.db.column) = { nullable: true },
    (buffalo.validate.rules).int32 = { gte: 0, lte: 200 }
  ];

  string status = 5 [
    (buffalo.db.column) = { default_value: "'active'", size: 20 }
  ];

  // Отношение: User has_many Orders
  repeated Order orders = 6 [
    (buffalo.db.relation) = {
      type: "has_many"
      model: "Order"
      foreign_key: "user_id"
    }
  ];

  // Отношение: User belongs_to Department
  string department_id = 7 [
    (buffalo.db.relation) = {
      type: "belongs_to"
      model: "Department"
      foreign_key: "department_id"
      on_delete: "SET NULL"
    }
  ];
}

// ═══════════════════════════════════════════
//  Order — таблица заказов
// ═══════════════════════════════════════════
message Order {
  option (buffalo.db.table) = {
    name: "orders"
    indexes: [
      { name: "idx_orders_user", columns: ["user_id"] },
      { name: "idx_orders_date", columns: ["created_at"], type: "btree" }
    ]
  };

  string id = 1 [(buffalo.db.column) = { primary_key: true, db_type: "UUID" }];
  string user_id = 2 [(buffalo.db.column) = { index: true }];
  double total = 3 [(buffalo.db.column) = { db_type: "DECIMAL(10,2)" }];
  string status = 4 [(buffalo.db.column) = { default_value: "'pending'" }];
}
```

---

## 🧬 Генерация кода

### Принцип наследования

```
BaseModel (генерируется buffalo)
  ├── User (генерируется из proto + аннотаций)
  ├── Order
  └── Department
```

Все модели **обязательно** наследуются от `BaseModel`. `BaseModel` содержит общие поля (id, created_at, updated_at, deleted_at) и общую логику (валидация, сериализация, repr).

---

### Python

#### `None` / `default` — чистый Python

```python
# Code generated by buffalo-db. DO NOT EDIT.
# Source: protos/user.proto

from __future__ import annotations
from dataclasses import dataclass, field
from datetime import datetime
from typing import Optional, List
from uuid import UUID, uuid4


@dataclass
class BaseModel:
    """Base model for all buffalo-db generated models."""
    id: UUID = field(default_factory=uuid4)
    created_at: datetime = field(default_factory=datetime.utcnow)
    updated_at: datetime = field(default_factory=datetime.utcnow)
    deleted_at: Optional[datetime] = None

    def to_dict(self) -> dict:
        """Convert model to dictionary."""
        ...

    def from_dict(cls, data: dict) -> "BaseModel":
        """Create model from dictionary."""
        ...


@dataclass
class User(BaseModel):
    """Пользователи системы"""
    __tablename__ = "users"
    __schema__ = "public"

    email: str = ""
    name: str = ""
    age: Optional[int] = None
    status: str = "active"
    department_id: Optional[str] = None
    orders: List["Order"] = field(default_factory=list)
```

#### `pydantic` — Pydantic v2

```python
# Code generated by buffalo-db. DO NOT EDIT.

from __future__ import annotations
from datetime import datetime
from typing import Optional, List
from uuid import UUID, uuid4
from pydantic import BaseModel as PydanticBaseModel, Field, ConfigDict


class BaseModel(PydanticBaseModel):
    """Base model for all buffalo-db generated models."""
    model_config = ConfigDict(
        from_attributes=True,
        populate_by_name=True,
        json_schema_extra={"generator": "buffalo-db"}
    )

    id: UUID = Field(default_factory=uuid4)
    created_at: datetime = Field(default_factory=datetime.utcnow)
    updated_at: datetime = Field(default_factory=datetime.utcnow)
    deleted_at: Optional[datetime] = None


class User(BaseModel):
    """Пользователи системы
    
    Table: public.users
    """
    model_config = ConfigDict(
        json_schema_extra={
            "tablename": "users",
            "schema": "public",
        }
    )

    email: str = Field(default="", max_length=255, json_schema_extra={"unique": True})
    name: str = Field(default="", max_length=100)
    age: Optional[int] = Field(default=None, ge=0, le=200)
    status: str = Field(default="active", max_length=20)
    department_id: Optional[str] = None
    orders: List["Order"] = Field(default_factory=list)
```

#### `pydantic@1.10` — Pydantic v1

```python
from pydantic import BaseModel as PydanticBaseModel, Field, validator

class BaseModel(PydanticBaseModel):
    class Config:
        orm_mode = True
        # v1-style config

    id: UUID = Field(default_factory=uuid4)
    ...
```

#### `sqlalchemy` — SQLAlchemy ORM

```python
# Code generated by buffalo-db. DO NOT EDIT.

from sqlalchemy import Column, String, Integer, DateTime, ForeignKey, CheckConstraint, Index
from sqlalchemy.dialects.postgresql import UUID
from sqlalchemy.orm import DeclarativeBase, Mapped, mapped_column, relationship
from datetime import datetime
import uuid


class BaseModel(DeclarativeBase):
    """Base model for all buffalo-db generated models."""
    id: Mapped[uuid.UUID] = mapped_column(UUID, primary_key=True, default=uuid.uuid4)
    created_at: Mapped[datetime] = mapped_column(DateTime, default=datetime.utcnow)
    updated_at: Mapped[datetime] = mapped_column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)
    deleted_at: Mapped[datetime | None] = mapped_column(DateTime, nullable=True)


class User(BaseModel):
    __tablename__ = "users"
    __table_args__ = (
        Index("idx_users_email", "email", unique=True),
        Index("idx_users_status", "status", "created_at"),
        CheckConstraint("age >= 0 AND age <= 200", name="chk_age"),
        {"schema": "public", "comment": "Пользователи системы"},
    )

    email: Mapped[str] = mapped_column(String(255), unique=True)
    name: Mapped[str] = mapped_column(String(100), nullable=False)
    age: Mapped[int | None] = mapped_column(Integer, nullable=True)
    status: Mapped[str] = mapped_column(String(20), server_default="active")
    department_id: Mapped[str | None] = mapped_column(ForeignKey("departments.id", ondelete="SET NULL"))
    
    orders: Mapped[list["Order"]] = relationship(back_populates="user")
```

---

### Go

#### `None` — чистые структуры

```go
// Code generated by buffalo-db. DO NOT EDIT.
package models

import (
    "time"
    "github.com/google/uuid"
)

// BaseModel is the base for all buffalo-db generated models.
type BaseModel struct {
    ID        uuid.UUID  `json:"id"`
    CreatedAt time.Time  `json:"created_at"`
    UpdatedAt time.Time  `json:"updated_at"`
    DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

// User — Пользователи системы
// Table: public.users
type User struct {
    BaseModel
    Email        string   `json:"email"`
    Name         string   `json:"name"`
    Age          *int32   `json:"age,omitempty"`
    Status       string   `json:"status"`
    DepartmentID *string  `json:"department_id,omitempty"`
    Orders       []Order  `json:"orders,omitempty"`
}
```

#### `gorm` — GORM

```go
// Code generated by buffalo-db. DO NOT EDIT.
package models

import (
    "time"
    "gorm.io/gorm"
    "github.com/google/uuid"
)

// BaseModel is the GORM base for all buffalo-db generated models.
type BaseModel struct {
    ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
    CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
    UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
    DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// User — Пользователи системы
// Table: public.users
type User struct {
    BaseModel
    Email        string  `gorm:"uniqueIndex:idx_users_email;size:255" json:"email"`
    Name         string  `gorm:"size:100;not null" json:"name"`
    Age          *int32  `gorm:"check:chk_age,age >= 0 AND age <= 200" json:"age,omitempty"`
    Status       string  `gorm:"size:20;default:'active';index:idx_users_status" json:"status"`
    DepartmentID *string `gorm:"index" json:"department_id,omitempty"`
    Orders       []Order `gorm:"foreignKey:UserID" json:"orders,omitempty"`
}

func (User) TableName() string { return "users" }
```

#### `sqlx` — sqlx tags

```go
type User struct {
    BaseModel
    Email string `db:"email" json:"email"`
    Name  string `db:"name" json:"name"`
    ...
}
```

---

### Rust

#### `None` — чистые структуры

```rust
// Code generated by buffalo-db. DO NOT EDIT.

use chrono::{DateTime, Utc};
use uuid::Uuid;
use serde::{Serialize, Deserialize};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BaseModel {
    pub id: Uuid,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
    pub deleted_at: Option<DateTime<Utc>>,
}

/// Пользователи системы
/// Table: public.users
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct User {
    #[serde(flatten)]
    pub base: BaseModel,
    pub email: String,
    pub name: String,
    pub age: Option<i32>,
    pub status: String,
    pub department_id: Option<String>,
}
```

#### `diesel` — Diesel ORM

```rust
use diesel::prelude::*;

#[derive(Queryable, Selectable, Identifiable, Debug)]
#[diesel(table_name = users)]
#[diesel(check_for_backend(diesel::pg::Pg))]
pub struct User {
    pub id: uuid::Uuid,
    pub email: String,
    pub name: String,
    pub age: Option<i32>,
    pub status: String,
    ...
}

#[derive(Insertable)]
#[diesel(table_name = users)]
pub struct NewUser {
    pub email: String,
    pub name: String,
    ...
}
```

---

### C++

#### `None` — структуры

```cpp
// Code generated by buffalo-db. DO NOT EDIT.
#pragma once

#include <string>
#include <optional>
#include <chrono>
#include <vector>

namespace myproject {
namespace models {

struct BaseModel {
    std::string id;
    std::chrono::system_clock::time_point created_at;
    std::chrono::system_clock::time_point updated_at;
    std::optional<std::chrono::system_clock::time_point> deleted_at;
};

/// Пользователи системы
/// Table: public.users
struct User : public BaseModel {
    std::string email;
    std::string name;
    std::optional<int32_t> age;
    std::string status = "active";
    std::optional<std::string> department_id;
    std::vector<Order> orders;
};

} // namespace models
} // namespace myproject
```

---

## 📦 SQL DDL генерация

### PostgreSQL

```sql
-- Code generated by buffalo-db. DO NOT EDIT.
-- Dialect: PostgreSQL
-- Source: protos/user.proto

CREATE SCHEMA IF NOT EXISTS public;

CREATE TABLE public.users (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    email       VARCHAR(255) NOT NULL UNIQUE,
    name        VARCHAR(100) NOT NULL,
    age         INTEGER,
    status      VARCHAR(20)  NOT NULL DEFAULT 'active',
    department_id UUID      REFERENCES departments(id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ,
    
    CONSTRAINT chk_age CHECK (age >= 0 AND age <= 200)
);

CREATE UNIQUE INDEX idx_users_email ON public.users (email);
CREATE INDEX idx_users_status ON public.users (status, created_at);

COMMENT ON TABLE public.users IS 'Пользователи системы';

-- ─────────────────────────────────────────

CREATE TABLE public.orders (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    total      DECIMAL(10,2) NOT NULL,
    status     VARCHAR(255) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_orders_user ON public.orders (user_id);
CREATE INDEX idx_orders_date ON public.orders (created_at);
```

---

## 🔄 Миграции

### Формат миграций

```
migrations/
├── 20260213_001_create_users.up.sql
├── 20260213_001_create_users.down.sql
├── 20260213_002_create_orders.up.sql
├── 20260213_002_create_orders.down.sql
└── buffalo_migrations.json          # метаданные миграций
```

### buffalo_migrations.json

```json
{
  "version": "1.21.0",
  "dialect": "postgresql",
  "migrations": [
    {
      "id": "20260213_001",
      "name": "create_users",
      "checksum": "sha256:abc123...",
      "created_at": "2026-02-13T10:00:00Z",
      "source_proto": "protos/user.proto"
    }
  ]
}
```

---

## 🔌 CLI команды

```bash
# ═══════════════════════════════════════════
# Инициализация и информация
# ═══════════════════════════════════════════
buffalo db init                           # извлечь db.proto, создать секцию в конфиге
buffalo db list-protos                    # показать встроенные proto
buffalo db status                         # статус: какие модели обнаружены

# ═══════════════════════════════════════════
# Генерация моделей
# ═══════════════════════════════════════════
buffalo db generate                       # генерация моделей для всех языков
buffalo db generate --lang python         # только для Python
buffalo db generate --lang go --orm gorm  # переопределить ORM из CLI
buffalo db generate --dry-run             # показать что будет сгенерировано

# ═══════════════════════════════════════════  
# SQL DDL
# ═══════════════════════════════════════════
buffalo db ddl                            # генерация DDL
buffalo db ddl --dialect mysql            # переопределить диалект
buffalo db ddl --output schema.sql        # вывод в файл

# ═══════════════════════════════════════════
# Миграции
# ═══════════════════════════════════════════
buffalo db migrate generate               # создать миграцию из текущих proto
buffalo db migrate diff                   # показать diff от последней миграции
buffalo db migrate validate               # проверить целостность миграций

# ═══════════════════════════════════════════
# Анализ
# ═══════════════════════════════════════════
buffalo db inspect                        # список таблиц, колонок, связей
buffalo db inspect --format json          # машиночитаемый вывод
buffalo db diagram                        # ER-диаграмма (mermaid)
buffalo db diagram --format dot           # Graphviz
buffalo db check-deps                     # проверить что ORM-зависимости установлены
```

---

## 📂 Структура файлов проекта

```
internal/
  db/
    types.go               # TableDef, ColumnDef, IndexDef, RelationDef, ORMPlugin
    parser.go              # Regex-парсинг аннотаций [(buffalo.db.*)]
    parser_test.go
    codegen.go             # CodeGenerator interface + фабрика
    codegen_test.go
    ddl.go                 # SQL DDL генератор (PostgreSQL, MySQL, SQLite)
    ddl_test.go
    migrate.go             # Генератор миграций, diff
    migrate_test.go
    plugin.go              # DBPlugin (plugin.Plugin) — интеграция в pipeline
    plugin_test.go
    inspect.go             # Анализ и инспекция моделей
    diagram.go             # ER-диаграммы (mermaid, dot)
    orm/                   # ORM-специфичные генераторы
      registry.go          # Реестр ORM-плагинов
      checker.go           # Проверка установленных зависимостей
      python_none.go       # Python dataclass
      python_pydantic.go   # Pydantic v1/v2
      python_sqlalchemy.go # SQLAlchemy
      go_none.go           # Go plain struct
      go_gorm.go           # GORM
      go_sqlx.go           # sqlx
      rust_none.go         # Rust plain struct
      rust_diesel.go       # Diesel
      rust_seaorm.go       # SeaORM
      cpp_none.go          # C++ struct

internal/
  embedded/
    proto/
      buffalo/
        db/
          db.proto         # ORM аннотации proto

internal/
  cli/
    db.go                  # CLI-команды buffalo db *
```

---

## 🔍 Проверка зависимостей ORM

При сборке buffalo проверяет, что указанный ORM-плагин доступен в окружении:

### Python
```bash
# pydantic
python -c "import pydantic; print(pydantic.VERSION)"

# pydantic@2.0
python -c "import pydantic; assert pydantic.VERSION.startswith('2.0')"

# sqlalchemy
python -c "import sqlalchemy; print(sqlalchemy.__version__)"
```

### Go
```bash
# gorm
go list -m gorm.io/gorm

# ent
go list -m entgo.io/ent
```

### Rust
```bash
# diesel
cargo metadata --format-version 1 | grep "diesel"

# sea-orm
cargo metadata --format-version 1 | grep "sea-orm"
```

Если зависимость не найдена — **warning** (не error), генерация продолжается с `None` fallback и сообщением:

```
⚠️  ORM plugin 'pydantic@2.0' not found in environment.
    Falling back to 'None' (pure Python dataclasses).
    Install: pip install 'pydantic>=2.0,<3.0'
```

---

## 📋 План реализации по фазам

### Фаза 0: Подготовка (1 задача)
| # | Задача | Файлы |
|---|--------|-------|
| 0.1 | Обновить `ROADMAP.md` — перенести buffalo.db из Backlog | `docs/ROADMAP.md` |

### Фаза 1: Ядро — типы, парсер, proto (5 задач)
| # | Задача | Файлы |
|---|--------|-------|
| 1.1 | Создать `db.proto` с аннотациями table/column/relation | `internal/embedded/proto/buffalo/db/db.proto` |
| 1.2 | Создать `types.go` — все структуры данных | `internal/db/types.go` |
| 1.3 | Обновить конфиг — добавить ORM поля в LanguagesConfig + секцию `db:` | `internal/config/config.go` |
| 1.4 | Создать `parser.go` — regex-парсинг аннотаций | `internal/db/parser.go` |
| 1.5 | Тесты парсера | `internal/db/parser_test.go` |

### Фаза 2: Кодогенерация — базовые модели (6 задач)
| # | Задача | Файлы |
|---|--------|-------|
| 2.1 | Создать `codegen.go` — интерфейс `DBCodeGenerator` + фабрика | `internal/db/codegen.go` |
| 2.2 | ORM registry + dependency checker | `internal/db/orm/registry.go`, `checker.go` |
| 2.3 | Python генераторы: `None`, `pydantic`, `sqlalchemy` | `internal/db/orm/python_*.go` |
| 2.4 | Go генераторы: `None`, `gorm`, `sqlx` | `internal/db/orm/go_*.go` |
| 2.5 | Rust генераторы: `None`, `diesel` | `internal/db/orm/rust_*.go` |
| 2.6 | C++ генератор: `None` | `internal/db/orm/cpp_none.go` |

### Фаза 3: SQL DDL и миграции (4 задачи)
| # | Задача | Файлы |
|---|--------|-------|
| 3.1 | Создать `ddl.go` — DDL генератор (PostgreSQL, MySQL, SQLite) | `internal/db/ddl.go` |
| 3.2 | Создать `migrate.go` — генератор миграций | `internal/db/migrate.go` |
| 3.3 | Создать `inspect.go` + `diagram.go` — анализ и ER-диаграммы | `internal/db/inspect.go`, `diagram.go` |
| 3.4 | Тесты DDL и миграций | `internal/db/ddl_test.go`, `migrate_test.go` |

### Фаза 4: Интеграция — plugin + CLI (4 задачи)
| # | Задача | Файлы |
|---|--------|-------|
| 4.1 | Создать `plugin.go` — DBPlugin (plugin.Plugin) | `internal/db/plugin.go` |
| 4.2 | Зарегистрировать DBPlugin в `internal/plugin/builtin.go` | `internal/plugin/builtin.go` |
| 4.3 | Создать `db.go` — CLI-команды | `internal/cli/db.go` |
| 4.4 | Обновить `internal/embedded/extract.go` для db.proto | `internal/embedded/extract.go` |

### Фаза 5: Тестирование и документация (4 задачи)
| # | Задача | Файлы |
|---|--------|-------|
| 5.1 | Integration tests | `internal/db/integration_test.go` |
| 5.2 | Codegen tests для каждого языка/ORM | `internal/db/codegen_test.go` |
| 5.3 | Тестовый proto-проект с db-аннотациями | `test-project/` |
| 5.4 | Документация: DB_GUIDE.md | `docs/DB_GUIDE.md` |

### Фаза 6: Конфиги-примеры и финализация (2 задачи)
| # | Задача | Файлы |
|---|--------|-------|
| 6.1 | Пример конфига `buffalo-with-db.yaml` | `configs/buffalo-with-db.yaml` |
| 6.2 | Обновить README.md, CHANGELOG | `README.md` |

---

## 📊 Сводка

| Метрика | Значение |
|---------|----------|
| **Новых пакетов** | 1 (`internal/db/`) + подпакет `orm/` |
| **Новых файлов** | ~25-30 |
| **Proto файлов** | 1 (`db.proto`) |
| **Поддержка языков** | 4 (Python, Go, Rust, C++) |
| **ORM плагинов** | 9+ (None×4, pydantic, sqlalchemy, gorm, sqlx, diesel) |
| **CLI команд** | ~12 подкоманд |
| **SQL диалектов** | 3 (PostgreSQL, MySQL, SQLite) |

---

## 🔗 Зависимости от существующего кода

| Компонент | Используется |
|-----------|-------------|
| `internal/plugin` | `Plugin`, `PluginType`, `Config`, `Input`, `Output`, hook points |
| `internal/config` | `LanguagesConfig`, `PythonConfig`, `GoConfig`, etc. |
| `internal/embedded` | `embed.FS`, `ExtractAllProtos()` |
| `internal/validation` | Паттерн реализации (parser + codegen + plugin) |
| `internal/permissions` | Паттерн generator + CLI |
| `pkg/errors` | `errors.Wrap`, `errors.New` |
| `pkg/logger` | `logger.Logger` |

---

*Этот документ является техническим планом для реализации buffalo.db v1.21.0.*
