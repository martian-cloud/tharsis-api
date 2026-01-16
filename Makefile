PROTOC_VERSION=25.6
GO_VERSION = 1.24
MODULE = $(shell go list -m)
VERSION ?= $(shell git describe --tags --always --dirty --match=v* 2> /dev/null || echo "1.0.0")
BUILD_TIMESTAMP ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
PACKAGES := $(shell go list -tags noui ./... | grep -v /vendor/)
LDFLAGS := -ldflags "-X main.Version=${VERSION} -X main.BuildTimestamp=${BUILD_TIMESTAMP}"

DB_URI ?= pgx://postgres:postgres@localhost:5432/tharsis?sslmode=disable#gitleaks:allow
MIGRATE := docker run -v $(shell pwd)/internal/db/migrations:/migrations --network host migrate/migrate:v4.18.3 -path=/migrations/ -database "$(DB_URI)"

# Build targets
.PHONY: build-tharsis
build-tharsis:  ## build the tharsis binary (includes UI)
	@echo "Building UI..."
	@cd frontend && npm install >/dev/null && npm run build
	@echo "Building tharsis binary..."
	CGO_ENABLED=0 go build ${LDFLAGS} -a -o apiserver $(MODULE)/cmd/apiserver

.PHONY: build-api
build-api:  ## build the API binary (no UI)
	CGO_ENABLED=0 go build ${LDFLAGS} -tags noui -a -o apiserver $(MODULE)/cmd/apiserver

.PHONY: build-job-executor
build-job-executor:  ## build the binaries
	CGO_ENABLED=0 go build ${LDFLAGS} -a -o job $(MODULE)/cmd/job

.PHONY: build-runner
build-runner:  ## build the binaries
	CGO_ENABLED=0 go build ${LDFLAGS} -a -o runner $(MODULE)/cmd/runner

# Code quality targets
.PHONY: lint
lint: ## run linting on Go and UI code
	@echo "Linting Go code..."
	@revive -set_exit_status $(PACKAGES)
	@echo "Checking Go formatting..."
	@UNFORMATTED=$$(gofmt -l . 2>/dev/null | grep -v vendor | grep -v testdata | grep -v '/pkg/mod/'); \
	if [ -n "$$UNFORMATTED" ]; then \
		echo "Files not formatted:"; \
		echo "$$UNFORMATTED"; \
		exit 1; \
	fi
	@echo "Linting UI code..."
	@cd frontend && npm install >/dev/null && npm run lint

.PHONY: vet
vet: ## run golint on all Go package
	@go vet -tags noui ./...

.PHONY: fmt
fmt: ## run "go fmt" on all Go packages
	@go fmt $(PACKAGES)

.PHONY: generate
generate: ## run go generate
	go generate -v ./...

.PHONY: protos
protos: ## generate code from a .proto file with protoc cli.
	@echo "Verify correct protoc version installed"
	@if [ $(shell which protoc | wc -l) = 0 ] || [ "$(shell protoc --version | awk '{print $$2}')" != $(PROTOC_VERSION) ]; then \
  		echo "Required protoc version is not installed. $(shell protoc --version | awk '{print $$2}') detected, $(PROTOC_VERSION) required."; \
  		echo "You can install the correct version from https://github.com/protocolbuffers/protobuf/releases/tag/v$(PROTOC_VERSION) or consider using nix."; \
		echo "See installation instructions https://grpc.io/docs/protoc-installation/#install-pre-compiled-binaries-any-os"; \
  		exit 1; \
	fi

	@go install tool

	@echo "Generating code from protos"
	protoc --go_out=pkg/protos/gen --go_opt=paths=source_relative \
		--go-grpc_out=pkg/protos/gen --go-grpc_opt=paths=source_relative \
		--proto_path=pkg/protos pkg/protos/*.proto

	@echo "Protos successfully generated"

# Test targets
.PHONY: test
test: ## run unit tests
	go test -tags noui ./...

.PHONY: integration
integration: ## run DB layer integration tests
	test/integration/run-integration-tests.sh

# UI targets
.PHONY: build-ui
build-ui: ## build the UI for production
	cd frontend && npm install >/dev/null && npm run build

.PHONY: dev-ui
dev-ui: ## start the UI development server
	cd frontend && npm install && npm start

.PHONY: relay
relay: ## run the UI relay compiler
	cd frontend && npm run relay

.PHONY: update-schema
update-schema: ## update the GraphQL schema
	cd frontend && npm run update-schema

# Docker targets
.PHONY: build-tharsis-docker
build-tharsis-docker:
	docker build --build-arg goversion=$(GO_VERSION) --target tharsis -t tharsis/tharsis .

.PHONY: build-job-docker
build-job-docker:
	docker build --build-arg goversion=$(GO_VERSION) --target job-executor -t tharsis/job-executor .

.PHONY: build-runner-docker
build-runner-docker:
	docker build --build-arg goversion=$(GO_VERSION) --target runner -t tharsis/runner .

.PHONY: run-tharsis-docker
run-tharsis-docker:
	docker run -it -p 8000:8000 -p 9090:9090 -v ~/.aws:/root/.aws --env-file .env.docker tharsis/tharsis

# Database targets
.PHONY: db-start
db-start: ## start the database server
	@mkdir -p testdata/postgres
	docker run --rm --name postgres -v $(shell pwd)/testdata:/testdata \
		-v $(shell pwd)/testdata/postgres:/var/lib/postgresql/data \
		-e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=tharsis -d -p 5432:5432 postgres:16

.PHONY: db-stop
db-stop: ## stop the database server
	docker stop postgres

.PHONY: migrate
migrate: ## run all new database migrations
	@echo "Running all new database migrations..."
	@$(MIGRATE) -verbose up

.PHONY: migrate-down
migrate-down: ## revert database to the last migration step
	@echo "Reverting database to the last migration step..."
	@$(MIGRATE) -verbose down 1

.PHONY: migrate-new
migrate-new: ## create a new database migration
	@read -p "Enter the name of the new migration: " name; \
	$(MIGRATE) create -ext sql -dir /migrations/ $${name// /_}

.PHONY: migrate-force
migrate-force: ## forces migrate version but doesn't run migration
	@read -p "Enter the version to force the migration to: " version; \
	$(MIGRATE) force $${version// /_}

.PHONY: migrate-reset
migrate-reset: ## reset database and re-run all migrations
	@echo "Resetting database..."
	@$(MIGRATE) -verbose drop -f
	@echo "Running all database migrations..."
	@$(MIGRATE) -verbose up
