.PHONY: oapi-gen gen build tidy lint test all integration_cleanup integration_build integration_up integration

ifneq (,$(wildcard ./.env))
    include .env
    export
endif

TESTARGS ?= ""

all: tidy build test lint

# ad5eada4f3ccc28a88477cef62ea21c17fc8aa01 -> v2.4.1
OAPI_CODEGEN_CMD = go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@9c09ef9e9d4be639bd3feff31ff2c06961421272 --config=./oapi-codegen.yaml
OAPI_SPEC = ./pkg/api
OAPI_YAML = ./openapi.yaml

oapi-gen: $(OAPI_SPEC)
	@$(OAPI_CODEGEN_CMD) --generate=types -o $(OAPI_SPEC)/types.gen.go $(OAPI_YAML)
	@$(OAPI_CODEGEN_CMD) --generate=client -o $(OAPI_SPEC)/client.gen.go $(OAPI_YAML)
	@$(OAPI_CODEGEN_CMD) --generate=spec -o $(OAPI_SPEC)/spec.gen.go $(OAPI_YAML)
	@$(OAPI_CODEGEN_CMD) --generate=std-http-server,strict-server -o $(OAPI_SPEC)/server.gen.go $(OAPI_YAML)

$(OAPI_SPEC):
	@mkdir -p $@

gen: oapi-gen

build: gen
	go build ./...

tidy:
	go mod tidy

lint:
	go run github.com/golangci/golangci-lint/cmd/golangci-lint@latest run --timeout 5m

%:
    @:

docker:
	@docker compose build && docker compose run --rm --service-ports app

test:
	go test -count 100 -p 10 -parallel 10 -race ./...

integration_cleanup:
	@docker compose -f docker-compose.yaml -f docker-compose.test.yaml down --volumes

integration_build:
	@docker compose -f docker-compose.yaml -f docker-compose.test.yaml build

integration_up:
	@docker compose -f docker-compose.yaml -f docker-compose.test.yaml up --abort-on-container-exit --exit-code-from test

integration: integration_cleanup integration_build integration_up integration_cleanup
