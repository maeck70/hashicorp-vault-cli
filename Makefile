.PHONY: all build clean run test vuln vulncheck

BINARY=bin/vault

all: build

build:
	@mkdir -p bin
	go build -o $(BINARY) ./cmd/vault

test:
	go test -v ./...

vuln:
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

vulncheck: vuln

run: build
	./$(BINARY)

clean:
	rm -rf bin/
