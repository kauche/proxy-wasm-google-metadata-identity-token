OS   := $(shell uname | awk '{print tolower($$0)}')
ARCH := $(shell case $$(uname -m) in (x86_64) echo amd64 ;; (aarch64) echo arm64 ;; (*) echo $$(uname -m) ;; esac)

BIN_DIR := ./.bin

TINYGO_VERSION := 0.25.0

TINYGO := $(abspath $(BIN_DIR)/tinygo-$(TINYGO_VERSION))/bin/tinygo

tinygo: $(TINYGO)
$(TINYGO):
	@curl -sSL "https://github.com/tinygo-org/tinygo/releases/download/v$(TINYGO_VERSION)/tinygo$(TINYGO_VERSION).$(OS)-$(ARCH).tar.gz" | tar -C $(BIN_DIR) -xzv tinygo
	@mv $(BIN_DIR)/tinygo $(BIN_DIR)/tinygo-$(TINYGO_VERSION)
