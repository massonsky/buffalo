# 🛠️ Tools

`buffalo tools` helps install and validate required toolchains for target languages.

## Check installed tools

```bash
buffalo tools check
buffalo tools check go python
```

## Install tools

```bash
buffalo tools install
buffalo tools install go python
buffalo tools install --all
buffalo tools install --dry-run
```

## List tools

```bash
buffalo tools list
buffalo tools list --all
```

## Tip

Run `buffalo tools check` in CI before build to fail fast on missing dependencies.
