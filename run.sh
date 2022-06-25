#!/usr/bin/env bash
export CGO_CFLAGS="-I/opt/src/td/tdlib/include" CGO_LDFLAGS="-L/opt/src/td/tdlib/lib"
if [[ $1 ]]
then
  echo "single account: $1";
   go build cmd/tgWatch.go && ./tgWatch $1;
else
  echo "all accounts";
  go build cmd/tgWatch.go && ./tgWatch;
fi