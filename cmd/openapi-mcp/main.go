package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "compile":
		runCompileCommand(os.Args[2:])
	case "serve":
		runServeCommand(os.Args[2:])
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(2)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, `Usage:
  openapi-mcp compile --output <profile.json> <openapi-spec.yaml>
  openapi-mcp serve --profile <profile.json> [--http :8080] [--base-url URL]

The server exposes stateless Streamable HTTP at /mcp. MCP clients provide their
own Authorization: Bearer token; the server stores no upstream credentials.`)
}
