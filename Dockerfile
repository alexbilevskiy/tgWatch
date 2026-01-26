FROM buildpack-deps:bookworm-scm AS tdlib

WORKDIR /

ENV TZ=Europe/Moscow

RUN apt-get update
RUN apt-get install -y git cmake build-essential gperf libssl-dev zlib1g-dev
# or with clang also
#RUN apt-get install -y clang libc++-dev libc++abi-dev

RUN git clone https://github.com/tdlib/td.git && cd td && git checkout cb863c1600082404428f1a84e407b866b9d412a8
RUN mkdir td/build

WORKDIR td/build

# NOTICE: must change ldflags in https://github.com/zelenin/go-tdlib/blob/master/client/tdjson_static.go, if compile with clang
#RUN CXXFLAGS="-stdlib=libc++" CC=/usr/bin/clang CXX=/usr/bin/clang++ cmake -DCMAKE_BUILD_TYPE=Release -DCMAKE_INSTALL_PREFIX:PATH=../tdlib ..
#RUN cmake --build . --target install -j 12

RUN cmake -DCMAKE_BUILD_TYPE=Release -DCMAKE_INSTALL_PREFIX:PATH=../tdlib ..
RUN cmake --build . -j 12 && make install

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
