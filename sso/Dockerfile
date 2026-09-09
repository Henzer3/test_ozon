FROM golang:1.26.3-alpine AS build

WORKDIR /src

COPY pkg/go.mod pkg/go.sum ./pkg/
COPY sso/go.mod sso/go.sum ./sso/

WORKDIR /src/sso
RUN go mod download

WORKDIR /src
COPY pkg ./pkg
COPY sso ./sso

WORKDIR /src/sso

ENV CGO_ENABLED=0
RUN go build -o /out/sso .

FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY --from=build /out/sso /app/sso
COPY sso/config.yaml /app/config.yaml

ENTRYPOINT ["/app/sso"]