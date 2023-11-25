APP?=mocha
LINT_VERSION = v1.54.2

.PHONY: help
## help: prints this help message
help:
	@echo "Usage: \n"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':'

.PHONY: install-tools
## install-tools: installs dependencies for tools
install-tools:
	@echo Installing tools from tools.go
	@cat tools.go | grep _ | awk -F'"' '{print $$2}' | xargs -tI % go install %

.PHONY: build
## build: builds the application
build:
	CGO_ENABLED=0 go build -a -installsuffix cgo -ldflags="-w -s" -o bin/${APP} cmd/server/main.go

.PHONY: fix-import-order
## fix-import-order: fixes import order
fix-import-order: install-tools
	gci write --skip-generated -s "standard" -s "default" --custom-order .

.PHONY: format-go
## format-go: formats go files
format-go: install-tools fix-import-order
	gofmt -s -w .
	go mod tidy

.PHONY: format
## format: formats files
format: format-go

.PHONY: test
## test: runs tests
test: install-tools
	gotest -p 1 -race -cover -v ./internal/...

.PHONY: unit-test
## unit-test: runs unit tests
unit-test: install-tools
	gotest -p 60 -race -coverpkg ./pkg/...,./internal/... -coverprofile=coverage.out -v ./pkg/... ./internal/...

.PHONY: coverage
## coverage: runs tests with coverage
coverage: unit-test
	grep -Ev "_mock.go|generated.go|_client.go|grpc_gateway.go|grpc_server.go|fake_*.go|_gen.go" coverage.out > coverage.filtered.out

.PHONY: generate-mock
## generate-mock: generates mock files
generate-mock: install-tools
	go generate ./...

.PHONY: generate
## generate: generates files
generate: generate-mock

.PHONY: lint
## lint: lints files
lint: install-tools
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin $(LINT_VERSION)
	golangci-lint run -c .golangci.yml ./...
	go mod verify

.PHONY: diff
## diff: shows diff
diff:
	git diff --exit-code
	if [ -n "$(git status --porcelain)" ]; then git status; exit 1; else exit 0; fi

.PHONY: check
check: generate format-go lint test
