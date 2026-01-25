# Makefile for wt - Git Worktree Setup Tool
#
# Standard targets:
#   make build      - Build the wt binary
#   make test       - Run test suite
#   make install    - Install wt to system (default: /usr/local)
#   make uninstall  - Remove wt from system
#   make clean      - Clean build artifacts
#
# Variables:
#   PREFIX          - Installation prefix (default: /usr/local)
#   DESTDIR         - Staging directory for package builds

# Installation directories
PREFIX ?= /usr/local
DESTDIR ?=
bindir = $(PREFIX)/bin
sharedir = $(PREFIX)/share

# Build variables
BINARY_NAME = wt
BUILD_DIR = bin
GO = go
GOFLAGS = -trimpath
LDFLAGS = -s -w

# Version info (can be overridden)
VERSION ?= dev
GIT_COMMIT = $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME = $(shell date -u '+%Y-%m-%d_%H:%M:%S')

# Inject version info at build time
LDFLAGS += -X 'main.version=$(VERSION)'
LDFLAGS += -X 'main.gitCommit=$(GIT_COMMIT)'
LDFLAGS += -X 'main.buildTime=$(BUILD_TIME)'

# Phony targets (not actual files)
.PHONY: all build test install uninstall clean help fmt vet lint

# Default target
all: build

# Display help
help:
	@echo "wt Makefile targets:"
	@echo "  make build            Build the wt binary"
	@echo "  make test             Run test suite"
	@echo "  make install          Install wt (default: $(PREFIX))"
	@echo "  make uninstall        Uninstall wt"
	@echo "  make clean            Clean build artifacts"
	@echo "  make fmt              Format Go code"
	@echo "  make vet              Run go vet"
	@echo "  make help             Show this help"
	@echo ""
	@echo "Variables:"
	@echo "  PREFIX=$(PREFIX)      Installation prefix"
	@echo "  DESTDIR=$(DESTDIR)    Staging directory"
	@echo "  VERSION=$(VERSION)    Version string"
	@echo ""
	@echo "Examples:"
	@echo "  make build                       # Build wt"
	@echo "  make install                     # Install to /usr/local"
	@echo "  make install PREFIX=/opt/local   # Install to /opt/local"
	@echo "  sudo make install                # System-wide install"
	@echo "  make install PREFIX=~/.local     # User install"
	@echo "  make build VERSION=1.0.0         # Build with version"

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/wt
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Run tests
test:
	@echo "Running wt test suite..."
	$(GO) test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GO) test -v -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Format Go code
fmt:
	@echo "Formatting Go code..."
	$(GO) fmt ./...

# Run go vet
vet:
	@echo "Running go vet..."
	$(GO) vet ./...

# Install target
install: build
	@echo "Installing wt to $(DESTDIR)$(PREFIX)..."
	@# Create directories
	mkdir -p $(DESTDIR)$(bindir)
	mkdir -p $(DESTDIR)$(sharedir)/doc/wt
	@# Install executable with execute permissions
	install -m 755 $(BUILD_DIR)/$(BINARY_NAME) $(DESTDIR)$(bindir)/$(BINARY_NAME)
	@# Install documentation
	install -m 644 README.md $(DESTDIR)$(sharedir)/doc/wt/
	install -m 644 MIGRATION.md $(DESTDIR)$(sharedir)/doc/wt/
	@[ -d docs ] && install -m 644 docs/*.md $(DESTDIR)$(sharedir)/doc/wt/ || true
	@echo "Installation complete!"
	@echo ""
	@echo "wt is now installed to: $(DESTDIR)$(bindir)/$(BINARY_NAME)"
	@echo "Documentation:          $(DESTDIR)$(sharedir)/doc/wt/"
	@echo ""
	@echo "Make sure $(bindir) is in your PATH"

# Uninstall target
uninstall:
	@echo "Uninstalling wt from $(DESTDIR)$(PREFIX)..."
	@# Remove executable
	rm -f $(DESTDIR)$(bindir)/$(BINARY_NAME)
	@# Remove documentation directory
	rm -rf $(DESTDIR)$(sharedir)/doc/wt
	@echo "Uninstallation complete!"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html
	@# Remove common temporary files
	find . -name '*~' -delete
	find . -name '*.bak' -delete
	find . -name '.DS_Store' -delete
	$(GO) clean
	@echo "Clean complete!"
