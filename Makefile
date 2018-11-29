PACKAGES := $(shell glide novendor)

BUILD_TAGS? := gelchain

### Development ###
all: glide_vendor_deps install test

gelchain_ubuntu: develop_bls develop_ubuntu develop_build

gelchain_pos_ubuntu: glide_vendor_deps_pos build

gelchain_bls_ubuntu_pre: develop_bls develop_ubuntu

gelchain_bls_centos_pre: develop_bls develop_centos

glide_vendor_deps_pos:
	@echo "build gelChain"
	@git checkout gelchain-wenbin
	@curl https://glide.sh/get | sh && glide cc && glide install

develop_bls:
	@echo "create bls environment"
	@bash ./scripts/build.sh -t blsdep

develop_ubuntu:
	@echo "create develop_environment"
	@bash ./scripts/develop_env_ubuntu.sh

develop_pos_ubuntu:
	@echo "create develop_environment"
	@bash ./scripts/develop_pos_ubuntu.sh

develop_centos:
	@echo "create develop_enviorment"
	@bash ./scripts/develop_env_centos.sh

develop_build:
	@bash $(GOPATH)/src/github.com/green-element-chain/gelchain/cmd/gelchain/rebuild.sh
	@bash $(GOPATH)/src/github.com/tendermint/tendermint/cmd/tendermint/rebuild.sh

glide_vendor_deps:
	@echo "build gelChain"
	@bash ./scripts/build.sh -t glide

install:
	@bash ./scripts/build.sh -t install

build:
	@bash ./scripts/build.sh -t build

test:
	@echo "--> Running go test"
	@go test $(PACKAGES)

clean:
	@bash ./scripts/build.sh -t clean


### Tooling ###
ci:
	@chmod +x ./tests/ci.sh
	@./tests/ci.sh
