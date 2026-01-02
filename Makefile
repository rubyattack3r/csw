BINARY_NAME=csw
CMD_PATH=./cmd/csw
DIST_DIR=dist
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

# Platform matrix
PLATFORMS := \
	linux/amd64 \
	linux/arm64 \
	windows/amd64 \
	darwin/amd64 \
	darwin/arm64

.PHONY: all build-all clean help fmt lint

all: build-all

help:
	@echo "CSW Build System"
	@echo ""
	@echo "Usage:"
	@echo "  make build-all      Build for all platforms"
	@echo "  make linux          Build for Linux (amd64/arm64)"
	@echo "  make windows        Build for Windows (amd64)"
	@echo "  make darwin         Build for macOS (amd64/arm64)"
	@echo "  make fmt            Format source code"
	@echo "  make lint           Run static analysis (lint)"
	@echo "  make clean          Remove distribution directory"
	@echo "  make help           Show this help message"

build-all: linux windows darwin

# Build targets for specific OS
linux: linux/amd64 linux/arm64
windows: windows/amd64
darwin: darwin/amd64 darwin/arm64

# Generic build rule
$(PLATFORMS):
	$(eval OS := $(word 1,$(subst /, ,$@)))
	$(eval ARCH := $(word 2,$(subst /, ,$@)))
	$(eval OUTPUT := $(DIST_DIR)/$(BINARY_NAME)-$(OS)-$(ARCH))
	$(eval EXT := $(if $(filter windows,$(OS)),.exe,))
	@echo "Building for $(OS)/$(ARCH)..."
	@mkdir -p $(DIST_DIR)
	@GOOS=$(OS) GOARCH=$(ARCH) CGO_ENABLED=0 go build -trimpath \
		-ldflags "-s -w -X main.Version=$(VERSION)" \
		-o $(OUTPUT)$(EXT) $(CMD_PATH)
	@sha256sum $(OUTPUT)$(EXT) > $(OUTPUT)$(EXT).sha256

clean:
	@echo "Cleaning up..."
	@rm -rf $(DIST_DIR)

fmt:
	@echo "Formatting source code..."
	@go fmt ./...

lint:
	@echo "Running lint..."
	@golangci-lint run ./...
