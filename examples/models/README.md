# Examples: `buffalo.models`

Этот каталог содержит **максимально полные** примеры аннотаций `buffalo.models` для Proto-моделей.

## Файлы

- `00_basic.proto` — минимальный рабочий пример.
- `10_model_options_full.proto` — полный набор `ModelOptions`.
- `20_field_options_full.proto` — полный набор `FieldModelOptions`.
- `30_relations_full.proto` — полный набор relation-сценариев (`BELONGS_TO`, `HAS_ONE`, `HAS_MANY`, `MANY_TO_MANY`).

---

## Быстрая проверка

Из корня репозитория:

1. Убедиться, что embedded proto доступны (если нужно):
   - `buffalo models init`

2. Посмотреть найденные модели:
   - `buffalo models list --proto ./examples/models`

3. Инспект конкретной модели:
   - `buffalo models inspect FieldOptionsShowcase --proto ./examples/models`

4. Генерация (пример):
   - `buffalo models generate --proto ./examples/models --lang python --orm pydantic@2.0 --output ./examples/generated/python`

---

## Карта покрытия опций

## `ModelOptions`

Покрыты в `10_model_options_full.proto`:

- `name`
- `table_name`
- `schema`
- `description`
- `tags`
- `abstract`
- `extends`
- `mixins`
- `indexes` (+ `type`, `where`, `comment`)
- `uniques`
- `checks`
- `soft_delete`
- `timestamps`
- `deprecated`
- `deprecated_message`
- `generate`

## `FieldModelOptions`

Покрыты в `20_field_options_full.proto`:

- Identity: `alias`, `description`
- Constraints: `primary_key`, `auto_increment`, `nullable`, `unique`, `default_value`, `max_length`, `min_length`, `precision`, `scale`
- Typing: `custom_type`, `db_type`
- Visibility/Behavior: `visibility`, `behavior`, `sensitive`, `deprecated`, `deprecated_message`
- Indexing: `index`, `index_type`
- Serialization: `json_name`, `xml_name`, `omit_empty`
- Relation: `relation` (детально)
- Docs: `example`, `comment`
- Auto: `auto_generate`, `auto_now`, `auto_now_add`
- Extras: `sequence`, `collation`, `ignore`, `db_ignore`, `api_ignore`, `tags`, `metadata`

## `RelationDef`

Покрыты в `30_relations_full.proto`:

- `type`
- `model`
- `foreign_key`
- `references`
- `join_table`
- `on_delete`
- `on_update`
- `eager`
- `through`
- `inverse_of`

---

## Примечание

Некоторые опции уже доступны в аннотациях, но могут быть частично поддержаны отдельными language/ORM генераторами. Эти examples показывают **канонический способ описания модели в proto**, а не гарантируют 1:1 эмит кода для каждого бэкенда в текущей версии.
