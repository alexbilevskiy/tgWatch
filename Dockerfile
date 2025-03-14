FROM buildpack-deps:bookworm-scm

WORKDIR /

ENV TZ=Europe/Moscow

RUN apt-get update
RUN apt-get install -y git cmake build-essential gperf libssl-dev zlib1g-dev

RUN git clone --depth 1 https://github.com/tdlib/td.git && cd td && git checkout 721300bcb4d0f2114505712f4dc6350af1ce1a09
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
#COPY --from=local-src /go-tdlib /go-tdlib

RUN make deps

COPY . .
RUN make build

CMD ["/tgWatch/tgWatch"]
