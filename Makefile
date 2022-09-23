# SPDX-License-Identifier: ISC
# Copyright (c) 2019-2021 Bitmark Inc.
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

.PHONY: default
default: build

.PHONY: config
config:
	if [ ! -f "./config.yaml" ]; then \
		cp config.yaml.sample ./config.yaml; \
	fi

.PHONY: nft-indexer
nft-indexer:
	go build -o bin/nft-indexer ./services/nft-indexer

.PHONY: nft-indexer-background
nft-indexer-background:
	go build -o bin/nft-indexer-background ./services/nft-indexer-background

.PHONY: nft-image-indexer
nft-image-indexer:
	go build -o bin/nft-image-indexer ./services/nft-image-indexer

.PHONY: nft-event-subscriber
nft-event-subscriber:
	go build -o bin/nft-event-subscriber ./services/nft-event-subscriber

.PHONY: nft-provenance-indexer
nft-provenance-indexer:
	go build -o bin/nft-provenance-indexer ./services/nft-provenance-indexer

.PHONY: run-nft-indexer
run-nft-indexer: nft-indexer
	./bin/nft-indexer -c config.yaml

.PHONY: run-nft-indexer-background
run-nft-indexer-background: nft-indexer-background
	./bin/nft-indexer-background -c config.yaml

.PHONY: run-nft-image-indexer
run-nft-image-indexer: nft-image-indexer
	./bin/nft-image-indexer -c config.yaml

.PHONY: run-nft-event-subscriber
run-nft-event-subscriber: nft-event-subscriber
	./bin/nft-event-subscriber -c config.yaml

.PHONY: run-nft-provenance-indexer
run-nft-provenance-indexer: nft-provenance-indexer
	./bin/nft-provenance-indexer -c config.yaml

.PHONY: build
build: nft-indexer nft-indexer-background nft-event-subscriber nft-provenance-indexer

.PHONY: run
run: config run-nft-indexer

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

.PHONY: build-nft-provenance-indexer
build-nft-provenance-indexer:
ifndef dist
	$(error dist is undefined)
endif
	$(DOCKER_BUILD_COMMAND) --build-arg dist=$(dist) \
	--build-arg GITHUB_USER=$(GITHUB_USER) \
	--build-arg GITHUB_TOKEN=$(GITHUB_TOKEN) \
	-t nft-provenance-indexer-$(dist) -f Dockerfile-provenance-indexer .
	docker tag nft-provenance-indexer-$(dist) 083397868157.dkr.ecr.ap-northeast-1.amazonaws.com/nft-provenance-indexer-$(dist)


.PHONY: build-nft-event-subscriber
build-nft-event-subscriber:
ifndef dist
	$(error dist is undefined)
endif
	$(DOCKER_BUILD_COMMAND) --build-arg dist=$(dist) \
	--build-arg GITHUB_USER=$(GITHUB_USER) \
	--build-arg GITHUB_TOKEN=$(GITHUB_TOKEN) \
	-t nft-indexer:event-subscriber-$(dist) -f Dockerfile-event-subscriber .
	docker tag nft-indexer:event-subscriber-$(dist) 083397868157.dkr.ecr.ap-northeast-1.amazonaws.com/nft-indexer:event-subscriber-$(dist)

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
	go vet -v ./... 2>&1 | \
	  awk '/^#.*$$/{ printf "\033[31m%s\033[0m\n",$$0 } /^[^#]/{ print $$0 }'

.PHONY: clean
clean:
	rm -rf bin
