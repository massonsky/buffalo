# 🦬 Buffalo — Multi-Language Protobuf/gRPC Build System

<div align="center">

[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.24+-00ADD8.svg)](https://go.dev/)
[![Version](https://img.shields.io/badge/version-1.21.0-green.svg)](https://github.com/massonsky/buffalo/releases)
[![Stars](https://img.shields.io/github/stars/massonsky/buffalo?style=social)](https://github.com/massonsky/buffalo)
[![Issues](https://img.shields.io/github/issues/massonsky/buffalo)](https://github.com/massonsky/buffalo/issues)

**Быстрые инкрементальные сборки protobuf/gRPC для Python, Go, Rust и C++**

</div>

---

## ✨ Почему Buffalo

- ⚡ **Скорость**: кэш, инкрементальные и параллельные сборки
- 🌍 **Мультиязычность**: Python / Go / Rust / C++
- 🧩 **Расширяемость**: плагины, шаблоны, workspace
- 🔐 **Security-first**: permissions audit + matrix
- 📈 **Наблюдаемость**: stats, metrics, dependency graph

---

## 🎬 30-секундный старт

```bash
buffalo init myproject
cd myproject
buffalo build
```

Хочешь быстрее в прод? Начни с [Quick Start](docs/readme/QUICK_START.md).

---

## 🗺️ Визуальная схема пайплайна

```mermaid
graph LR
    A[.proto files] --> B[buffalo parse]
    B --> C{Build Targets}
    C --> D[Python]
    C --> E[Go]
    C --> F[Rust]
    C --> G[C++]
    B --> H[validate]
    B --> I[permissions]
    B --> J[models]
    J --> K[ORM-aware model generation]
```

## 🧠 Карта возможностей

```mermaid
mindmap
  root((Buffalo))
    Build
      Incremental cache
      Parallel workers
      Watch mode
    Modeling
      buffalo.models
      ORMs
      Multi-language codegen
    Security
      permissions matrix
      audit
      policy generation
    Analysis
      graph
      metrics
      doctor
    Monorepo
      workspace
      affected builds
```

---

## 📚 Читай по разделам (коротко и удобно)

- **Установка** → [docs/readme/INSTALLATION.md](docs/readme/INSTALLATION.md)
- **Быстрый старт** → [docs/readme/QUICK_START.md](docs/readme/QUICK_START.md)
- **CLI шпаргалка** → [docs/readme/CLI_CHEATSHEET.md](docs/readme/CLI_CHEATSHEET.md)
- **Инструменты (`buffalo tools`)** → [docs/readme/TOOLS.md](docs/readme/TOOLS.md)
- **Graph + Workspace** → [docs/readme/GRAPH_AND_WORKSPACE.md](docs/readme/GRAPH_AND_WORKSPACE.md)
- **Permissions/RBAC** → [docs/readme/PERMISSIONS.md](docs/readme/PERMISSIONS.md)
- **Models (`buffalo.models`)** → [docs/readme/MODELS.md](docs/readme/MODELS.md)

---

## 🧪 Примеры

- Proto model examples: [examples/models](examples/models)
- Generated examples: [examples/gen](examples/gen)

---

## 📖 Полная документация

- [QUICKSTART.md](QUICKSTART.md)
- [INSTALL.md](INSTALL.md)
- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)
- [docs/CONFIG_GUIDE.md](docs/CONFIG_GUIDE.md)
- [docs/PLUGIN_GUIDE.md](docs/PLUGIN_GUIDE.md)
- [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md)
- [docs/METRICS_GUIDE.md](docs/METRICS_GUIDE.md)
- [docs/ROADMAP.md](docs/ROADMAP.md)

---

## 🤝 Контрибьютинг

- Bugs/Ideas: [GitHub Issues](https://github.com/massonsky/buffalo/issues)
- Discussions: [GitHub Discussions](https://github.com/massonsky/buffalo/discussions)
- Contribution guide: [CONTRIBUTING.md](CONTRIBUTING.md)


