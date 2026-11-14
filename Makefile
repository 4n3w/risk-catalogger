BINARY_NAME=riskcatalog
# Arguments for `make run`, e.g. make run ARGS="-f my-catalog.yml -markdown"
ARGS ?=

.PHONY: all build run clean test

all: build test

build:
	GOARCH=arm64 GOOS=darwin go build -o bin/${BINARY_NAME}-darwin-arm64 ./cmd
	GOARCH=amd64 GOOS=darwin go build -o bin/${BINARY_NAME}-darwin-amd64 ./cmd
	GOARCH=amd64 GOOS=linux go build -o bin/${BINARY_NAME}-linux-amd64 ./cmd

run:
	go build -o ${BINARY_NAME} ./cmd
	./${BINARY_NAME} ${ARGS}

clean:
	go clean
	rm -f ${BINARY_NAME} bin/${BINARY_NAME}-darwin-arm64 bin/${BINARY_NAME}-darwin-amd64 bin/${BINARY_NAME}-linux-amd64

test:
	go test ./...
