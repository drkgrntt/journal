# Simple Makefile for a Go project

# The exact templ CLI version this module's generated code is compatible
# with - kept in sync with go.mod automatically (not hardcoded a second
# time) so bumping the dependency there is the only place that needs to
# change. Mirrors the Dockerfile's own exact-version pin.
TEMPL_VERSION := $(shell awk '/github.com\/a-h\/templ /{print $$2}' go.mod)

# Build the application
all: build test
templ-install:
	@if ! command -v templ > /dev/null; then \
		read -p "Go's 'templ' is not installed on your machine. Do you want to install it? [Y/n] " choice; \
		if [ "$$choice" = "n" ] || [ "$$choice" = "N" ]; then \
			echo "You chose not to install templ. Exiting..."; \
			exit 1; \
		fi; \
	fi
	@if [ "$$(templ version 2>/dev/null)" != "$(TEMPL_VERSION)" ]; then \
		echo "Installed templ CLI doesn't match go.mod's github.com/a-h/templ ($(TEMPL_VERSION)) - a version-mismatched generator produces code the pinned runtime can't compile. Installing $(TEMPL_VERSION)..."; \
		go install github.com/a-h/templ/cmd/templ@$(TEMPL_VERSION); \
	fi
	@if [ ! -x "$$(command -v templ)" ]; then \
		echo "templ installation failed. Exiting..."; \
		exit 1; \
	fi

build: templ-install
	@echo "Building..."
	@templ generate
	
	@go build -o main cmd/api/main.go

# Run the application
run:
	@go run cmd/api/main.go
# Create DB container
docker-run:
	@if docker compose -f docker-compose.dev.yml up --build 2>/dev/null; then \
		: ; \
	else \
		echo "Falling back to Docker Compose V1"; \
		docker-compose -f docker-compose.dev.yml up --build; \
	fi

# Shutdown DB container
docker-down:
	@if docker compose -f docker-compose.dev.yml down 2>/dev/null; then \
		: ; \
	else \
		echo "Falling back to Docker Compose V1"; \
		docker-compose -f docker-compose.dev.yml down; \
	fi

# Test the application
test:
	@echo "Testing..."
	@go test ./... -v
# Integrations Tests for the application
itest:
	@echo "Running integration tests..."
	@go test ./internal/database -v

# Clean the binary
clean:
	@echo "Cleaning..."
	@rm -f main

# Live Reload
watch:
	@if command -v air > /dev/null; then \
            air; \
            echo "Watching...";\
        else \
            read -p "Go's 'air' is not installed on your machine. Do you want to install it? [Y/n] " choice; \
            if [ "$$choice" != "n" ] && [ "$$choice" != "N" ]; then \
                go install github.com/air-verse/air@latest; \
                air; \
                echo "Watching...";\
            else \
                echo "You chose not to install air. Exiting..."; \
                exit 1; \
            fi; \
        fi

.PHONY: all build run test clean watch docker-run docker-down itest templ-install
