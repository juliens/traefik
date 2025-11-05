FROM golang:1.25

COPY ./go.mod /traefik/go.mod
COPY ./go.sum /traefik/go.sum


WORKDIR /traefik

RUN go mod download

COPY . /traefik

CMD go run ./cmd/builder
