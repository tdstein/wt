# Makefile for wt - Git Worktree Setup Tool
#
# Standard targets:
#   make install    - Install wt to system (default: /usr/local)
#   make uninstall  - Remove wt from system
#   make test       - Run test suite
#   make clean      - Clean temporary files
#
# Variables:
#   PREFIX          - Installation prefix (default: /usr/local)
#   DESTDIR         - Staging directory for package builds

# Installation directories
PREFIX ?= /usr/local
DESTDIR ?=
bindir = $(PREFIX)/bin
libdir = $(PREFIX)/lib
sharedir = $(PREFIX)/share

# Project files
BIN_SCRIPT = bin/wt
LIB_FILES = lib/parse.sh lib/repo.sh lib/agent.sh lib/metadata.sh lib/conflict.sh
DOC_FILES = README.md docs/parallel-workflows.md

# Phony targets (not actual files)
.PHONY: all install uninstall test clean help

# Default target
all:
	@echo "wt is a shell script tool - no compilation needed"
	@echo "Run 'make install' to install, or 'make help' for more options"

# Display help
help:
	@echo "wt Makefile targets:"
	@echo "  make install          Install wt (default: $(PREFIX))"
	@echo "  make uninstall        Uninstall wt"
	@echo "  make test             Run test suite"
	@echo "  make clean            Clean temporary files"
	@echo "  make help             Show this help"
	@echo ""
	@echo "Variables:"
	@echo "  PREFIX=$(PREFIX)      Installation prefix"
	@echo "  DESTDIR=$(DESTDIR)    Staging directory"
	@echo ""
	@echo "Examples:"
	@echo "  make install                    # Install to /usr/local"
	@echo "  make install PREFIX=/opt/local  # Install to /opt/local"
	@echo "  sudo make install               # System-wide install"
	@echo "  make install PREFIX=~/.local    # User install"

# Install target
install:
	@echo "Installing wt to $(DESTDIR)$(PREFIX)..."
	@# Create directories
	mkdir -p $(DESTDIR)$(bindir)
	mkdir -p $(DESTDIR)$(libdir)/wt
	mkdir -p $(DESTDIR)$(sharedir)/doc/wt
	@# Install executable with execute permissions
	install -m 755 $(BIN_SCRIPT) $(DESTDIR)$(bindir)/wt
	@# Install library files with read permissions
	install -m 644 $(LIB_FILES) $(DESTDIR)$(libdir)/wt/
	@# Install documentation
	install -m 644 $(DOC_FILES) $(DESTDIR)$(sharedir)/doc/wt/
	@echo "Installation complete!"
	@echo ""
	@echo "wt is now installed to: $(DESTDIR)$(bindir)/wt"
	@echo "Library files:          $(DESTDIR)$(libdir)/wt/"
	@echo "Documentation:          $(DESTDIR)$(sharedir)/doc/wt/"
	@echo ""
	@echo "Make sure $(bindir) is in your PATH"

# Uninstall target
uninstall:
	@echo "Uninstalling wt from $(DESTDIR)$(PREFIX)..."
	@# Remove executable
	rm -f $(DESTDIR)$(bindir)/wt
	@# Remove library directory
	rm -rf $(DESTDIR)$(libdir)/wt
	@# Remove documentation directory
	rm -rf $(DESTDIR)$(sharedir)/doc/wt
	@echo "Uninstallation complete!"

# Run tests
test:
	@echo "Running wt test suite..."
	./tests/test-runner.sh

# Clean temporary files
clean:
	@echo "Cleaning temporary files..."
	@# Remove common temporary files
	find . -name '*~' -delete
	find . -name '*.bak' -delete
	find . -name '.DS_Store' -delete
	@echo "Clean complete!"
