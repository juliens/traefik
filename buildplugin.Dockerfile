FROM golang:1.25

COPY ./go.mod /traefik/go.mod
COPY ./go.sum /traefik/go.sum


WORKDIR /traefik

RUN go mod download

COPY ./cmd /traefik/cmd
COPY ./pkg /traefik/pkg
COPY ./webui /traefik/webui


RUN go build -o /tmp/traefik ./cmd/traefik

RUN mkdir /src

RUN sh -c  "go version -m /tmp/traefik > /src/version.txt"

RUN rm -rf /traefik /tmp/traefik

COPY ./script /src/script

WORKDIR /src

RUN mkdir -p ./cmd/plugindl

COPY ./go.mod /src/go.mod
COPY ./go.sum /src/go.sum

CMD bash
