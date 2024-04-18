FROM buildpack-deps:bookworm-scm

WORKDIR /

ENV TZ=Europe/Moscow

RUN apt-get update
RUN apt-get install -y git cmake build-essential gperf libssl-dev zlib1g-dev

RUN git clone https://github.com/tdlib/td.git && cd td && git checkout d7203eb719304866a7eb7033ef03d421459335b8
RUN cd td && mkdir build && cd build && cmake -DCMAKE_BUILD_TYPE=Release .. && cmake --build . -j 4 && make install

COPY --from=golang:1.22 /usr/local/go/ /usr/local/go/

ENV PATH="/usr/local/go/bin:${PATH}"

WORKDIR /tgWatch
COPY go.mod .
COPY go.sum .
COPY Makefile .

# uncomment to build against local copy of go-tdlib;
# also add line "replace github.com/zelenin/go-tdlib => ../go-tdlib" to go.mod
# and use dockerBuildLocal.sh with custom build-context
#COPY --from=gopath /src/go-tdlib /go-tdlib

RUN make deps

COPY . .
RUN make build

CMD ["/tgWatch/tgWatch"]
