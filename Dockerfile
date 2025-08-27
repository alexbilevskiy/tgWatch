FROM buildpack-deps:bookworm-scm as tdlib

WORKDIR /

ENV TZ=Europe/Moscow

RUN apt-get update
RUN apt-get install -y git cmake build-essential gperf libssl-dev zlib1g-dev

RUN git clone https://github.com/tdlib/td.git && cd td && git checkout 721300bcb4d0f2114505712f4dc6350af1ce1a09
RUN cd td && mkdir build && cd build && cmake -DCMAKE_BUILD_TYPE=Release -DCMAKE_INSTALL_PREFIX:PATH=../tdlib .. && cmake --build . -j 4 && make install

FROM golang:1.22-bookworm

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
