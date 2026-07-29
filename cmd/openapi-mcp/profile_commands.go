package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/ubermorgenland/openapi-mcp/pkg/openapi2mcp"
)

func runCompileCommand(args []string) {
	flags := flag.NewFlagSet("compile", flag.ExitOnError)
	outputPath := flags.String("output", "", "Path to the compiled MCP profile")
	_ = flags.Parse(args)
	if flags.NArg() != 1 || *outputPath == "" {
		fmt.Fprintln(os.Stderr, "Usage: openapi-mcp compile --output <profile.json> <openapi-spec.yaml>")
		os.Exit(2)
	}
	profile, err := openapi2mcp.CompileProfileFile(flags.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to compile OpenAPI spec: %v\n", err)
		os.Exit(1)
	}
	if err := openapi2mcp.WriteCompiledProfile(*outputPath, profile); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write compiled profile: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "Compiled %d MCP tools into %s\n", len(profile.Tools), *outputPath)
}

func runServeCommand(args []string) {
	flags := flag.NewFlagSet("serve", flag.ExitOnError)
	profilePath := flags.String("profile", "", "Path to a compiled MCP profile")
	httpAddr := flags.String("http", ":8080", "HTTP listen address")
	baseURL := flags.String("base-url", "", "Override the upstream API base URL")
	reloadInterval := flags.Duration("reload-interval", 2*time.Second, "How often to check the profile for updates; 0 disables reload")
	_ = flags.Parse(args)
	if flags.NArg() != 0 || *profilePath == "" {
		fmt.Fprintln(os.Stderr, "Usage: openapi-mcp serve --profile <profile.json> [--http :8080] [--base-url URL]")
		os.Exit(2)
	}
	if *baseURL != "" {
		if err := os.Setenv("OPENAPI_BASE_URL", *baseURL); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to set base URL: %v\n", err)
			os.Exit(1)
		}
	}

	runtime := &profileRuntime{profilePath: *profilePath}
	if err := runtime.reload(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load compiled profile: %v\n", err)
		os.Exit(1)
	}
	if *reloadInterval > 0 {
		go runtime.watch(*reloadInterval)
	}

	mux := http.NewServeMux()
	mux.Handle("/mcp", runtime.handler())
	mux.Handle("/mcp/", runtime.handler())
	mux.HandleFunc("/health", runtime.health)
	fmt.Fprintf(os.Stderr, "Starting stateless Streamable HTTP MCP server on %s\n", *httpAddr)
	if err := http.ListenAndServe(*httpAddr, mux); err != nil {
		fmt.Fprintf(os.Stderr, "MCP server failed: %v\n", err)
		os.Exit(1)
	}
}

type profileRuntime struct {
	profilePath string
	server      atomic.Pointer[mcp.Server]
	mu          sync.RWMutex
	profile     *openapi2mcp.CompiledProfile
	modifiedAt  time.Time
}

func (runtime *profileRuntime) reload() error {
	profile, doc, operations, err := openapi2mcp.LoadCompiledProfile(runtime.profilePath)
	if err != nil {
		return err
	}
	server := createServerWithOptions("openapi-mcp", doc.Info.Version, doc, operations, "", false)
	info, err := os.Stat(runtime.profilePath)
	if err != nil {
		return err
	}
	runtime.mu.Lock()
	runtime.profile = profile
	runtime.modifiedAt = info.ModTime()
	runtime.mu.Unlock()
	runtime.server.Store(server)
	fmt.Fprintf(os.Stderr, "Activated profile %s with %d tools\n", profile.SourceSHA256[:12], len(profile.Tools))
	return nil
}

func (runtime *profileRuntime) watch(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		info, err := os.Stat(runtime.profilePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Profile reload check failed: %v\n", err)
			continue
		}
		runtime.mu.RLock()
		modified := runtime.modifiedAt
		runtime.mu.RUnlock()
		if !info.ModTime().After(modified) {
			continue
		}
		if err := runtime.reload(); err != nil {
			fmt.Fprintf(os.Stderr, "Profile reload rejected; previous profile remains active: %v\n", err)
		}
	}
}

func (runtime *profileRuntime) handler() http.Handler {
	return mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return runtime.server.Load()
	}, &mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true, PropagateRequestCancellation: true})
}

func (runtime *profileRuntime) health(writer http.ResponseWriter, _ *http.Request) {
	runtime.mu.RLock()
	defer runtime.mu.RUnlock()
	if runtime.profile == nil {
		http.Error(writer, "no active profile", http.StatusServiceUnavailable)
		return
	}
	writer.Header().Set("Content-Type", "application/json")
	_, _ = fmt.Fprintf(writer, `{"status":"ok","profile":"%s","tools":%d}`, runtime.profile.SourceSHA256, len(runtime.profile.Tools))
}
