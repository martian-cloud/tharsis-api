GO_VERSION = 1.20.2
MODULE = $(shell go list -m)
VERSION ?= $(shell git describe --tags --always --dirty --match=v* 2> /dev/null || echo "1.0.0")
PACKAGES := $(shell go list ./... | grep -v /vendor/)
LDFLAGS := -ldflags "-X main.Version=${VERSION}"

DB_URI ?= pgx://postgres:postgres@localhost:5432/tharsis?sslmode=disable
MIGRATE := docker run -v $(shell pwd)/internal/db/migrations:/migrations --network host migrate/migrate:v4.15.2 -path=/migrations/ -database "$(DB_URI)"

.PHONY: build-api
build-api:  ## build the binaries
	CGO_ENABLED=0 go build ${LDFLAGS} -a -o apiserver $(MODULE)/cmd/apiserver

.PHONY: build-job-executor
build-job-executor:  ## build the binaries
	CGO_ENABLED=0 go build ${LDFLAGS} -a -o job $(MODULE)/cmd/job

.PHONY: build-runner
build-runner:  ## build the binaries
	CGO_ENABLED=0 go build ${LDFLAGS} -a -o runner $(MODULE)/cmd/runner

.PHONY: lint
lint: ## run golint on all Go package
	@golint -set_exit_status $(PACKAGES)

.PHONY: vet
vet: ## run golint on all Go package
	@go vet $(PACKAGES)

.PHONY: fmt
fmt: ## run "go fmt" on all Go packages
	@go fmt $(PACKAGES)

.PHONY: test
test: ## run unit tests
	go test ./...

.PHONY: integration
integration: ## run DB layer integration tests
	test/integration/run-integration-tests.sh

.PHONY: generate
generate: ## run go generate
	go generate -v ./...

.PHONY: build-api-docker
build-api-docker:
	docker build --build-arg goversion=$(GO_VERSION) --target api -t tharsis/api .

.PHONY: build-job-docker
build-job-docker:
	docker build --build-arg goversion=$(GO_VERSION) --target job-executor -t tharsis/job-executor .

.PHONY: build-runner-docker
build-runner-docker:
	docker build --build-arg goversion=$(GO_VERSION) --target runner -t tharsis/runner .

.PHONY: run-api-docker
run-api-docker:
	docker run -it -p 8000:8000 -p 9090:9090 -v ~/.aws:/root/.aws --env-file .env.docker tharsis/api

.PHONY: db-start
db-start: ## start the database server
	@mkdir -p testdata/postgres
	docker run --rm --name postgres -v $(shell pwd)/testdata:/testdata \
		-v $(shell pwd)/testdata/postgres:/var/lib/postgresql/data \
		-e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=tharsis -d -p 5432:5432 postgres

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
