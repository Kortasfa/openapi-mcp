package openapi2mcp

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCompiledProfileRoundTrip(t *testing.T) {
	profile, err := CompileProfile([]byte(`openapi: 3.0.0
info:
  title: Example
  version: 1.0.0
paths:
  /items:
    get:
      operationId: listItems
      responses:
        '200':
          description: OK
`))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "profile.json")
	if err := WriteCompiledProfile(path, profile); err != nil {
		t.Fatal(err)
	}
	loaded, doc, operations, err := LoadCompiledProfile(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.SourceSHA256 != profile.SourceSHA256 || doc.Info.Title != "Example" || len(operations) != 1 {
		t.Fatalf("unexpected profile result: %#v, %#v", loaded, operations)
	}
	if err := os.WriteFile(path, []byte(`{"version":1,"sourceSha256":"invalid"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := LoadCompiledProfile(path); err == nil {
		t.Fatal("expected invalid profile to fail")
	}
	profile.Tools[0].OperationID = "tampered"
	if err := WriteCompiledProfile(path, profile); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := LoadCompiledProfile(path); err == nil {
		t.Fatal("expected tampered tools to fail")
	}
}
