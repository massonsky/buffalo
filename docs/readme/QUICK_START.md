# 🚀 Quick Start

## 1) Initialize project

```bash
buffalo init myproject
cd myproject
```

## 2) Add proto file

```protobuf
syntax = "proto3";

package myservice;

service Greeter {
  rpc SayHello (HelloRequest) returns (HelloReply) {}
}

message HelloRequest {
  string name = 1;
}

message HelloReply {
  string message = 1;
}
```

## 3) Configure `buffalo.yaml`

```yaml
project:
  name: myproject

proto:
  paths:
    - ./protos

languages:
  python:
    enabled: true
  go:
    enabled: true
```

## 4) Build

```bash
buffalo build
```

## 5) Useful next commands

```bash
buffalo doctor
buffalo validate -p ./protos
buffalo watch
```
