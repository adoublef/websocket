# syntax=docker/dockerfile:1

ARG GO_VERSION=1.21
ARG DISTROLESS=static-debian11:nonroot-amd64

FROM golang:${GO_VERSION} AS base
WORKDIR /usr/src

COPY go.* .
RUN go mod download

COPY . .

FROM base AS test
RUN go test -v -cover -count 1 ./...

FROM base AS build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags "-w -s" \
    -buildvcs=false \
    -o /usr/bin/a .

# do something

FROM gcr.io/distroless/${DISTROLESS} AS final
WORKDIR /opt

USER nonroot:nonroot
COPY --from=build --chown=nonroot:nonroot /usr/bin/a .

CMD ["./a"]

EXPOSE 8080