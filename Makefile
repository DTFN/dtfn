GOTOOLS := \
					 github.com/karalabe/xgo \
					 github.com/alecthomas/gometalinter

# PACKAGES := $(shell glide novendor)

BUILD_TAGS? := dtfn

VERSION_TAG := 1.0.0-beta

### Development ###
all: vendor_deps install test
dev: vendor_deps build
dtfn_ubuntu: bls_ubuntu develop_ubuntu develop_build

develop_bls:
	@echo "create bls environment"
	@bash ./scripts/build.sh -t blsdep

vendor_deps:
	@echo "build dtfn"
	@bash ./scripts/build.sh -t mod

install:
	@bash ./scripts/build.sh -t install

build:
	@bash ./scripts/build.sh -t build

test:
	@echo "--> Running go test"
	# @go test $(PACKAGES)

clean:
	@bash ./scripts/build.sh -t clean

bls_ubuntu:
	@echo "create bls environment"
	@bash ./scripts/install_bls_ubuntu.sh

develop_ubuntu:
	@echo "create develop_environment"
	@bash ./scripts/develop_env_ubuntu.sh

bls_centos:
	@echo "create bls environment"
	@bash ./scripts/install_bls_centos.sh

develop_centos:
	@echo "create develop_enviorment"
	@bash ./scripts/develop_env_centos.sh

develop_build:
	@bash $(GOPATH)/src/github.com/DTFN/dtfn/cmd/dtfn/rebuild.sh
	@bash $(GOPATH)/src/github.com/tendermint/tendermint/cmd/tendermint/rebuild.sh

### Tooling ###
ci:
	@chmod +x ./tests/ci.sh
	@./tests/ci.sh