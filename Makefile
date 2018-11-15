GOTOOLS := \
					 github.com/karalabe/xgo \
					 github.com/alecthomas/gometalinter

PACKAGES := $(shell glide novendor)

BUILD_TAGS? := gelchain

VERSION_TAG := 0.6.0

BUILD_FLAGS = -ldflags "-X github.com/green-element-chain/gelchain/version.GitCommit=`git rev-parse --short HEAD`"


### Development ###
all: glide_vendor_deps install test

gelchain: glide_vendor_deps build

gelchain_pos: glide_vendor_deps_pos build

gelchain_bls_ubuntu_pre: bls_ubuntu develop_ubuntu

gelchain_bls_centos_pre: bls_centos develop_centos

glide_vendor_deps:
	@echo "build gelChain"
	@curl https://glide.sh/get | sh && glide install

glide_vendor_deps_pos:
	@echo "build gelChain"
	@git checkout gelchain-pos
	@curl https://glide.sh/get | sh && glide cc && glide install

bls_ubuntu:
	@echo "create bls environment"
	@bash ./scripts/install_bls_ubuntu.sh

bls_centos:
	@echo "create bls environment"
	@bash ./scripts/install_bls_centos.sh

develop_ubuntu:
	@echo "create develop_environment"
	@bash ./scripts/develop_env_ubuntu.sh

develop_centos:
	@echo "create develop_enviorment"
	@bash ./scripts/develop_env_centos.sh

develop_build:
	@bash $(GOPATH)/src/github.com/green-element-chain/gelchain/cmd/gelchain/rebuild.sh
	@bash $(GOPATH)/src/github.com/tendermint/tendermint/cmd/tendermint/rebuild.sh

install:
	CGO_ENABLED=1 go install $(BUILD_FLAGS) ./cmd/gelchain

build:
	CGO_ENABLED=1 go build $(BUILD_FLAGS) -o ./build/gelchain ./cmd/gelchain

test:
	@echo "--> Running go test"
	@go test $(PACKAGES)



### Tooling ###

ci:
	@chmod +x ./tests/ci.sh
	@./tests/ci.sh
