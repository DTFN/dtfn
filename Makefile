PACKAGES := $(shell glide novendor)

BUILD_TAGS? := gelchain

### Development ###
all: glide_vendor_deps install test
dev: glide_vendor_deps build

develop_bls:
	@echo "create bls environment"
	@bash ./scripts/build.sh -t blsdep

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
