# Buffalo Templates Testing

## Созданные шаблоны

### Go Templates (custom-go)
- `message.go.tmpl` - генерация структур сообщений с методами Validate() и String()
- `service.go.tmpl` - генерация клиентских и серверных интерфейсов для gRPC сервисов
- `message_test.go.tmpl` - генерация unit-тестов для сообщений
- `README.md.tmpl` - генерация документации проекта

### Python Templates (custom-python)
- `message.py.tmpl` - генерация dataclass'ов с методами to_dict(), to_json(), validate()
- `service.py.tmpl` - генерация async servicer'ов и client stub'ов
- `test_message.py.tmpl` - генерация pytest тестов

### Rust Templates (custom-rust)
- `message.rs.tmpl` - генерация структур с derive макросами (Serialize, Deserialize)

## Статус тестирования

✅ **template list** - работает корректно
- Показывает все 3 зарегистрированных шаблона
- Отображает: имя, язык, путь, паттерны

✅ **template list --lang {language}** - работает корректно
- Фильтрация по go: 1 шаблон
- Фильтрация по python: 1 шаблон
- Фильтрация по rust: 1 шаблон

✅ **template validate --template {name}** - работает корректно
- Валидация custom-go: успешно
- Валидация custom-python: успешно
- Валидация custom-rust: успешно
- Проверяет: имя, язык, путь, паттерны, переменные

## Конфигурация

Шаблоны настроены в `buffalo.yaml`:

```yaml
templates:
  - name: custom-go
    language: go
    path: ./templates/go
    patterns: ["**/*.tmpl"]
    enabled: true
    vars:
      packagePrefix: github.com/yourorg

  - name: custom-python
    language: python
    path: ./templates/python
    patterns: ["**/*.tmpl"]
    enabled: true
    vars:
      modulePrefix: proto_gen

  - name: custom-rust
    language: rust
    path: ./templates/rust
    patterns: ["**/*.tmpl"]
    enabled: true
    vars:
      cratePrefix: proto_rust
```

## Структура шаблонов

```
templates/
├── go/
│   ├── message.go.tmpl       (структуры данных)
│   ├── service.go.tmpl       (gRPC интерфейсы)
│   ├── message_test.go.tmpl  (тесты)
│   └── README.md.tmpl        (документация)
├── python/
│   ├── message.py.tmpl       (dataclasses)
│   ├── service.py.tmpl       (async servicers)
│   └── test_message.py.tmpl  (pytest)
└── rust/
    └── message.rs.tmpl       (serde structs)
```

## Возможности шаблонов

### Message Templates
- Валидация полей (Required/Optional)
- Сериализация (JSON, String)
- Десериализация (from_dict, from_json)
- Type safety

### Service Templates
- Client/Server интерфейсы
- Async/Sync варианты
- Unimplemented stubs
- gRPC metadata

### Test Templates
- Unit тесты для каждого сообщения
- Тесты валидации
- Тесты сериализации
- Edge cases

## Следующие шаги

1. ✅ Создание базовых шаблонов
2. ✅ Тестирование команд list/validate
3. 🔄 Тестирование генерации (template generate)
4. 🔄 Интеграция с build процессом
5. 🔄 Тестирование hook points (pre/post)

---
Дата тестирования: 17 января 2026
