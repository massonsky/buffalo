package main

import (
	"fmt"
	"os"

	"github.com/massonsky/buffalo/internal/version"
)

func main() {
	fmt.Println("🦬 Buffalo - Protobuf/gRPC Multi-Language Builder")
	fmt.Println(version.Info())
	fmt.Println("\nProject initialized! Coming soon...")
	os.Exit(0)
}
