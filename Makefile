deps:
	go mod download
build:
	go build cmd/tgWatch.go
build-local:
	CGO_CFLAGS="-I/opt/src/td/tdlib/include" CGO_LDFLAGS="-L/opt/src/td/tdlib/lib" go build cmd/tgWatch.go
