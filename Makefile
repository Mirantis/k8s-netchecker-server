BUILD_DIR=_output
BIN_NAME=server
UTILITY_IMAGE_NAME=k8s-netchecker-server.build
RELEASE_IMAGE_NAME?=quay.io/l23network/k8s-netchecker-server
RELEASE_IMAGE_TAG?=latest
TARGET_IMAGE=$(RELEASE_IMAGE_NAME):$(RELEASE_IMAGE_TAG)

$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

.PHONY: build-local
build-local: $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BIN_NAME) $(glide novendor)

.PHONY: rebuild-local
rebuild-local: clean build-local

.PHONY: get-deps
get-deps:
	glide install --strip-vendor

.PHONY: test-local
test-local:
	go test -v $(glide novendor)

.PHONY: clean
clean:
	rm -rf $(BUILD_DIR)

.PHONY: clean-all
clean-all: clean
	docker rmi $(UTILITY_IMAGE_NAME)
	docker rmi $(TARGET_IMAGE)

build-utility-image: Dockerfile.build
	docker build -f Dockerfile.build -t $(UTILITY_IMAGE_NAME) .

go-build-containerized:  $(BUILD_DIR) build-utility-image
	docker run --rm  \
		-v $(PWD):/go/src/github.com/Mirantis/k8s-netchecker-server:ro \
		-v $(PWD)/$(BUILD_DIR):/go/src/github.com/Mirantis/k8s-netchecker-server/$(BUILD_DIR) \
		-w /go/src/github.com/Mirantis/k8s-netchecker-server/ \
		$(UTILITY_IMAGE_NAME) bash -c '\
			CGO_ENABLED=0 go build -x -o $(BUILD_DIR)/$(BIN_NAME) -ldflags "-s -w" $(glide novendor) &&\
			chown -R $(shell id -u):$(shell id -u) $(BUILD_DIR)'

build-release-image: go-build-containerized
	docker build -t $(TARGET_IMAGE) .

test-containerized: build-utility-image
	docker run --rm \
		-v $(PWD):/go/src/github.com/Mirantis/k8s-netchecker-server:ro \
		$(UTILITY_IMAGE_NAME) go test -v $(glide novendor)
