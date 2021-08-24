BINARY_NAME=main
GOOS=$$(go env GOOS )
GOARCH=$$(go env GOARCH )

BINARY_PATH=./bin/${GOOS}-${GOARCH}-${BINARY_NAME}

all: build windows wasm test

build:
	go build -o ${BINARY_PATH}

windows:
	GOOS=windows GOARCH=amd64	go build -ldflags="-H windowsgui" -o ./bin/windows-amd64-${BINARY_NAME}.exe

wasm:
	GOARCH=wasm GOOS=js go build -o ./web/${BINARY_NAME}.wasm *.go

test:
	go test -v *.go

run:
	go build -o ${BINARY_PATH} *.go
	${BINARY_PATH}

clean:
	go clean
	rm ./bin/${BINARY_NAME}
