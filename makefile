
## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

## tidy: format code and tidy modfile
.PHONY: tidy
tidy:
	go fmt ./...
	go mod tidy -v

# audit: run quality control checks
.PHONY: audit
audit:
	go mod verify
	go vet ./...
	go run honnef.co/go/tools/cmd/staticcheck@latest -checks=all,-ST1000,-ST1003,-U1000 ./...
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...
	go test -race -buildvcs -vet=off ./...

## test: run all tests
.PHONY: test
test:
	go test -tags dev -v ./...

## race_test: run all tests with race detector
.PHONY: race_test
race_test:
	go test -v -tags dev -race -buildvcs ./...

## test/cover: run all tests and display coverage
.PHONY: test/cover
test/cover:
	go test -v -tags dev -race -buildvcs -coverprofile=/tmp/coverage.out ./...
	go tool cover -html=/tmp/coverage.out

## build: build the application
.PHONY: build
build:
	@echo "Building frontend..."
	(cd ui && pnpm install && pnpm run build)
	@echo "Building backend..."
	@VERSION=$$(git describe --tags --exact-match 2>/dev/null || echo "dev"); \
	COMMIT=$$(git rev-parse --short HEAD 2>/dev/null || echo "unknown"); \
	echo "  Version: $$VERSION"; \
	echo "  Commit: $$COMMIT"; \
	go build -ldflags "-X github.com/geerew/off-course/version.Version=$$VERSION -X github.com/geerew/off-course/version.Commit=$$COMMIT" -o offcourse .
	@echo "Build complete: offcourse"
