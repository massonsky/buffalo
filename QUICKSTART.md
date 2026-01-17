# 🚀 Quick Start - Buffalo Build System

**TL;DR** для быстрого старта

## ⚡ Самый быстрый способ (1 минута)

### Linux/macOS
```bash
git clone https://github.com/massonsky/buffalo.git && cd buffalo
curl -sSL https://raw.githubusercontent.com/massonsky/buffalo/dev/install.sh | bash
buffalo --version  # Done! ✅
```

### Windows (PowerShell)
```powershell
git clone https://github.com/massonsky/buffalo.git; cd buffalo
irm https://raw.githubusercontent.com/massonsky/buffalo/dev/install.ps1 | iex
buffalo --version  # Done! ✅
```

### Docker
```bash
git clone https://github.com/massonsky/buffalo.git && cd buffalo
docker build -t buffalo:latest .
docker run --rm -v $(pwd):/workspace buffalo:latest buffalo --version
```

---

## 🛠 Разработка и сборка

### Linux/macOS
```bash
./build.sh build          # сборка
./build.sh test           # тесты
./build.sh check          # проверки (fmt+vet+lint+test)
./build.sh install        # установка в систему
```

### Windows (PowerShell)
```powershell
.\build.ps1 build         # сборка
.\build.ps1 test          # тесты
.\build.ps1 check         # проверки
.\build.ps1 install       # установка
```

### Makefile (Linux/macOS)
```bash
make build                # сборка
make test                 # тесты
make check                # проверки
sudo make install-system  # установка
```

---

## 📝 Чёткий лист команд

| Задача | Linux/macOS | Windows | Docker |
|--------|-------------|---------|--------|
| Установка | `curl \| bash` | `irm \| iex` | `docker build` |
| Сборка | `./build.sh build` | `.\build.ps1 build` | `docker run ... build` |
| Тесты | `./build.sh test` | `.\build.ps1 test` | `docker run ... test` |
| Все ОС | `./build.sh build-all` | `.\build.ps1 build-all` | built-in |
| Справка | `./build.sh help` | `.\build.ps1 help` | `docker run ... help` |

---

## 📦 Методы установки

| Метод | Linux/macOS | Windows | Просто |
|-------|-------------|---------|--------|
| **Скрипт (рекомендуется)** | ✅ | ✅ | ⭐⭐⭐ |
| **Из исходников** | ✅ | ✅ | ⭐⭐ |
| **Docker** | ✅ | ✅ | ⭐⭐ |
| **Go** | ✅ | ✅ | ⭐ |

---

## 🔍 Требования

- **Go 1.21+**
- **protoc 3.20+**
- **Git**

Всё остальное опционально для разработки.

---

## 📚 Подробная документация

- **[BUILD_SYSTEM.md](BUILD_SYSTEM.md)** - Полное руководство всех методов и опций
- **[INSTALL.md](INSTALL.md)** - Подробная установка с решением проблем
- **[README.md](README.md)** - Основная информация о проекте

---

## 🆘 Проблемы?

```bash
# Очистить и пересобрать
./build.sh clean-all && ./build.sh build

# Проверить версию
./build.sh version

# Получить справку
./build.sh help
```

Для подробного решения проблем см. [BUILD_SYSTEM.md](BUILD_SYSTEM.md#-решение-проблем)

---

**Готово к работе!** 🎉
