#!/usr/bin/env bash

#HOST=my-server.net
#PORT=443
#TLS="--tls"

HOST=192.168.88.48
PORT=8092
TLS=


set -x
docker run --net=host --rm -i -t -v "$(pwd):/mount:ro"  ghcr.io/ktr0731/evans:latest --host $HOST --port $PORT --path /vendor --path /mount --proto schemas/proto/api/tgwatch.proto $TLS
