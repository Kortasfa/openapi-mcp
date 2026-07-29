package openapi2mcp

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
)

const CompiledProfileVersion = 2

type CompiledTool struct {
	OperationID string         `json:"operationId"`
	Path        string         `json:"path"`
	Method      string         `json:"method"`
	InputSchema map[string]any `json:"inputSchema"`
}

type CompiledProfile struct {
	Version      int            `json:"version"`
	GeneratedAt  time.Time      `json:"generatedAt"`
	SourceSHA256 string         `json:"sourceSha256"`
	OpenAPISpec  string         `json:"openapiSpec"`
	Tools        []CompiledTool `json:"tools"`
}

func CompileProfile(source []byte) (*CompiledProfile, error) {
	doc, err := LoadOpenAPISpecFromBytes(source)
	if err != nil {
		return nil, err
	}
	operations := ExtractOpenAPIOperations(doc)
	tools := make([]CompiledTool, 0, len(operations))
	for _, operation := range operations {
		tools = append(tools, CompiledTool{
			OperationID: operation.OperationID,
			Path:        operation.Path,
			Method:      operation.Method,
			InputSchema: BuildInputSchemaWithContext(operation.Parameters, operation.RequestBody, doc),
		})
	}
	sort.Slice(tools, func(left, right int) bool { return tools[left].OperationID < tools[right].OperationID })
	digest := sha256.Sum256(source)
	return &CompiledProfile{
		Version:      CompiledProfileVersion,
		GeneratedAt:  time.Now().UTC(),
		SourceSHA256: hex.EncodeToString(digest[:]),
		OpenAPISpec:  string(source),
		Tools:        tools,
	}, nil
}

func CompileProfileFile(sourcePath string) (*CompiledProfile, error) {
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		return nil, err
	}
	return CompileProfile(source)
}

func WriteCompiledProfile(path string, profile *CompiledProfile) error {
	if profile == nil {
		return fmt.Errorf("compiled profile is required")
	}
	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	temporaryPath := path + ".tmp"
	if err := os.WriteFile(temporaryPath, data, 0o600); err != nil {
		return err
	}
	return os.Rename(temporaryPath, path)
}

func LoadCompiledProfile(path string) (*CompiledProfile, *openapi3.T, []OpenAPIOperation, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, nil, err
	}
	var profile CompiledProfile
	if err := json.Unmarshal(data, &profile); err != nil {
		return nil, nil, nil, err
	}
	if profile.Version != CompiledProfileVersion {
		return nil, nil, nil, fmt.Errorf("unsupported compiled profile version %d", profile.Version)
	}
	digest := sha256.Sum256([]byte(profile.OpenAPISpec))
	if profile.SourceSHA256 != hex.EncodeToString(digest[:]) {
		return nil, nil, nil, fmt.Errorf("compiled profile source checksum does not match")
	}
	expected, err := CompileProfile([]byte(profile.OpenAPISpec))
	if err != nil {
		return nil, nil, nil, err
	}
	actualTools, err := json.Marshal(profile.Tools)
	if err != nil {
		return nil, nil, nil, err
	}
	expectedTools, err := json.Marshal(expected.Tools)
	if err != nil {
		return nil, nil, nil, err
	}
	if !bytes.Equal(actualTools, expectedTools) {
		return nil, nil, nil, fmt.Errorf("compiled profile tools do not match its OpenAPI source")
	}
	doc, err := LoadOpenAPISpecFromBytes([]byte(profile.OpenAPISpec))
	if err != nil {
		return nil, nil, nil, err
	}
	operations := ExtractOpenAPIOperations(doc)
	if len(operations) != len(profile.Tools) {
		return nil, nil, nil, fmt.Errorf("compiled profile contains %d tools, but source resolves to %d", len(profile.Tools), len(operations))
	}
	return &profile, doc, operations, nil
}
