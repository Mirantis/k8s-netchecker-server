BUILD_DIR=_output
BIN_NAME=server
BUILD_IMAGE_NAME=k8s-netchecker-server.build
DEPLOY_IMAGE_NAME=aateem/k8s-netchecker-server
DEPLOY_IMAGE_TAG=golang

$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

.PHONY: build-local
build-local: $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BIN_NAME) $(glide novendor)

.PHONY: rebuild-local
rebuild-local: clean build-local

.PHONY: get-deps
get-deps:
	glide install

.PHONY: test-local
test-local:
	go test -v $(glide novendor)

.PHONY: clean
clean:
	rm -rf $(BUILD_DIR)

.PHONY: clean-all
clean-all: clean
	docker rmi $(BUILD_IMAGE_NAME)
	docker rmi $(DEPLOY_IMAGE_NAME):$(DEPLOY_IMAGE_TAG)

prepare-build-container: Dockerfile.build
	docker build -f Dockerfile.build -t $(BUILD_IMAGE_NAME) .

build-containerized:  $(BUILD_DIR) prepare-build-container
	docker run --rm  \
		-v $(PWD):/go/src/github.com/Mirantis/k8s-netchecker-server:ro \
		-v $(PWD)/$(BUILD_DIR):/go/src/github.com/Mirantis/k8s-netchecker-server/$(BUILD_DIR) \
		-w /go/src/github.com/Mirantis/k8s-netchecker-server/ \
		$(BUILD_IMAGE_NAME) bash -c '\
			CGO_ENABLED=0 go build -x -o $(BUILD_DIR)/$(BIN_NAME) -ldflags "-s -w" $(glide novendor) &&\
			chown -R $(shell id -u):$(shell id -u) $(BUILD_DIR)'

prepare-deploy-container: build-containerized
	docker build -t $(DEPLOY_IMAGE_NAME):$(DEPLOY_IMAGE_TAG) .

test-containerized: prepare-build-container
	docker run --rm \
		-v $(PWD):/go/src/github.com/Mirantis/k8s-netchecker-server:ro \
		$(BUILD_IMAGE_NAME) go test -v $(glide novendor)
