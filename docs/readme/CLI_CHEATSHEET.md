# 🧰 CLI Cheatsheet

## Build

```bash
buffalo build
buffalo build --lang python
buffalo build --jobs 8
buffalo rebuild
```

## Quality & diagnostics

```bash
buffalo check
buffalo doctor
buffalo validate -p ./protos
buffalo lint
```

## Dev loop

```bash
buffalo watch
buffalo diff
buffalo stats
buffalo metrics
```

## Cleanup

```bash
buffalo clear cache
buffalo clear generated
buffalo clear all
```

## Advanced modules

```bash
buffalo tools check
buffalo graph --format mermaid --output deps.md
buffalo permissions summary -p ./protos
buffalo workspace list
```
