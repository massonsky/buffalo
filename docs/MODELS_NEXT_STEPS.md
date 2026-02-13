# 🧩 Buffalo Models — Next Steps Roadmap

> Документ описывает дальнейшие шаги развития `buffalo.models`: генерация моделей, методов, классов, доп. опций и языковых фич (декораторы, derive, атрибуты, теги и т.д.).

---

## 1) Цели развития

1. Сделать генерацию моделей **предсказуемой и расширяемой** для всех языков.
2. Добавить генерацию не только полей, но и **методов/классов инфраструктуры** (repo, factory, query helpers).
3. Поддержать **feature parity** между языками без потери их нативных сильных сторон.
4. Перевести часть “жёстко зашитых” решений в **конфигурируемые опции**.

---

## 2) Приоритеты (что делать первым)

## P0 — Критично

- Единый feature-matrix для всех генераторов (что поддерживается/игнорируется).
- Генерация relations (включая `belongs_to`, `many_to_many`, `on_delete`, `on_update`) в ORM-режимах.
- Генерация composite indexes / unique / check constraints там, где ORM это поддерживает.
- Поддержка `Generate = ["model", "repo", "factory"]` на уровне кода, а не только в типах.
- Стабильные golden-тесты для всех языков и ORM комбинаций.

## P1 — Важно

- Генерация validation-декораторов/атрибутов/тегов из model metadata.
- Улучшение nullable/default behavior и platform-specific defaults.
- Расширяемая система language hooks (pre/post field emit, pre/post model emit).

## P2 — Nice to have

- Темплейт-оверрайды для командного кастомного стиля.
- Генерация документации по моделям (schema docs / markdown).
- Генерация миграций (черновой режим) для ORM-плагинов.

---

## 3) Общая архитектура улучшений

### 3.1. Единый intermediate model graph

Вместо прямой генерации из `ModelDef` добавить промежуточный слой:

- `ResolvedModel` / `ResolvedField`
- нормализованные relation edges
- итоговые default/nullable semantics
- capability flags per language/ORM

Плюсы:

- меньше дублирования логики в `codegen_*`
- проще добавлять новый язык/ORM
- централизованная валидация несовместимых опций

### 3.2. Capability-based генерация

Для каждого генератора объявлять capabilities:

- `supports_relations`
- `supports_composite_index`
- `supports_field_validators`
- `supports_soft_delete`
- `supports_schema`

Если аннотация не поддерживается — явный warning с actionable text.

### 3.3. Пайплайн генерации

1. Parse proto annotations
2. Resolve model graph
3. Validate capabilities
4. Emit artifacts (`model`, `repo`, `factory`, `query`, `init`)
5. Post-process imports / format

---

## 4) Генерация методов и классов

### 4.1. `model` (текущий слой)

Расширить:

- computed/virtual hooks
- field-level docs/metadata passthrough
- consistent deprecation handling

### 4.2. `repo`

Генерировать базовый repository слой:

- `Create`, `GetByID`, `List`, `Update`, `Delete`
- фильтрация persistable fields
- paging/sorting placeholders
- transaction-aware signatures

### 4.3. `factory`

Генерировать factory/helper класс/модуль:

- `New<Model>()` с sane defaults
- `With<Field>()` fluent API (где уместно)
- test fixture helpers

### 4.4. `query` (опционально)

- typed query predicates
- field-safe order/filter constants
- relation preload/include helpers

---

## 5) Языковые фишки и конкретные улучшения

## Python

### Pydantic

- Генерировать:
  - `@field_validator` / `@model_validator`
  - `@computed_field` для computed behavior
  - `model_config` из visibility/sensitive metadata
- Поддержать pydantic v1/v2 различия через адаптерный слой.

### SQLAlchemy

- `belongs_to`, `has_one`, `has_many`, `many_to_many` (`secondary=...`)
- `ForeignKey(..., ondelete=..., onupdate=...)`
- Composite indexes / constraints через `__table_args__`
- Опционально: mixin classes (timestamps/soft delete/audit)

### Dataclass режим

- `@property`/`@cached_property` для computed fields
- сериализационные helper methods (`to_dict`, `from_dict`)

## Go

### GORM

- relation tags: `foreignKey`, `references`, `constraint:OnUpdate...,OnDelete...`
- composite indexes и named indexes
- soft delete и timestamps как mixin embedding
- optional `validate` tags (go-playground/validator)

### sqlx

- расширить beyond plain struct:
  - query constants
  - scan helpers
  - nullable wrappers strategy

### Общие улучшения Go

- генерация методов:
  - `TableName()`
  - `IsZero()`
  - `Validate()` (опционально)

## Rust

### Diesel

- `Associations`, `#[diesel(belongs_to(...))]`
- отдельные `Insertable`, `AsChangeset`, `Selectable`
- schema-qualified table support
- derive `Default` при безопасных defaults

### Serde / validator

- `#[serde(...)]` опции по metadata
- `validator` crate derives/attributes (опционально)

## C++

- Сгенерировать полноценные `to_json` / `from_json` (nlohmann/json)
- Разделить `.h` и `.cpp` где есть логика методов
- Добавить builder pattern для сложных моделей
- Опциональные атрибуты/макросы для валидации

---

## 6) Дополнительные опции конфигурации

Добавить в `buffalo.yaml` (или раздел models):

- `models.strict_capabilities: bool`
- `models.generate: [model, repo, factory, query]`
- `models.decorators:` (Python-specific)
- `models.derive:` (Rust-specific)
- `models.tags.go` / `models.tags.python` / `models.tags.rust`
- `models.naming:` стратегия (`snake_case`, `camelCase`, `PascalCase`)
- `models.nullable_strategy:` (`pointer`, `optional`, `zero-value`)

---

## 7) Тестовая стратегия

1. Golden tests per language/ORM (snapshot generated files)
2. Matrix tests:
   - language × ORM × feature flags
3. Regression tests на найденные баги:
   - bool defaults в Python
   - braces в Rust
   - visibility sections в C++
   - config load / `--all`
4. E2E test project pipeline:
   - list/inspect/generate/check-deps/build/validate/permissions

---

## 8) План релизов (предложение)

### v1.22

- Capabilities layer
- Relations (MVP) в Python SQLAlchemy + Go GORM
- Composite indexes (MVP)
- Golden tests baseline

### v1.23

- `repo` generation (Go/Python first)
- Pydantic validators / computed fields
- Rust Diesel associations + changeset

### v1.24

- `factory` + `query` artifacts
- C++ json serialization
- Advanced config knobs

---

## 9) Быстрые практические задачи (next sprint)

1. Ввести `GeneratorCapabilities` интерфейс.
2. Реализовать relation emitters для:
   - Python SQLAlchemy
   - Go GORM
3. Добавить обработку `IndexDef/Uniques/Checks` в ORM-генераторы.
4. Запустить единые golden-тесты для `internal/models`.
5. Включить `Generate` artifacts (`repo`, `factory`) хотя бы для Go/Python MVP.

---

## 10) Definition of Done для этапа “Models 2.0”

- Не менее 80% покрытие feature-matrix во всех 4 языках.
- Все аннотации либо генерируются, либо дают явный warning с причиной.
- Генерация `model + repo + factory` минимум в 2 языках.
- Полный E2E в test-project проходит без ручных правок generated кода.
- Документация в `docs/` синхронизирована с реальными возможностями.
