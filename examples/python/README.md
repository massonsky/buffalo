# Python gRPC Example

This example demonstrates using Buffalo to generate Python code from protobuf definitions.

## Prerequisites

```bash
# Install protobuf compiler
# On Ubuntu/Debian:
sudo apt-get install protobuf-compiler

# On macOS:
brew install protobuf

# On Windows:
# Download from https://github.com/protocolbuffers/protobuf/releases

# Install Python dependencies
pip install grpcio grpcio-tools
```

## Generate Python Code

Run Buffalo from the repository root:

```bash
# Build Buffalo (if not already built)
go build -o bin/buffalo.exe ./cmd/buffalo

# Generate Python code
./bin/buffalo.exe build --lang python
```

Or from this directory:

```bash
../../bin/buffalo.exe build --lang python
```

## Generated Files

Buffalo will generate:

```
generated/
└── python/
    ├── __init__.py
    ├── greeter_pb2.py        # Protobuf message classes
    └── greeter_pb2_grpc.py   # gRPC service stubs
```

## Using the Generated Code

### Server Example

```python
import grpc
from concurrent import futures
import time

# Import generated code
from generated.python import greeter_pb2
from generated.python import greeter_pb2_grpc


class GreeterServicer(greeter_pb2_grpc.GreeterServicer):
    def SayHello(self, request, context):
        message = f"Hello, {request.name}!"
        return greeter_pb2.HelloResponse(
            message=message,
            timestamp=int(time.time())
        )
    
    def SayHelloStream(self, request, context):
        for i in range(request.count):
            message = f"Hello #{i+1}, {request.name}!"
            yield greeter_pb2.HelloResponse(
                message=message,
                timestamp=int(time.time())
            )
            time.sleep(1)


def serve():
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    greeter_pb2_grpc.add_GreeterServicer_to_server(
        GreeterServicer(), server
    )
    server.add_insecure_port('[::]:50051')
    server.start()
    print("Server started on port 50051")
    server.wait_for_termination()


if __name__ == '__main__':
    serve()
```

### Client Example

```python
import grpc

# Import generated code
from generated.python import greeter_pb2
from generated.python import greeter_pb2_grpc


def run():
    # Create a channel
    with grpc.insecure_channel('localhost:50051') as channel:
        # Create a stub
        stub = greeter_pb2_grpc.GreeterStub(channel)
        
        # Call SayHello
        response = stub.SayHello(
            greeter_pb2.HelloRequest(name="World", count=1)
        )
        print(f"Response: {response.message}")
        
        # Call SayHelloStream
        print("\\nStreaming responses:")
        for response in stub.SayHelloStream(
            greeter_pb2.HelloRequest(name="Stream", count=3)
        ):
            print(f"  {response.message}")


if __name__ == '__main__':
    run()
```

## Configuration

See `buffalo.yaml` for configuration options:

- `proto.paths`: Directories containing .proto files
- `output.per_language.python`: Output directory for Python code
- `languages.python.options.grpc`: Enable gRPC code generation
- `build.workers`: Number of parallel compilation workers
- `build.incremental`: Enable incremental builds with caching

## Testing

```bash
# Generate code
../../bin/buffalo.exe build --lang python

# Run server (in one terminal)
python server.py

# Run client (in another terminal)
python client.py
```

## Type Hints

For better IDE support and type checking, you can enable .pyi stub generation:

```yaml
languages:
  python:
    options:
      typing: true
```

Then use mypy for type checking:

```bash
pip install mypy
mypy server.py
```
