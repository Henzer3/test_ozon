FROM golang:1.26.3-alpine

RUN apk add --no-cache build-base

ENV CGO_ENABLED=1

COPY . /tests
WORKDIR /tests

ENTRYPOINT ["go", "test", "-race", "-v", "./..."]