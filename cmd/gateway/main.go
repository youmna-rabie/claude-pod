package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("claude-pod gateway starting...")

	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// TODO: load config, wire dependencies, start server
	fmt.Println("gateway stub — not yet implemented")
	return nil
}
