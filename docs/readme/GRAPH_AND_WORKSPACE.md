# 🕸️ Graph & Workspace

## Dependency graph

```bash
buffalo graph
buffalo graph --format mermaid --output deps.md
buffalo graph --format dot --output deps.dot
buffalo graph analyze --cycles
buffalo graph stats
```

## Workspace for monorepos

```bash
buffalo workspace init --discover
buffalo workspace list
buffalo workspace graph --format mermaid
buffalo workspace affected --since HEAD~5
buffalo workspace build --parallel 4
```

## Why this matters

- See dependency hotspots
- Build only affected projects
- Speed up large mono-repos
