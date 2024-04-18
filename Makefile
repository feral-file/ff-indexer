# SPDX-License-Identifier: ISC
# Copyright (c) 2019-2024 Bitmark Inc.
# Use of this source code is governed by an ISC
# license that can be found in the LICENSE file.

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

BUILD_LIST = nft-indexer
.PHONY: nft-indexer
nft-indexer:
	go build -o bin/nft-indexer ./services/nft-indexer

BUILD_LIST += nft-indexer-background
.PHONY: nft-indexer-background
nft-indexer-background:
	go build -o bin/nft-indexer-background ./services/nft-indexer-background

BUILD_LIST += nft-indexer-grpc
.PHONY: nft-indexer-grpc
nft-indexer-grpc:
	go build -o bin/nft-indexer-grpc ./services/nft-indexer-grpc

BUILD_LIST += nft-image-indexer
.PHONY: nft-image-indexer
nft-image-indexer:
	go build -o bin/nft-image-indexer ./services/nft-image-indexer

BUILD_LIST += nft-event-processor
.PHONY: nft-event-processor
nft-event-processor:
	go build -o bin/nft-event-processor ./services/nft-event-processor

BUILD_LIST += nft-provenance-indexer
.PHONY: nft-provenance-indexer
nft-provenance-indexer:
	go build -o bin/nft-provenance-indexer ./services/nft-provenance-indexer

BUILD_LIST += nft-account-token-indexer
.PHONY: nft-account-token-indexer
nft-account-token-indexer:
	go build -o bin/nft-account-token-indexer ./services/nft-account-token-indexer

BUILD_LIST += nft-ethereum-emitter
.PHONY: nft-ethereum-emitter
nft-ethereum-emitter:
	go build -o bin/nft-ethereum-emitter ./services/nft-event-processor-ethereum-emitter

BUILD_LIST += nft-tezos-emitter
.PHONY: nft-tezos-emitter
nft-tezos-emitter:
	go build -o bin/nft-tezos-emitter ./services/nft-event-processor-tezos-emitter

# run

.PHONY: run-nft-indexer
run-nft-indexer: nft-indexer
	./bin/nft-indexer -c config.yaml

.PHONY: run-nft-indexer-background
run-nft-indexer-background: nft-indexer-background
	./bin/nft-indexer-background -c config.yaml

.PHONY: run-nft-indexer-grpc
run-nft-indexer-grpc: nft-indexer-grpc
	./bin/nft-indexer-grpc -c config.yaml

.PHONY: run-nft-image-indexer
run-nft-image-indexer: nft-image-indexer
	./bin/nft-image-indexer -c config.yaml

.PHONY: run-nft-event-processor
run-nft-event-processor: nft-event-processor
	./bin/nft-event-processor -c config.yaml

.PHONY: run-nft-provenance-indexer
run-nft-provenance-indexer: nft-provenance-indexer
	./bin/nft-provenance-indexer -c config.yaml

.PHONY: run-nft-account-token-indexer
run-nft-account-token-indexer: nft-account-token-indexer
	./bin/nft-account-token-indexer -c config.yaml

.PHONY: run-nft-ethereum-emitter
run-nft-ethereum-emitter: nft-ethereum-emitter
	./bin/nft-ethereum-emitter -c config.yaml

.PHONY: run-nft-tezos-emitter
run-nft-tezos-emitter: nft-tezos-emitter
	./bin/nft-tezos-emitter -c config.yaml

.PHONY: run
run: config run-nft-indexer

# rebuild items

.PHONY: renew-event-processor-grpc
renew-event-processor-grpc:
	protoc --proto_path=protos --go-grpc_out=services/nft-event-processor/ --go_out=services/nft-event-processor/ event-processor.proto

.PHONY: generate-nft-indexer-graphql
generate-nft-indexer-graphql:
	${MAKE} -C services/nft-indexer/graph/ all

.PHONY: build-rebuild
build-rebuild: generate-nft-indexer-graphql renew-event-processor-grpc build

#BL=nft-indexer nft-indexer-background nft-event-processor nft-provenance-indexer nft-account-token-indexer nft-ethereum-emitter nft-tezos-emitter
.PHONY: build
build: ${BUILD_LIST}

.PHONY: build-nft-indexer
build-nft-indexer:
ifndef dist
	$(error dist is undefined)
endif
	$(DOCKER_BUILD_COMMAND) --build-arg dist=$(dist) \
	--build-arg GITHUB_USER=$(GITHUB_USER) \
	--build-arg GITHUB_TOKEN=$(GITHUB_TOKEN) \
	-t nft-indexer:api-$(dist) .
	docker tag nft-indexer:api-$(dist) 083397868157.dkr.ecr.ap-northeast-1.amazonaws.com/nft-indexer:api-$(dist)

.PHONY: build-nft-indexer-background
build-nft-indexer-background:
ifndef dist
	$(error dist is undefined)
endif
	$(DOCKER_BUILD_COMMAND) --build-arg dist=$(dist) \
	--build-arg GITHUB_USER=$(GITHUB_USER) \
	--build-arg GITHUB_TOKEN=$(GITHUB_TOKEN) \
	-t nft-indexer:background-$(dist) -f Dockerfile-background .
	docker tag nft-indexer:background-$(dist) 083397868157.dkr.ecr.ap-northeast-1.amazonaws.com/nft-indexer:background-$(dist)

.PHONY: build-nft-indexer-grpc-image
build-nft-indexer-grpc-image:
ifndef dist
	$(error 'dist is undefined')
endif
	$(DOCKER_BUILD_COMMAND) --build-arg dist=$(dist) \
	--build-arg GITHUB_USER=$(GITHUB_USER) \
	--build-arg GITHUB_TOKEN=$(GITHUB_TOKEN) \
	-t nft-indexer:grpc-'${dist}' -f Dockerfile-grpc .
	docker tag nft-indexer:grpc-'${dist}' 083397868157.dkr.ecr.ap-northeast-1.amazonaws.com/nft-indexer:grpc-'${dist}'

.PHONY: build-nft-provenance-indexer
build-nft-provenance-indexer:
ifndef dist
	$(error dist is undefined)
endif
	$(DOCKER_BUILD_COMMAND) --build-arg dist=$(dist) \
	--build-arg GITHUB_USER=$(GITHUB_USER) \
	--build-arg GITHUB_TOKEN=$(GITHUB_TOKEN) \
	-t nft-indexer:provenance-indexer-$(dist) -f Dockerfile-provenance-indexer .
	docker tag nft-indexer:provenance-indexer-$(dist) 083397868157.dkr.ecr.ap-northeast-1.amazonaws.com/nft-indexer:provenance-indexer-$(dist)

.PHONY: build-nft-account-token-indexer
build-nft-account-token-indexer:
ifndef dist
	$(error dist is undefined)
endif
	$(DOCKER_BUILD_COMMAND) --build-arg dist=$(dist) \
	--build-arg GITHUB_USER=$(GITHUB_USER) \
	--build-arg GITHUB_TOKEN=$(GITHUB_TOKEN) \
	-t nft-indexer:account-token-indexer-$(dist) -f Dockerfile-account-token-indexer .
	docker tag nft-indexer:account-token-indexer-$(dist) 083397868157.dkr.ecr.ap-northeast-1.amazonaws.com/nft-indexer:account-token-indexer-$(dist)

.PHONY: build-nft-ethereum-emitter
build-nft-ethereum-emitter:
ifndef dist
	$(error dist is undefined)
endif
	$(DOCKER_BUILD_COMMAND) --build-arg dist=$(dist) \
	--build-arg GITHUB_USER=$(GITHUB_USER) \
	--build-arg GITHUB_TOKEN=$(GITHUB_TOKEN) \
	-t nft-indexer:ethereum-emitter-$(dist) -f Dockerfile-ethereum-emitter .
	docker tag nft-indexer:ethereum-emitter-$(dist) 083397868157.dkr.ecr.ap-northeast-1.amazonaws.com/nft-indexer:ethereum-emitter-$(dist)

.PHONY: build-nft-tezos-emitter
build-nft-tezos-emitter:
ifndef dist
	$(error dist is undefined)
endif
	$(DOCKER_BUILD_COMMAND) --build-arg dist=$(dist) \
	--build-arg GITHUB_USER=$(GITHUB_USER) \
	--build-arg GITHUB_TOKEN=$(GITHUB_TOKEN) \
	-t nft-indexer:tezos-emitter-$(dist) -f Dockerfile-tezos-emitter .
	docker tag nft-indexer:tezos-emitter-$(dist) 083397868157.dkr.ecr.ap-northeast-1.amazonaws.com/nft-indexer:tezos-emitter-$(dist)

.PHONY: build-nft-event-processor
build-nft-event-processor:
ifndef dist
	$(error dist is undefined)
endif
	$(DOCKER_BUILD_COMMAND) --build-arg dist=$(dist) \
	--build-arg GITHUB_USER=$(GITHUB_USER) \
	--build-arg GITHUB_TOKEN=$(GITHUB_TOKEN) \
	-t nft-indexer:event-processor-$(dist) -f Dockerfile-event-processor .
	docker tag nft-indexer:event-processor-$(dist) 083397868157.dkr.ecr.ap-northeast-1.amazonaws.com/nft-indexer:event-processor-$(dist)

.PHONY: build-nft-image-indexer
build-nft-image-indexer:
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
image: build-nft-indexer build-nft-indexer-background

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
