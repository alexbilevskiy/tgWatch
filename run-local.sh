#!/usr/bin/env bash
set -x
CGO_CFLAGS="-I/opt/src/td/tdlib/include/" CGO_LDFLAGS="-L/opt/src/td/tdlib/lib" go run cmd/test/main.go
