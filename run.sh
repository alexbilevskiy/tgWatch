#!/usr/bin/env bash
export CGO_CFLAGS="-I/opt/src/td/tdlib/include -I/opt/src/vosk-api/src" CGO_LDFLAGS="-L/opt/src/td/tdlib/lib -L/opt/src/vosk-api/src -L/opt/src/kaldi/tools/openfst/src -L/opt/src/kaldi/tools/OpenBLAS/install/lib"
if [[ $1 ]]
then
  echo "single account: $1";
   go build cmd/tgWatch.go && ./tgWatch $1;
else
  echo "all accounts";
  go build cmd/tgWatch.go && ./tgWatch;
fi