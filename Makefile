.PHONY: all build build-linux build-windows run clean install update-deps

BINARY_NAME = omnilyrics-bridge
MAIN_FILE = main.go

GO = go
GOOS ?= $(shell $(GO) env GOHOSTOS)
GOARCH ?= $(shell $(GO) env GOHOSTARCH)

all: build

build:
	@echo "Building for $(GOOS)/$(GOARCH)..."
ifneq ($(GOOS),windows)
	@$(GO) build -o $(BINARY_NAME) .
else
	@$(GO) build -tags windows -o $(BINARY_NAME).exe .
endif

build-linux:
	@echo "Building for Linux..."
	@GOOS=linux GOARCH=amd64 $(GO) build -o $(BINARY_NAME)-linux-amd64 $(MAIN_FILE)

build-windows:
	@echo "Building for Windows..."
	@GOOS=windows GOARCH=amd64 CGO_ENABLED=1 $(GO) build -tags windows -o $(BINARY_NAME).exe $(MAIN_FILE)

run: build
	@./$(BINARY_NAME)

clean:
ifneq ($(GOOS),windows)
	@rm -f $(BINARY_NAME)
else
	@rm -f $(BINARY_NAME).exe
endif
	@rm -f $(BINARY_NAME)-linux-amd64
	@rm -f $(BINARY_NAME)-*.exe

install:
	$(GO) install

update-deps:
	$(GO) mod download
ifeq ($(GOOS),windows)
	$(GO) mod vendor
endif