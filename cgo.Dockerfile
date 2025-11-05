# syntax=docker/dockerfile:1.2
FROM golang

COPY ./ /src

WORKDIR /src

ENV CGO_ENABLED=1
RUN go build -o /traefik ./cmd/traefik

EXPOSE 80
VOLUME ["/tmp"]

ENTRYPOINT ["/traefik"]
