BUILD_DIR=_output
BIN_NAME=server

$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

.PHONY: build-local
build-local: $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BIN_NAME) ./...

.PHONY: rebuild-local
rebuild-local: clean build-local

.PHONY: get-deps
get-deps:
	go get github.com/golang/glog
	go get github.com/julienschmidt/httprouter

.PHONY: clean
clean:
	rm -rf $(BUILD_DIR)
