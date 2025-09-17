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

BUILD_LIST = api-gateway
.PHONY: api-gateway
api-gateway:
	go build -o bin/api-gateway ./services/api-gateway

BUILD_LIST += workflow-runner
.PHONY: workflow-runner
workflow-runner:
	go build -o bin/workflow-runner ./services/workflow-runner

BUILD_LIST += grpc-gateway
.PHONY: grpc-gateway
grpc-gateway:
	go build -o bin/grpc-gateway ./services/grpc-gateway

BUILD_LIST += image-indexer
.PHONY: image-indexer
image-indexer:
	go build -o bin/image-indexer ./services/image-indexer

BUILD_LIST += event-processor
.PHONY: event-processor
event-processor:
	go build -o bin/event-processor ./services/event-processor

BUILD_LIST += provenance-indexer
.PHONY: provenance-indexer
provenance-indexer:
	go build -o bin/provenance-indexer ./services/provenance-indexer

BUILD_LIST += ethereum-event-emitter
.PHONY: ethereum-event-emitter
ethereum-event-emitter:
	go build -o bin/ethereum-event-emitter ./services/ethereum-event-emitter

BUILD_LIST += tezos-event-emitter
.PHONY: tezos-event-emitter
tezos-event-emitter:
	go build -o bin/tezos-event-emitter ./services/tezos-event-emitter

# run

.PHONY: run-api-gateway
run-api-gateway: api-gateway
	./bin/api-gateway -c config.yaml

.PHONY: run-workflow-runner
run-workflow-runner: workflow-runner
	./bin/workflow-runner -c config.yaml

.PHONY: run-grpc-gateway
run-grpc-gateway: grpc-gateway
	./bin/grpc-gateway -c config.yaml

.PHONY: run-image-indexer
run-image-indexer: image-indexer
	./bin/image-indexer -c config.yaml

.PHONY: run-event-processor
run-event-processor: event-processor
	./bin/event-processor -c config.yaml

.PHONY: run-provenance-indexer
run-provenance-indexer: provenance-indexer
	./bin/provenance-indexer -c config.yaml

.PHONY: run-ethereum-event-emitter
run-ethereum-event-emitter: ethereum-event-emitter
	./bin/ethereum-event-emitter -c config.yaml

.PHONY: run-tezos-event-emitter
run-tezos-event-emitter: tezos-event-emitter
	./bin/tezos-event-emitter -c config.yaml

# rebuild items

.PHONY: generate-event-processor-grpc
generate-event-processor-grpc:
	protoc --proto_path=protos --go-grpc_out=services/event-processor/ --go_out=services/event-processor/ event-processor.proto

.PHONY: generate-gateway-grpc
generate-gateway-grpc:
	protoc --proto_path=protos --go-grpc_out=services/grpc-gateway/ --go_out=services/grpc-gateway/ gateway.proto

.PHONY: generate-api-gateway-graphql
generate-api-gateway-graphql:
	${MAKE} -C services/api-gateway/graph/ all

.PHONY: build-rebuild
build-rebuild: generate-api-gateway-graphql generate-event-processor-grpc generate-gateway-grpc build

.PHONY: build
build: ${BUILD_LIST}

.PHONY: build-api-gateway
build-api-gateway:
ifndef dist
	$(error dist is undefined)
endif
	$(DOCKER_BUILD_COMMAND) --build-arg dist=$(dist) \
	--build-arg GITHUB_USER=$(GITHUB_USER) \
	--build-arg GITHUB_TOKEN=$(GITHUB_TOKEN) \
	-t nft-indexer:api-$(dist) -f Dockerfile-api-gateway .
	docker tag nft-indexer:api-$(dist) 083397868157.dkr.ecr.ap-northeast-1.amazonaws.com/nft-indexer:api-$(dist)

.PHONY: build-workflow-runner
build-workflow-runner:
ifndef dist
	$(error dist is undefined)
endif
	$(DOCKER_BUILD_COMMAND) --build-arg dist=$(dist) \
	--build-arg GITHUB_USER=$(GITHUB_USER) \
	--build-arg GITHUB_TOKEN=$(GITHUB_TOKEN) \
	-t nft-indexer:background-$(dist) -f Dockerfile-workflow-runner .
	docker tag nft-indexer:background-$(dist) 083397868157.dkr.ecr.ap-northeast-1.amazonaws.com/nft-indexer:background-$(dist)

.PHONY: build-grpc-gateway
build-grpc-gateway:
ifndef dist
	$(error 'dist is undefined')
endif
	$(DOCKER_BUILD_COMMAND) --build-arg dist=$(dist) \
	--build-arg GITHUB_USER=$(GITHUB_USER) \
	--build-arg GITHUB_TOKEN=$(GITHUB_TOKEN) \
	-t nft-indexer:grpc-'${dist}' -f Dockerfile-grpc-gateway .
	docker tag nft-indexer:grpc-'${dist}' 083397868157.dkr.ecr.ap-northeast-1.amazonaws.com/nft-indexer:grpc-'${dist}'

.PHONY: build-provenance-indexer
build-provenance-indexer:
ifndef dist
	$(error dist is undefined)
endif
	$(DOCKER_BUILD_COMMAND) --build-arg dist=$(dist) \
	--build-arg GITHUB_USER=$(GITHUB_USER) \
	--build-arg GITHUB_TOKEN=$(GITHUB_TOKEN) \
	-t nft-indexer:provenance-indexer-$(dist) -f Dockerfile-provenance-indexer .
	docker tag nft-indexer:provenance-indexer-$(dist) 083397868157.dkr.ecr.ap-northeast-1.amazonaws.com/nft-indexer:provenance-indexer-$(dist)

.PHONY: build-ethereum-event-emitter
build-ethereum-event-emitter:
ifndef dist
	$(error dist is undefined)
endif
	$(DOCKER_BUILD_COMMAND) --build-arg dist=$(dist) \
	--build-arg GITHUB_USER=$(GITHUB_USER) \
	--build-arg GITHUB_TOKEN=$(GITHUB_TOKEN) \
	-t nft-indexer:ethereum-emitter-$(dist) -f Dockerfile-ethereum-event-emitter .
	docker tag nft-indexer:ethereum-emitter-$(dist) 083397868157.dkr.ecr.ap-northeast-1.amazonaws.com/nft-indexer:ethereum-emitter-$(dist)

.PHONY: build-tezos-event-emitter
build-tezos-event-emitter:
ifndef dist
	$(error dist is undefined)
endif
	$(DOCKER_BUILD_COMMAND) --build-arg dist=$(dist) \
	--build-arg GITHUB_USER=$(GITHUB_USER) \
	--build-arg GITHUB_TOKEN=$(GITHUB_TOKEN) \
	-t nft-indexer:tezos-emitter-$(dist) -f Dockerfile-tezos-event-emitter .
	docker tag nft-indexer:tezos-emitter-$(dist) 083397868157.dkr.ecr.ap-northeast-1.amazonaws.com/nft-indexer:tezos-emitter-$(dist)

.PHONY: build-event-processor
build-event-processor:
ifndef dist
	$(error dist is undefined)
endif
	$(DOCKER_BUILD_COMMAND) --build-arg dist=$(dist) \
	--build-arg GITHUB_USER=$(GITHUB_USER) \
	--build-arg GITHUB_TOKEN=$(GITHUB_TOKEN) \
	-t nft-indexer:event-processor-$(dist) -f Dockerfile-event-processor .
	docker tag nft-indexer:event-processor-$(dist) 083397868157.dkr.ecr.ap-northeast-1.amazonaws.com/nft-indexer:event-processor-$(dist)

.PHONY: build-image-indexer
build-image-indexer:
ifndef dist
	$(error dist is undefined)
endif
	$(DOCKER_BUILD_COMMAND) --build-arg dist=$(dist) \
	--build-arg GITHUB_USER=$(GITHUB_USER) \
	--build-arg GITHUB_TOKEN=$(GITHUB_TOKEN) \
	-t nft-indexer:image-indexer-$(dist) -f Dockerfile-image-indexer .
	docker tag nft-indexer:image-indexer-$(dist) 083397868157.dkr.ecr.ap-northeast-1.amazonaws.com/nft-indexer:image-indexer-$(dist)

.PHONY: build-chromep
build-chromep:
ifndef dist
	$(error dist is undefined)
endif
	$(DOCKER_BUILD_COMMAND) --build-arg dist=$(dist) \
	-t nft-indexer:chromep-$(dist) -f Dockerfile-chromep .
	docker tag nft-indexer:chromep-$(dist) 083397868157.dkr.ecr.ap-northeast-1.amazonaws.com/nft-indexer:chromep-$(dist)

.PHONY: image
image: build-api-gateway build-workflow-runner build-grpc-gateway build-provenance-indexer build-ethereum-event-emitter build-tezos-event-processor build-event-processor build-image-indexer build-chromep

.PHONY: push
push:
ifndef dist
	$(error dist is undefined)
endif
	aws ecr get-login-password | docker login --username AWS --password-stdin 083397868157.dkr.ecr.ap-northeast-1.amazonaws.com
	docker push 083397868157.dkr.ecr.ap-northeast-1.amazonaws.com/nft-indexer:$(dist)

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
