FROM buildpack-deps:bookworm-scm

WORKDIR /

ENV TZ=Europe/Moscow

RUN apt-get update
RUN apt-get install -y git cmake build-essential gperf libssl-dev zlib1g-dev

RUN git clone https://github.com/tdlib/td.git && cd td && git checkout ec788c7505c4f2b31b59743d2f4f97d6fdcba451
RUN cd td && mkdir build && cd build && cmake -DCMAKE_BUILD_TYPE=Release .. && cmake --build . -j 4 && make install

COPY --from=golang:1.18 /usr/local/go/ /usr/local/go/

ENV PATH="/usr/local/go/bin:${PATH}"

WORKDIR /tgWatch
COPY . .

RUN make build

COPY config.json ./

CMD ["/tgWatch/tgWatch"]
