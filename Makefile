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

default: build

config:
	if [ ! -f "./config.yaml" ]; then \
		cp config.yaml.sample ./config.yaml; \
	fi

nft-indexer:
	go build -o bin/nft-indexer ./services/nft-indexer

nft-indexer-background:
	go build -o bin/nft-indexer-background ./services/nft-indexer-background

run-nft-indexer: nft-indexer
	./bin/nft-indexer -c config.yaml

run-nft-indexer-background: nft-indexer-background
	./bin/nft-indexer-background -c config.yaml

build: nft-indexer

run: config run-nft-indexer

build-nft-indexer:
ifndef dist
	$(error dist is undefined)
endif
	$(DOCKER_BUILD_COMMAND) --build-arg dist=$(dist) \
	--build-arg GITHUB_USER=$(GITHUB_USER) \
	--build-arg GITHUB_TOKEN=$(GITHUB_TOKEN) \
	-t nft-indexer:$(dist) .
	docker tag nft-indexer:$(dist) 083397868157.dkr.ecr.ap-northeast-1.amazonaws.com/nft-indexer:$(dist)

image: build-nft-indexer

push:
ifndef dist
	$(error dist is undefined)
endif
	aws ecr get-login-password | docker login --username AWS --password-stdin 083397868157.dkr.ecr.ap-northeast-1.amazonaws.com
	docker push 083397868157.dkr.ecr.ap-northeast-1.amazonaws.com/nft-indexer:$(dist)

test:
	go test ./...

clean:
	rm -rf bin
