# Binaries will be built into the ./bin directory
.PHONY: all test import-specs-from-files seed-database clean

all: bin/mcp-client bin/openapi-mcp bin/spec-manager bin/import-specs bin/seed-database

bin/mcp-client: $(shell find pkg -type f -name '*.go') $(shell find cmd/mcp-client -type f -name '*.go')
	@mkdir -p bin
	go build -o bin/mcp-client ./cmd/mcp-client

bin/openapi-mcp: $(shell find pkg -type f -name '*.go') $(shell find cmd/openapi-mcp -type f -name '*.go')
	@mkdir -p bin
	go build -o bin/openapi-mcp ./cmd/openapi-mcp

bin/spec-manager: $(shell find pkg -type f -name '*.go') $(shell find cmd/spec-manager -type f -name '*.go')
	@mkdir -p bin
	go build -o bin/spec-manager ./cmd/spec-manager

bin/import-specs: $(shell find pkg -type f -name '*.go') $(shell find cmd/import-specs -type f -name '*.go')
	@mkdir -p bin
	go build -o bin/import-specs ./cmd/import-specs

bin/seed-database: $(shell find pkg -type f -name '*.go') $(shell find cmd/seed-database -type f -name '*.go')
	@mkdir -p bin
	go build -o bin/seed-database ./cmd/seed-database

test:
	go test ./...

import-specs-from-files:
	@echo "Importing specs from ./specs directory..."
	DATABASE_URL="${DATABASE_URL}" ./bin/import-specs

seed-database:
	@echo "Seeding database with predefined spec configuration..."
	DATABASE_URL="${DATABASE_URL}" ./bin/seed-database

clean:
	rm -f bin/mcp-client bin/openapi-mcp bin/spec-manager bin/import-specs bin/seed-database
