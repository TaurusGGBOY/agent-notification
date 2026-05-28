.PHONY: build clean test

build:
	GOOS=windows GOARCH=amd64 go build -ldflags="-H=windowsgui" -o agent-notify-server.exe

build-arm:
	GOOS=windows GOARCH=arm64 go build -ldflags="-H=windowsgui" -o agent-notify-server-arm64.exe

clean:
	rm -f agent-notify-server.exe agent-notify-server-arm64.exe

test:
	go test -v ./...

run:
	go run .

deps:
	go mod tidy
