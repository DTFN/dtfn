GOTOOLS := \
					 github.com/karalabe/xgo \
					 github.com/alecthomas/gometalinter

PACKAGES := $(shell glide novendor)

BUILD_TAGS? := ethermint

VERSION_TAG := 0.5.3

BUILD_FLAGS = -ldflags "-X github.com/tendermint/ethermint/version.GitCommit=`git rev-parse --short HEAD`"


### Development ###
all: glide_vendor_deps install test

gelchain: glide_vendor_deps build

glide_vendor_deps:
	@echo "build gelChain"
	@curl https://glide.sh/get | sh && glide install

develop:
    @echo "create develop_enviorment"
    @bash ./scripts/develop_env.sh

install:
	CGO_ENABLED=1 go install $(BUILD_FLAGS) ./cmd/ethermint

build:
	CGO_ENABLED=1 go build $(BUILD_FLAGS) -o ./build/gelchain ./cmd/ethermint

test:
	@echo "--> Running go test"
	@go test $(PACKAGES)



### Tooling ###

ci:
	@chmod +x ./tests/ci.sh
	@./tests/ci.sh
