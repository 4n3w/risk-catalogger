BINARY_NAME=riskcatalog

all: build test

build:
	GOARCH=amd64 GOOS=darwin go build -o bin/${BINARY_NAME}-darwin cmd/main.go
	GOARCH=amd64 GOOS=linux go build -o bin/${BINARY_NAME}-linux cmd/main.go

run:
	go build -o ${BINARY_NAME} cmd/main.go
	./${BINARY_NAME}

clean:
	go clean
	rm ${BINARY_NAME}-darwin
	rm ${BINARY_NAME}-linux

test:
	go test -v pkg/riskcatalog/incident_test.go