.PHONY:

dist =
GITHUB_USER =
GITHUB_TOKEN =

ARCH = $(shell /usr/bin/uname -m)

DOCKER_BUILD_COMMAND = docker build

ifeq ($(ARCH), arm64)
DOCKER_BUILD_COMMAND = docker buildx build --platform linux/amd64 --load
endif

# needed to allow go list to work
GOPRIVATE = github.com/bitmark-inc/*
export GOPRIVATE


.PHONY: all
all: build

.PHONY: default
default: build

.PHONY: config
config:
	if [ ! -f "./config.yaml" ]; then \
		cp config.yaml.sample ./config.yaml; \
	fi

# build

BUILD_LIST = build-api-gateway
.PHONY: build-api-gateway
build-api-gateway:
	go build -o bin/api-gateway ./services/api-gateway

BUILD_LIST += build-workflow-runner
.PHONY: build-workflow-runner
build-workflow-runner:
	go build -o bin/workflow-runner ./services/workflow-runner

BUILD_LIST += build-grpc-gateway
.PHONY: build-grpc-gateway
build-grpc-gateway:
	go build -o bin/grpc-gateway ./services/grpc-gateway

BUILD_LIST += build-image-indexer
.PHONY: build-image-indexer
build-image-indexer:
	go build -o bin/image-indexer ./services/image-indexer

BUILD_LIST += build-event-processor
.PHONY: build-event-processor
build-event-processor:
	go build -o bin/event-processor ./services/event-processor

BUILD_LIST += build-provenance-indexer
.PHONY: build-provenance-indexer
build-provenance-indexer:
	go build -o bin/provenance-indexer ./services/provenance-indexer

BUILD_LIST += build-ethereum-event-emitter
.PHONY: build-ethereum-event-emitter
build-ethereum-event-emitter:
	go build -o bin/ethereum-event-emitter ./services/ethereum-event-emitter

BUILD_LIST += build-tezos-event-emitter
.PHONY: build-tezos-event-emitter
build-tezos-event-emitter:
	go build -o bin/tezos-event-emitter ./services/tezos-event-emitter

# run
RUN_LIST = run-grpc-gateway
.PHONY: run-grpc-gateway
run-grpc-gateway: grpc-gateway
	./bin/grpc-gateway -c config.yaml

RUN_LIST += run-event-processor
.PHONY: run-event-processor
run-event-processor: event-processor
	./bin/event-processor -c config.yaml

RUN_LIST += run-api-gateway
.PHONY: run-api-gateway
run-api-gateway: build-api-gateway
	./bin/api-gateway -c config.yaml

RUN_LIST += run-ethereum-event-emitter
.PHONY: run-ethereum-event-emitter
run-ethereum-event-emitter: build-ethereum-event-emitter
	./bin/ethereum-event-emitter -c config.yaml

RUN_LIST += run-tezos-event-emitter
.PHONY: run-tezos-event-emitter
run-tezos-event-emitter: build-tezos-event-emitter
	./bin/tezos-event-emitter -c config.yaml

RUN_LIST += run-workflow-runner
.PHONY: run-workflow-runner
run-workflow-runner: build-workflow-runner
	./bin/workflow-runner -c config.yaml

RUN_LIST += run-provenance-indexer
.PHONY: run-provenance-indexer
run-provenance-indexer: build-provenance-indexer
	./bin/provenance-indexer -c config.yaml

RUN_LIST += run-image-indexer
.PHONY: run-image-indexer
run-image-indexer: build-image-indexer
	./bin/image-indexer -c config.yaml

# generate codes
.PHONY: generate-event-processor-grpc
generate-event-processor-grpc:
	protoc --proto_path=protos --go-grpc_out=services/event-processor/ --go_out=services/event-processor/ event-processor.proto

.PHONY: generate-gateway-grpc
generate-gateway-grpc:
	protoc --proto_path=protos --go-grpc_out=services/grpc-gateway/ --go_out=services/grpc-gateway/ gateway.proto

.PHONY: generate-api-gateway-graphql
generate-api-gateway-graphql:
	${MAKE} -C services/api-gateway/graph/ all

.PHONY: rebuild
rebuild: generate-api-gateway-graphql generate-event-processor-grpc generate-gateway-grpc build

.PHONY: build
build: ${BUILD_LIST}

.PHONY: run
run: ${RUN_LIST}

# Build docker images

.PHONY: build-image-api-gateway
build-image-api-gateway:
ifndef dist
	$(error dist is undefined)
endif
	$(DOCKER_BUILD_COMMAND) --build-arg dist=$(dist) \
	--build-arg GITHUB_USER=$(GITHUB_USER) \
	--build-arg GITHUB_TOKEN=$(GITHUB_TOKEN) \
	-t nft-indexer:api-$(dist) -f Dockerfile-api-gateway .

.PHONY: build-image-workflow-runner
build-image-workflow-runner:
ifndef dist
	$(error dist is undefined)
endif
	$(DOCKER_BUILD_COMMAND) --build-arg dist=$(dist) \
	--build-arg GITHUB_USER=$(GITHUB_USER) \
	--build-arg GITHUB_TOKEN=$(GITHUB_TOKEN) \
	-t nft-indexer:background-$(dist) -f Dockerfile-workflow-runner .

.PHONY: build-image-grpc-gateway
build-image-grpc-gateway:
ifndef dist
	$(error 'dist is undefined')
endif
	$(DOCKER_BUILD_COMMAND) --build-arg dist=$(dist) \
	--build-arg GITHUB_USER=$(GITHUB_USER) \
	--build-arg GITHUB_TOKEN=$(GITHUB_TOKEN) \
	-t nft-indexer:grpc-'${dist}' -f Dockerfile-grpc-gateway .

.PHONY: build-image-provenance-indexer
build-image-provenance-indexer:
ifndef dist
	$(error dist is undefined)
endif
	$(DOCKER_BUILD_COMMAND) --build-arg dist=$(dist) \
	--build-arg GITHUB_USER=$(GITHUB_USER) \
	--build-arg GITHUB_TOKEN=$(GITHUB_TOKEN) \
	-t nft-indexer:provenance-indexer-$(dist) -f Dockerfile-provenance-indexer .

.PHONY: build-image-ethereum-event-emitter
build-image-ethereum-event-emitter:
ifndef dist
	$(error dist is undefined)
endif
	$(DOCKER_BUILD_COMMAND) --build-arg dist=$(dist) \
	--build-arg GITHUB_USER=$(GITHUB_USER) \
	--build-arg GITHUB_TOKEN=$(GITHUB_TOKEN) \
	-t nft-indexer:ethereum-emitter-$(dist) -f Dockerfile-ethereum-event-emitter .

.PHONY: build-image-tezos-event-emitter
build-image-tezos-event-emitter:
ifndef dist
	$(error dist is undefined)
endif
	$(DOCKER_BUILD_COMMAND) --build-arg dist=$(dist) \
	--build-arg GITHUB_USER=$(GITHUB_USER) \
	--build-arg GITHUB_TOKEN=$(GITHUB_TOKEN) \
	-t nft-indexer:tezos-emitter-$(dist) -f Dockerfile-tezos-event-emitter .

.PHONY: build-image-event-processor
build-image-event-processor:
ifndef dist
	$(error dist is undefined)
endif
	$(DOCKER_BUILD_COMMAND) --build-arg dist=$(dist) \
	--build-arg GITHUB_USER=$(GITHUB_USER) \
	--build-arg GITHUB_TOKEN=$(GITHUB_TOKEN) \
	-t nft-indexer:event-processor-$(dist) -f Dockerfile-event-processor .

.PHONY: build-image-image-indexer
build-image-image-indexer:
ifndef dist
	$(error dist is undefined)
endif
	$(DOCKER_BUILD_COMMAND) --build-arg dist=$(dist) \
	--build-arg GITHUB_USER=$(GITHUB_USER) \
	--build-arg GITHUB_TOKEN=$(GITHUB_TOKEN) \
	-t nft-indexer:image-indexer-$(dist) -f Dockerfile-image-indexer .

.PHONY: build-image
build-image: build-image-api-gateway build-image-workflow-runner build-image-grpc-gateway build-image-provenance-indexer build-image-ethereum-event-emitter build-image-tezos-event-processor build-image-event-processor build-image-image-indexer

.PHONY: test
test:
	go test ./...

.PHONY: vet
vet:
	go mod tidy
	@-[ X"$$(uname -s)" = X"FreeBSD" ] && set-sockio
	go vet -v ./... 2>&1 | \
	  awk '/^#.*$$/{ printf "\033[31m%s\033[0m\n",$$0 } /^[^#]/{ print $$0 }'

.PHONY: list-updates
list-updates:
	@printf 'scanning for direct dependencies...\n'
	go list -u -f '{{if (and (not (or .Main .Indirect)) .Update)}}{{.Path}}: {{.Version}} -> {{.Update.Version}}{{end}}' -m all

.PHONY: list-updates-include-deps
list-updates-include-deps:
	@printf 'scanning all dependencies...\n'
	go list -u -m all

.PHONY: update-picker
update-picker:
	@printf 'searching dependencies...\n'
	@go list -u -f '{{if (and (not (or .Main .Indirect)) .Update)}}{{.Path}} {{.Version}} {{.Update.Version}}{{end}}' -m all | ( \
	  while read path version update junk   ; \
	  do                                      \
	    printf 'update: %s  from: %s  to: %s' "$${path}" "$${version}" "$${update}"  ; \
	    read -p '   [yes/NO] ? ' < /dev/tty yorn junk ; \
	    case "$${yorn}" in           \
	      ([yY]*)                    \
	        go get -u "$${path}"   ; \
	        ;;                       \
	      (*)                        \
	        printf '...skipped\n'  ; \
	        ;;                       \
	    esac                       ; \
	  done                         ; \
	)
	go mod tidy

.PHONY: update-patch-level
update-patch-level:
	-go get -u=patch -t all
	go mod tidy

.PHONY: update-full
update-full:
	-go get -u -t all
	go mod tidy

.PHONY: clean
clean:
	rm -rf bin

.PHONY: complete-clean
complete-clean: clean
	go clean -cache -fuzzcache -modcache -testcache

# Docker Compose Sequential Build Script
# Builds services in dependency order using docker compose up service_name -d --build

.PHONY: docker-build-ordered
docker-build-ordered:
	@echo "Building services in dependency order..."
	@echo "Step 1: Building foundation services..."
	GITHUB_USER=$(GITHUB_USER) GITHUB_TOKEN=$(GITHUB_TOKEN) docker compose up mongodb -d
	GITHUB_USER=$(GITHUB_USER) GITHUB_TOKEN=$(GITHUB_TOKEN) docker compose up postgres -d
	GITHUB_USER=$(GITHUB_USER) GITHUB_TOKEN=$(GITHUB_TOKEN) docker compose up cadence-postgresql -d
	@echo "Waiting for foundation services to be healthy..."
	@until docker compose ps mongodb --format "{{.Status}}" | grep -q "healthy" && docker compose ps postgres --format "{{.Status}}" | grep -q "healthy" && docker compose ps cadence-postgresql --format "{{.Status}}" | grep -q "healthy"; do \
		echo "Waiting for foundation services..."; \
		sleep 2; \
	done
	@echo "Step 2: Building core services..."
	GITHUB_USER=$(GITHUB_USER) GITHUB_TOKEN=$(GITHUB_TOKEN) docker compose up cadence -d --build
	@echo "Waiting for cadence to be healthy..."
	@until docker compose ps cadence --format "{{.Status}}" | grep -q "healthy"; do \
		echo "Waiting for cadence..."; \
		sleep 2; \
	done
	@echo "Step 3: Building cadence-web and grpc-gateway..."
	GITHUB_USER=$(GITHUB_USER) GITHUB_TOKEN=$(GITHUB_TOKEN) docker compose up cadence-web -d
	GITHUB_USER=$(GITHUB_USER) GITHUB_TOKEN=$(GITHUB_TOKEN) docker compose up grpc-gateway -d --build
	@echo "Waiting for grpc-gateway to be healthy..."
	@until docker compose ps grpc-gateway --format "{{.Status}}" | grep -q "healthy"; do \
		echo "Waiting for grpc-gateway..."; \
		sleep 2; \
	done
	@echo "Step 4: Building event-processor and api-gateway..."
	GITHUB_USER=$(GITHUB_USER) GITHUB_TOKEN=$(GITHUB_TOKEN) docker compose up event-processor -d --build
	GITHUB_USER=$(GITHUB_USER) GITHUB_TOKEN=$(GITHUB_TOKEN) docker compose up api-gateway -d --build
	@echo "Waiting for services to be healthy..."
	@until docker compose ps event-processor --format "{{.Status}}" | grep -q "healthy" && docker compose ps api-gateway --format "{{.Status}}" | grep -q "healthy"; do \
		echo "Waiting for services..."; \
		sleep 2; \
	done
	@echo "Step 5: Building ethereum-event-emitter, tezos-event-emitter, workflow-runner, provenance-indexer, and image-indexer..."
	GITHUB_USER=$(GITHUB_USER) GITHUB_TOKEN=$(GITHUB_TOKEN) docker compose up ethereum-event-emitter -d --build
	GITHUB_USER=$(GITHUB_USER) GITHUB_TOKEN=$(GITHUB_TOKEN) docker compose up tezos-event-emitter -d --build
	GITHUB_USER=$(GITHUB_USER) GITHUB_TOKEN=$(GITHUB_TOKEN) docker compose up workflow-runner -d --build
	GITHUB_USER=$(GITHUB_USER) GITHUB_TOKEN=$(GITHUB_TOKEN) docker compose up provenance-indexer -d --build
	GITHUB_USER=$(GITHUB_USER) GITHUB_TOKEN=$(GITHUB_TOKEN) docker compose up image-indexer -d --build
	@echo "All services built and started in dependency order!"

.PHONY: help
help:
	@$(foreach m, ${MAKEFILE_LIST},                          \
	  printf 'toplevel targets from: %s\n' '${m}'          ; \
	  awk '/^[.]PHONY/{ print "  " $$2 }' '${m}' | sort -u ; \
	)


# Makefile debugging

# use like make print-VARIABLE_NAME
# note "print-" is always lowercase, VARIABLE_NAME is case sensitive
.PHONY: print-%
print-%:
	@printf '%s: %s\n' "$(patsubst print-%,%,${@})"  "${$(patsubst print-%,%,${@})}"
