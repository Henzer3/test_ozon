FROM golang:1.26.3-alpine AS build

WORKDIR /src/post-service

COPY post-service/go.mod post-service/go.sum ./
COPY pkg/go.mod pkg/go.sum /src/pkg/

RUN go mod download

COPY post-service /src/post-service
COPY pkg /src/pkg

ENV CGO_ENABLED=0

RUN go build -o /out/post-service ./cmd/app

FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY --from=build /out/post-service /app/post-service
COPY post-service/config.yaml /app/config.yaml

EXPOSE 8080

ENTRYPOINT ["/app/post-service"]