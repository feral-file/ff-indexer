# SPDX-License-Identifier: ISC
# Copyright (c) 2019-2021 Bitmark Inc.
# Use of this source code is governed by an ISC
# license that can be found in the LICENSE file.

.PHONY:

ARCH = $(shell /usr/bin/uname -m)

DOCKER_BUILD_COMMAND = docker build

ifeq ($(ARCH), arm64)
DOCKER_BUILD_COMMAND = docker buildx build --platform linux/amd64 --load
endif

default: build

build:
	npm run build

run: build
	./entrypoint.sh

build-image:
ifndef dist
	$(error dist is undefined)
endif
	$(DOCKER_BUILD_COMMAND) -t nft-indexer:tezos-marketplace-indexer-$(dist) . -f Dockerfile-nft-indexer
	docker tag nft-indexer:tezos-marketplace-indexer-$(dist) 083397868157.dkr.ecr.ap-northeast-1.amazonaws.com/nft-indexer:tezos-marketplace-indexer-$(dist)

add-contract:
ifndef name
	$(error name is undefined)
endif
ifndef address
	$(error address is undefined)
endif
	npx dappetizer init --network mainnet --contractName=$(name) $(address)
