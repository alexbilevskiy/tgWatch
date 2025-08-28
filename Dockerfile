FROM buildpack-deps:bookworm-scm as tdlib

WORKDIR /

ENV TZ=Europe/Moscow

RUN apt-get update
RUN apt-get install -y git cmake build-essential gperf libssl-dev zlib1g-dev

RUN git clone https://github.com/tdlib/td.git && cd td && git checkout 971684a3dcc7bdf99eec024e1c4f57ae729d6d53
RUN cd td && mkdir build && cd build && cmake -DCMAKE_BUILD_TYPE=Release -DCMAKE_INSTALL_PREFIX:PATH=../tdlib .. && cmake --build . -j 12 && make install

FROM golang:1.24-bookworm

RUN apt-get update
RUN apt-get install -y libssl-dev zlib1g-dev

WORKDIR /tgWatch

COPY --from=tdlib /td/tdlib/ /td/tdlib/

COPY go.mod .
COPY go.sum .
RUN go mod download -x

COPY . .
RUN CGO_CFLAGS="-I/td/tdlib/include" CGO_LDFLAGS="-L/td/tdlib/lib" go build cmd/tgWatch.go

CMD ["/tgWatch/tgWatch"]
