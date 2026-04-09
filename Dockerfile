ARG DOCKER_REGISTRY
FROM ${DOCKER_REGISTRY}/tdlib:latest AS tdlib
FROM golang:1.24-bookworm

RUN apt-get update
RUN apt-get install -y libssl-dev zlib1g-dev

WORKDIR /tgwatch

COPY --from=tdlib /td/tdlib/ /td/tdlib/

COPY go.mod .
COPY go.sum .
#RUN go mod download -x
COPY vendor .

COPY . .
RUN CGO_CFLAGS="-I/td/tdlib/include" CGO_LDFLAGS="-L/td/tdlib/lib" go build -o tgwatch cmd/tgwatch/main.go

CMD ["/tgwatch/tgwatch"]
