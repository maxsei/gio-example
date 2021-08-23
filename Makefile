BINARY_NAME=main

all: build wasm test

build:
	go build -o ${BINARY_NAME} main.go

wasm:
	GOARCH=wasm GOOS=js go build -o ./web/main.wasm *.go

test:
	go test -v main.go

run:
	go build -o ${BINARY_NAME} main.go
	./${BINARY_NAME}

clean:
	go clean
	rm ${BINARY_NAME}
