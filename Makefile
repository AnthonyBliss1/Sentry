-include .env

HUB_DIR := Hub
NODE_DIR := Node

HUB_BUILD_DIR := build/Hub
NODE_BUILD_DIR := build/Node

HUB_BINARY := sentry-hub
NODE_BINARY := sentry-node

HUB_DEVICE := $(HUB_DEVICE)
NODE_DEVICE := $(NODE_DEVICE)

.PHONY: all hub node clean dirs deploy-hub deploy-node

all: hub node

dirs:
	@mkdir -p $(HUB_BUILD_DIR) $(NODE_BUILD_DIR)

hub: dirs
	cd $(HUB_DIR) && GOOS=darwin GOARCH=arm64 go build -o ../$(HUB_BUILD_DIR)/$(HUB_BINARY)
	scp $(HUB_BUILD_DIR)/$(HUB_BINARY) $(HUB_DEVICE):~

node: dirs
	cd $(NODE_DIR) && GOOS=linux GOARCH=arm GOARM=7 go build -o ../$(NODE_BUILD_DIR)/$(NODE_BINARY)
	scp $(NODE_BUILD_DIR)/$(NODE_BINARY) $(NODE_DEVICE):~

clean:
	rm -rf build
