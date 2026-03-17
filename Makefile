.PHONY: build test lint clean run install

BINARY=crontui
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
LDFLAGS  = -s -w -X github.com/meru143/crontui/internal/version.Version=$(VERSION) \
                   -X github.com/meru143/crontui/internal/version.Commit=$(COMMIT)

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

run: build
	./$(BINARY)

install:
	go install -ldflags "$(LDFLAGS)" .

test:
	go test ./... -v -race -cover

lint:
	golangci-lint run ./...

clean:
	rm -f $(BINARY)
	rm -f coverage.out coverage.html

coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
