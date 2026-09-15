.PHONY: all build clean run test

BINARY=bin/vault

all: build

build:
	@mkdir -p bin
	go build -o $(BINARY) .

test:
	go test -v ./...

run: build
	./$(BINARY)

clean:
	rm -rf bin/
