package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/ubermorgenland/openapi-mcp/pkg/openapi2mcp"
)

func startServer(flags *cliFlags, ops []openapi2mcp.OpenAPIOperation, doc *openapi3.T) {
	if flags.httpAddr != "" && len(flags.mounts) > 0 {
		mux := http.NewServeMux()
		for _, mount := range flags.mounts {
			d, err := openapi3.NewLoader().LoadFromFile(mount.SpecPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to load %s: %v\n", mount.SpecPath, err)
				os.Exit(1)
			}
			srv := createServerWithOptions("openapi-mcp", d.Info.Version, d, openapi2mcp.ExtractOpenAPIOperations(d), flags.logFile, flags.noLogTruncation)
			handler := openapi2mcp.HandlerForStreamableHTTP(srv, mount.BasePath)
			mux.Handle(mount.BasePath, handler)
			mux.Handle(mount.BasePath+"/", handler)
		}
		fmt.Fprintf(os.Stderr, "Starting stateless Streamable HTTP server on %s\n", flags.httpAddr)
		if err := http.ListenAndServe(flags.httpAddr, mux); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to start HTTP server: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if doc == nil && len(flags.args) > 0 {
		var err error
		doc, err = openapi2mcp.LoadOpenAPISpec(flags.args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to load OpenAPI spec: %v\n", err)
			os.Exit(1)
		}
		ops = openapi2mcp.ExtractOpenAPIOperations(doc)
	}
	if doc == nil {
		fmt.Fprintln(os.Stderr, "No OpenAPI specification available")
		os.Exit(1)
	}

	srv := createServerWithOptions("openapi-mcp", doc.Info.Version, doc, ops, flags.logFile, flags.noLogTruncation)
	if flags.dryRun || flags.docFile != "" || flags.summary {
		return
	}
	if flags.httpAddr != "" {
		mux := http.NewServeMux()
		handler := openapi2mcp.HandlerForStreamableHTTP(srv, "/mcp")
		mux.Handle("/mcp", handler)
		mux.Handle("/mcp/", handler)
		fmt.Fprintf(os.Stderr, "Starting stateless Streamable HTTP server on %s\n", flags.httpAddr)
		if err := http.ListenAndServe(flags.httpAddr, mux); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to start HTTP server: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if err := srv.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start MCP stdio server: %v\n", err)
		os.Exit(1)
	}
}

func createServerWithOptions(name, version string, doc *openapi3.T, ops []openapi2mcp.OpenAPIOperation, logFile string, _ bool) *mcp.Server {
	if logFile != "" {
		fmt.Fprintf(os.Stderr, "Warning: --log-file is no longer supported by the official MCP SDK\n")
	}
	srv := mcp.NewServer(&mcp.Implementation{Name: name, Version: version}, &mcp.ServerOptions{
		Capabilities: &mcp.ServerCapabilities{},
		SchemaCache:  mcp.NewSchemaCache(),
	})
	openapi2mcp.RegisterOpenAPITools(srv, ops, doc, nil)
	return srv
}
