.PHONY: build build-release build-docker clean test test-ci install

# Build flags to exclude BTRFS and GPG support (avoids requiring btrfs-progs-devel and gpgme)
BUILD_TAGS := exclude_graphdriver_btrfs,containers_image_openpgp
BINARY_NAME := images-archiver
DOCKER_IMAGE ?= images-archiver
LDFLAGS := 
CGO_ENABLED ?= 0

build:
	CGO_ENABLED=$(CGO_ENABLED) go build -v -tags $(BUILD_TAGS) $(if $(LDFLAGS),-ldflags "$(LDFLAGS)") -o $(BINARY_NAME) .

# Build optimized release binary (smaller size, no debug symbols)
build-release:
	CGO_ENABLED=$(CGO_ENABLED) go build -v -tags $(BUILD_TAGS) -ldflags="-s -w" -o $(BINARY_NAME) .

# Build Docker image using pre-built binary
build-docker: build-release
	docker build -t $(DOCKER_IMAGE) .

clean:
	rm -f $(BINARY_NAME) $(BINARY_NAME)-*
	rm -rf tmp-* coverage.txt docker-context

test:
	go test -v -tags $(BUILD_TAGS) ./...

# Run tests with coverage and race detection (for CI)
test-ci:
	go test -v -race -tags "$(BUILD_TAGS)" -coverprofile=coverage.txt -covermode=atomic ./...

install:
	go install -tags $(BUILD_TAGS) .

