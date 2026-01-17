#!/usr/bin/env python3
"""Test script for generated protobuf code"""

import sys
sys.path.insert(0, 'generated')

# Import generated code
from protos import example_pb2, example_pb2_grpc

# Test message creation
request = example_pb2.ExampleRequest(id="test-123")
print(f"✅ Created ExampleRequest: id={request.id}")

response = example_pb2.ExampleResponse(
    id="test-123",
    name="Test Example",
    value=42
)
print(f"✅ Created ExampleResponse: id={response.id}, name={response.name}, value={response.value}")

# Test serialization
serialized = response.SerializeToString()
print(f"✅ Serialized to {len(serialized)} bytes")

# Test deserialization
response2 = example_pb2.ExampleResponse()
response2.ParseFromString(serialized)
print(f"✅ Deserialized: id={response2.id}, name={response2.name}, value={response2.value}")

# Show available services
print(f"\n✅ Available gRPC stub: {example_pb2_grpc.ExampleServiceStub}")

print("\n🎉 All tests passed! Python code generation is working correctly.")
