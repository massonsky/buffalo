# 🧩 Models (`buffalo.models`)

Generate typed models for Python/Go/Rust/C++ with optional ORM-specific output.

## Explore models

```bash
buffalo models list --proto ./protos
buffalo models inspect User --proto ./protos
```

## Generate models

```bash
buffalo models generate --proto ./protos --lang python --orm pydantic@2.0 --output ./generated/models/python
buffalo models generate --proto ./protos --lang go --orm gorm --output ./generated/models/go
buffalo models generate --proto ./protos --lang rust --orm diesel@2.1 --output ./generated/models/rust
buffalo models generate --proto ./protos --lang cpp --orm None --output ./generated/models/cpp
```

## Check dependencies

```bash
buffalo models check-deps --lang python --orm pydantic@2.0
buffalo models check-deps --lang go --orm gorm
```

## More

- Deep roadmap: [docs/MODELS_NEXT_STEPS.md](../MODELS_NEXT_STEPS.md)
- Proto examples: [examples/models](../../examples/models)
