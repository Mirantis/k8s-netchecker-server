# Copyright 2017 Mirantis
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


IMAGE_REPO_SERVER ?= mirantis/k8s-netchecker-server
IMAGE_REPO_AGENT ?= mirantis/k8s-netchecker-agent
HELM_SERVER_PATH ?= helm-chart/netchecker-server
HELM_AGENT_PATH ?= helm-chart/netchecker-agent
HELM_SCRIPT_NAME ?= get_helm.sh
# repo for biuld agent docker image
NETCHECKER_REPO ?= k8s-netchecker-agent
DOCKER_BUILD ?= no

BUILD_DIR = _output
VENDOR_DIR = vendor
ROOT_DIR = $(abspath $(dir $(lastword $(MAKEFILE_LIST))))

# kubeadm-dind-cluster supports k8s versions:
# "v1.4", "v1.5" and "v1.6".
DIND_CLUSTER_VERSION ?= v1.5

ENV_PREPARE_MARKER = .env-prepare.complete
BUILD_IMAGE_MARKER = .build-image.complete


ifeq ($(DOCKER_BUILD), yes)
	_DOCKER_GOPATH = /go
	_DOCKER_WORKDIR = $(_DOCKER_GOPATH)/src/github.com/Mirantis/k8s-netchecker-server/
	_DOCKER_IMAGE  = golang:1.7
	DOCKER_DEPS = apt-get update; apt-get install -y libpcap-dev;
	DOCKER_EXEC = docker run --rm -it -v "$(ROOT_DIR):$(_DOCKER_WORKDIR)" \
		-w "$(_DOCKER_WORKDIR)" $(_DOCKER_IMAGE)
else
	DOCKER_EXEC =
	DOCKER_DEPS =
endif


.PHONY: help
help:
	@echo "For containerized "make get-deps""
	@echo "and "make test" export DOCKER_BUILD=yes"
	@echo ""
	@echo "Usage: 'make <target>'"
	@echo ""
	@echo "Targets:"
	@echo "help                - Print this message and exit"
	@echo "get-deps            - Install project dependencies"
	@echo "build               - Build k8s-netchecker-server binary"
	@echo "containerized-build - Build k8s-netchecker-server binary in container"
	@echo "build-image         - Build docker image"
	@echo "test                - Run all tests"
	@echo "unit                - Run unit tests"
	@echo "e2e                 - Run e2e tests"
	@echo "docker-publish      - Push images to Docker Hub registry"
	@echo "clean               - Delete binaries"
	@echo "clean-k8s           - Delete kubeadm-dind-cluster"
	@echo "clean-all           - Delete binaries and vendor files"


.PHONY: get-deps
get-deps: $(VENDOR_DIR)


.PHONY: build
build: $(BUILD_DIR)/server


.PHONY: containerized-build
containerized-build:
	make build DOCKER_BUILD=yes


.PHONY: build-image
build-image: $(BUILD_IMAGE_MARKER)


.PHONY: unit
unit:
	$(DOCKER_EXEC) go test -v ./pkg/...


.PHONY: e2e
e2e: $(BUILD_DIR)/e2e.test $(ENV_PREPARE_MARKER)
	echo "TODO: sudo $(BUILD_DIR)/e2e.test"


.PHONY: test
test: unit e2e


.PHONY: docker-publish
docker-publish:
	IMAGE_REPO=$(IMAGE_REPO_SERVER) bash ./scripts/docker_publish.sh


.PHONY: clean
clean:
	rm -rf $(BUILD_DIR)


.PHONY: clean-k8s
clean-k8s:
	rm -f ./scripts/$(HELM_SCRIPT_NAME)
	rm -rf $(HOME)/.helm
	bash ./scripts/dind-cluster-$(DIND_CLUSTER_VERSION).sh clean
	rm -f ./scripts/dind-cluster-$(DIND_CLUSTER_VERSION).sh
	rm -rf $(HOME)/.kubeadm-dind-cluster
	rm -rf $(HOME)/.kube
	rm -f $(ENV_PREPARE_MARKER)


.PHONY: clean-all
clean-all: clean clean-k8s
	rm -rf $(VENDOR_DIR)
	docker rmi -f $(IMAGE_REPO_SERVER)
	docker rmi -f $(IMAGE_REPO_AGENT)
	rm -f $(BUILD_IMAGE_MARKER)


$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)


$(VENDOR_DIR):
	$(DOCKER_EXEC) bash -xc 'go get github.com/Masterminds/glide && \
		glide install --strip-vendor; \
		chown $(shell id -u):$(shell id -g) -R $(VENDOR_DIR)'


$(BUILD_DIR)/server: $(BUILD_DIR) $(VENDOR_DIR)
	$(DOCKER_EXEC) bash -xc '$(DOCKER_DEPS) \
		CGO_ENABLED=0 go build --ldflags "-s -w" \
		-x -o $@ ./cmd/server.go; \
		chown $(shell id -u):$(shell id -g) -R $(BUILD_DIR)'


$(BUILD_DIR)/e2e.test: $(BUILD_DIR) $(VENDOR_DIR)
	$(DOCKER_EXEC) echo "TODO: go test -c -o $@ ./test/e2e/"


$(BUILD_IMAGE_MARKER): $(BUILD_DIR)/server
	docker build -t $(IMAGE_REPO_SERVER) .
	touch $(BUILD_IMAGE_MARKER)


$(ENV_PREPARE_MARKER): build-image
	NETCHECKER_REPO=$(NETCHECKER_REPO) bash ./scripts/build_image_server_or_agent.sh
	bash ./scripts/kubeadm_dind_cluster.sh
	IMAGE_REPO_SERVER=$(IMAGE_REPO_SERVER) IMAGE_REPO_AGENT=$(IMAGE_REPO_AGENT) bash ./scripts/import_images.sh
	NETCHECKER_REPO=$(NETCHECKER_REPO) bash ./scripts/helm_install_and_deploy.sh
	touch $(ENV_PREPARE_MARKER)
