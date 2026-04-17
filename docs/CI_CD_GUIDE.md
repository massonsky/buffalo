# Buffalo CI/CD Integration Guide

This guide explains how to integrate Buffalo into your CI/CD pipelines for automated protobuf compilation.

## Table of Contents

- [GitHub Actions](#github-actions)
- [GitLab CI](#gitlab-ci)
- [Jenkins](#jenkins)
- [Pre-commit Hooks](#pre-commit-hooks)
- [Docker Integration](#docker-integration)
- [Best Practices](#best-practices)

---

## GitHub Actions

Buffalo uses a two-stage CI/CD pipeline with automatic promotion from `dev` to `main`.

### Pipeline Flow

```
push to dev/devb
      │
      ├── Lint (golangci-lint)
      ├── Test (ubuntu, windows, macos)
      │
      └── Build (5 platform binaries)
              │
              └── Promote (merge to main → tag v1.30.<hash>)
                            │
                            └── Release (GitHub Release with binaries)
```

### Workflow Files

| File | Trigger | Purpose |
|------|---------|---------|
| `ci.yml` | push to `dev`/`devb`, PR | Lint → Test → Build → Promote to main + tag |
| `buffalo-release.yml` | tag `v*` | Build release binaries, create GitHub Release |
| `buffalo-build.yml` | PR only | Quick build verification |

### Version Format

Version is automatically set to `1.30.<short-git-hash>` and injected via ldflags:

```
-X github.com/massonsky/buffalo/internal/version.Version=1.30.abc1234
-X github.com/massonsky/buffalo/internal/version.BuildDate=2026-04-17
-X github.com/massonsky/buffalo/internal/version.GitCommit=abc1234
```

### Using Buffalo in Your Project

Install from latest release:

```yaml
name: Build with Buffalo

on:
  push:
    branches: [main, develop]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install Buffalo
        run: |
          LATEST=$(curl -s https://api.github.com/repos/massonsky/buffalo/releases/latest | grep tag_name | cut -d'"' -f4)
          curl -L -o buffalo https://github.com/massonsky/buffalo/releases/download/${LATEST}/buffalo-linux-amd64
          chmod +x buffalo
          sudo mv buffalo /usr/local/bin/

      - name: Build Proto Files
        run: buffalo build
```

Location: `.github/workflows/`

### Advanced Configuration

#### Multi-language Setup

```yaml
- name: Setup Go
  uses: actions/setup-go@v5
  with:
    go-version: '1.21'

- name: Setup Python
  uses: actions/setup-python@v5
  with:
    python-version: '3.11'

- name: Install Language Plugins
  run: |
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
    pip install grpcio-tools
```

#### Caching

```yaml
- name: Cache Buffalo Dependencies
  uses: actions/cache@v4
  with:
    path: |
      .buffalo/
      generated/
    key: ${{ runner.os }}-buffalo-${{ hashFiles('buffalo.yaml', 'protos/**/*.proto') }}
```

#### Artifacts

```yaml
- name: Upload Generated Files
  uses: actions/upload-artifact@v4
  with:
    name: generated-proto
    path: generated/
    retention-days: 7
```

---

## GitLab CI

### Quick Start

1. **Create `.gitlab-ci.yml`** in project root:

```yaml
image: ubuntu:22.04

stages:
  - build
  - test

buffalo:build:
  stage: build
  before_script:
    - apt-get update -qq
    - apt-get install -y curl protobuf-compiler
    - curl -L -o /usr/local/bin/buffalo https://github.com/massonsky/buffalo/releases/download/v0.7.0/buffalo-linux-amd64
    - chmod +x /usr/local/bin/buffalo
  script:
    - buffalo build
  artifacts:
    paths:
      - generated/
    expire_in: 7 days
```

2. **Push to GitLab** - pipeline runs automatically

### Full Template

See `examples/ci/gitlab-ci.yml` for a complete template with:
- Multi-stage pipeline (setup, validate, build, test, deploy)
- Language-specific test jobs
- Dry-run support
- Artifact management

### GitLab Runner Configuration

For self-hosted runners:

```yaml
variables:
  BUFFALO_CACHE_DIR: /cache/buffalo

cache:
  key: ${CI_PROJECT_ID}
  paths:
    - .buffalo/
    - ${BUFFALO_CACHE_DIR}
```

---

## Jenkins

### Declarative Pipeline

```groovy
pipeline {
    agent any
    
    environment {
        BUFFALO_VERSION = '0.7.0'
        PATH = "${env.PATH}:/usr/local/bin"
    }
    
    stages {
        stage('Setup') {
            steps {
                sh '''
                    curl -L -o /usr/local/bin/buffalo \
                        https://github.com/massonsky/buffalo/releases/download/v${BUFFALO_VERSION}/buffalo-linux-amd64
                    chmod +x /usr/local/bin/buffalo
                    buffalo version
                '''
            }
        }
        
        stage('Doctor Check') {
            steps {
                sh 'buffalo doctor'
            }
        }
        
        stage('Build') {
            steps {
                sh 'buffalo build'
            }
        }
        
        stage('Archive') {
            steps {
                archiveArtifacts artifacts: 'generated/**/*', fingerprint: true
            }
        }
    }
    
    post {
        always {
            cleanWs()
        }
    }
}
```

### Scripted Pipeline

```groovy
node {
    stage('Checkout') {
        checkout scm
    }
    
    stage('Install Buffalo') {
        sh '''
            curl -L -o buffalo https://github.com/massonsky/buffalo/releases/download/v0.7.0/buffalo-linux-amd64
            chmod +x buffalo
            sudo mv buffalo /usr/local/bin/
        '''
    }
    
    stage('Build Proto') {
        sh 'buffalo build'
    }
    
    stage('Publish') {
        archiveArtifacts 'generated/**/*'
    }
}
```

---

## Pre-commit Hooks

Buffalo supports [pre-commit](https://pre-commit.com/) framework for Git hooks.

### Installation

1. **Install pre-commit**:
   ```bash
   pip install pre-commit
   ```

2. **Create `.pre-commit-config.yaml`** in project root:
   ```yaml
   repos:
     - repo: https://github.com/massonsky/buffalo
       rev: v0.7.0
       hooks:
         - id: buffalo-lint
         - id: buffalo-format
         - id: buffalo-validate
   ```

3. **Install hooks**:
   ```bash
   pre-commit install
   ```

### Available Hooks

- **`buffalo-lint`** - Lint proto files
- **`buffalo-format`** - Format proto files
- **`buffalo-validate`** - Validate proto syntax
- **`buffalo-check`** - Check buffalo.yaml config
- **`buffalo-dry-run`** - Run dry-run build (optional, can be slow)

### Configuration Example

See `examples/pre-commit-config.yaml` for a complete example with:
- Buffalo hooks
- Standard pre-commit hooks (trailing whitespace, etc.)
- YAML linting

---

## Docker Integration

### Using Buffalo in Docker

#### Dockerfile Example

```dockerfile
FROM golang:1.21-alpine AS builder

# Install dependencies
RUN apk add --no-cache curl protobuf protobuf-dev

# Install Buffalo
ARG BUFFALO_VERSION=0.7.0
RUN curl -L -o /usr/local/bin/buffalo \
    https://github.com/massonsky/buffalo/releases/download/v${BUFFALO_VERSION}/buffalo-linux-amd64 && \
    chmod +x /usr/local/bin/buffalo

# Install language plugins
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

WORKDIR /workspace

# Copy proto files
COPY protos/ ./protos/
COPY buffalo.yaml ./

# Build proto files
RUN buffalo build

# Output artifacts
FROM scratch
COPY --from=builder /workspace/generated/ /generated/
```

#### Docker Compose

```yaml
version: '3.8'

services:
  buffalo-build:
    image: ubuntu:22.04
    volumes:
      - ./protos:/workspace/protos
      - ./buffalo.yaml:/workspace/buffalo.yaml
      - ./generated:/workspace/generated
    working_dir: /workspace
    command: |
      bash -c "
        apt-get update &&
        apt-get install -y curl protobuf-compiler &&
        curl -L -o /usr/local/bin/buffalo https://github.com/massonsky/buffalo/releases/download/v0.7.0/buffalo-linux-amd64 &&
        chmod +x /usr/local/bin/buffalo &&
        buffalo build
      "
```

### Multi-stage Build

```dockerfile
# Stage 1: Build proto files
FROM massonsky/buffalo:0.7.0 AS proto-builder
COPY protos/ /protos/
COPY buffalo.yaml /
RUN buffalo build

# Stage 2: Build Go application
FROM golang:1.21 AS go-builder
COPY --from=proto-builder /generated/go /app/proto
COPY . /app
WORKDIR /app
RUN go build -o server .

# Stage 3: Runtime
FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=go-builder /app/server /server
CMD ["/server"]
```

---

## Best Practices

### 1. Version Pinning

Always pin Buffalo version in CI:

```yaml
env:
  BUFFALO_VERSION: "0.7.0"  # Pinned version
```

### 2. Caching

Enable caching for faster builds:

**GitHub Actions:**
```yaml
- uses: actions/cache@v4
  with:
    path: .buffalo/
    key: ${{ hashFiles('buffalo.yaml') }}
```

**GitLab CI:**
```yaml
cache:
  paths:
    - .buffalo/
```

### 3. Dry-Run Before Merge

Run dry-run on PRs to catch errors early:

```yaml
- name: Dry-Run Build
  if: github.event_name == 'pull_request'
  run: buffalo build --dry-run
```

### 4. Environment Validation

Use `buffalo doctor` to validate environment:

```yaml
- name: Check Environment
  run: buffalo doctor
```

### 5. Parallel Builds

For large projects, use Buffalo's parallel workers:

```yaml
build:
  workers: 4  # In buffalo.yaml
```

Or via CLI:
```bash
buffalo build --workers 4
```

### 6. Artifact Management

Store generated files as artifacts for debugging:

```yaml
- name: Upload Artifacts
  if: failure()
  uses: actions/upload-artifact@v4
  with:
    name: build-logs
    path: |
      generated/
      .buffalo/logs/
```

### 7. Notification

Notify on build failures:

**GitHub Actions:**
```yaml
- name: Notify Slack
  if: failure()
  uses: 8398a7/action-slack@v3
  with:
    status: ${{ job.status }}
    webhook_url: ${{ secrets.SLACK_WEBHOOK }}
```

**GitLab CI:**
```yaml
after_script:
  - |
    if [ "$CI_JOB_STATUS" == "failed" ]; then
      curl -X POST -H 'Content-type: application/json' \
        --data '{"text":"Build failed!"}' \
        $SLACK_WEBHOOK
    fi
```

### 8. Security

- Use secrets for sensitive data:
  ```yaml
  env:
    PROTOC_LICENSE: ${{ secrets.PROTOC_LICENSE }}
  ```
- Scan generated code:
  ```yaml
  - name: Security Scan
    run: gosec ./generated/go/...
  ```

---

## Troubleshooting

### Common Issues

**1. Buffalo not found**
```bash
# Fix: Ensure Buffalo is in PATH
export PATH=$PATH:/usr/local/bin
which buffalo
```

**2. protoc not found**
```bash
# Fix: Install protoc
apt-get install -y protobuf-compiler
```

**3. Language plugin not found**
```bash
# Fix: Install required plugins
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
pip install grpcio-tools
```

**4. Permission denied**
```bash
# Fix: Make Buffalo executable
chmod +x /usr/local/bin/buffalo
```

### Debug Mode

Enable verbose logging:

```bash
buffalo build --verbose
```

Or set log level in config:

```yaml
logging:
  level: debug
```

---

## Resources

- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [GitLab CI Documentation](https://docs.gitlab.com/ee/ci/)
- [Jenkins Pipeline Documentation](https://www.jenkins.io/doc/book/pipeline/)
- [Pre-commit Framework](https://pre-commit.com/)
- [Buffalo CLI Commands](./CLI_COMMANDS.md)
- [Buffalo Configuration](./CONFIGURATION.md)

---

## Examples

Full working examples are available in `examples/ci/`:

- `github-actions-basic.yml` - Basic GitHub Actions
- `github-actions-advanced.yml` - Advanced with caching, multi-language
- `gitlab-ci.yml` - Complete GitLab CI pipeline
- `Jenkinsfile` - Jenkins pipeline
- `pre-commit-config.yaml` - Pre-commit hooks setup
- `docker-compose.yml` - Docker Compose example

---

**Version:** 1.30  
**Last Updated:** April 2026
