CONTAINER_RUNTIME ?= docker

GO_BIN := $(shell go env GOPATH)/bin
export PATH := $(GO_BIN):$(PATH)

up: down
	$(CONTAINER_RUNTIME) compose up --build -d

down:
	$(CONTAINER_RUNTIME) compose down

clean:
	$(CONTAINER_RUNTIME) compose down -v

run-tests:
	$(CONTAINER_RUNTIME) run --rm --network=host tests:latest

test:
	make clean
	make up
	@echo wait cluster to start && sleep 10
	make run-tests
	make clean
	@echo "test finished"

generate: generate-graphql generate-proto

generate-graphql:
	go -C post-service tool gqlgen generate

generate-proto:
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		pkg/sso/sso.proto

unit-test:
	go test -cover ./post-service/... ./sso/... ./pkg/...
