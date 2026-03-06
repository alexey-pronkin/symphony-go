SHELL := /bin/bash
ROOT_DIR := $(CURDIR)
LOCAL_CACHE_DIR := $(ROOT_DIR)/.cache
GOLANGCI_CACHE_DIR := $(LOCAL_CACHE_DIR)/golangci-lint
GO_BUILD_CACHE_DIR := $(LOCAL_CACHE_DIR)/go-build
UV_CACHE_DIR := $(LOCAL_CACHE_DIR)/uv

GO := $(shell command -v go 2>/dev/null)
GOFMT := $(shell command -v gofmt 2>/dev/null)
GOLANGCI_LINT := $(shell command -v golangci-lint 2>/dev/null)
RUFF := $(shell command -v ruff 2>/dev/null)
NPM := $(shell command -v npm 2>/dev/null)
UV := $(shell command -v uv 2>/dev/null)
UVX := $(shell command -v uvx 2>/dev/null)
TRIVY := $(shell command -v trivy 2>/dev/null)
DOCKER := $(shell command -v docker 2>/dev/null)
DOCKER_COMPOSE := $(shell command -v docker-compose 2>/dev/null)

.PHONY: hooks-install format format-check lint \
	format-go format-check-go lint-go \
	format-frontend format-check-frontend lint-frontend \
	format-python format-check-python lint-python \
	scan-security scan-secrets scan-config scan-images compose-validate

hooks-install:
	git config core.hooksPath .githooks

format: format-go format-frontend format-python

format-check: format-check-go format-check-frontend format-check-python

lint: lint-go lint-frontend lint-python

format-go:
	@if [ -z "$(GOFMT)" ]; then echo "gofmt not found" >&2; exit 1; fi
	@find arpego -name '*.go' -type f -print0 | xargs -0 $(GOFMT) -w

format-check-go:
	@if [ -z "$(GOFMT)" ]; then echo "gofmt not found" >&2; exit 1; fi
	@unformatted="$$(find arpego -name '*.go' -type f -exec $(GOFMT) -l {} +)"; \
	if [ -n "$$unformatted" ]; then \
		echo "Go files need formatting:" >&2; \
		echo "$$unformatted" >&2; \
		exit 1; \
	fi

lint-go:
	@if [ -n "$(GOLANGCI_LINT)" ]; then \
		mkdir -p "$(GOLANGCI_CACHE_DIR)"; \
		mkdir -p "$(GO_BUILD_CACHE_DIR)"; \
		cd arpego && GOCACHE="$(GO_BUILD_CACHE_DIR)" GOLANGCI_LINT_CACHE="$(GOLANGCI_CACHE_DIR)" $(GOLANGCI_LINT) run ./...; \
	elif [ -n "$(GO)" ]; then \
		mkdir -p "$(GO_BUILD_CACHE_DIR)"; \
		cd arpego && GOCACHE="$(GO_BUILD_CACHE_DIR)" $(GO) vet ./...; \
	else \
		echo "Neither golangci-lint nor go is available" >&2; \
		exit 1; \
	fi

format-frontend:
	@if [ -z "$(NPM)" ]; then echo "npm not found" >&2; exit 1; fi
	@$(NPM) --prefix libretto run format

format-check-frontend:
	@if [ -z "$(NPM)" ]; then echo "npm not found" >&2; exit 1; fi
	@$(NPM) --prefix libretto run format:check

lint-frontend:
	@if [ -z "$(NPM)" ]; then echo "npm not found" >&2; exit 1; fi
	@$(NPM) --prefix libretto run lint

format-python:
	@if [ -n "$(RUFF)" ]; then \
		cd scripts && $(RUFF) format .; \
	elif [ -n "$(UVX)" ]; then \
		mkdir -p "$(UV_CACHE_DIR)"; \
		cd scripts && UV_CACHE_DIR="$(UV_CACHE_DIR)" $(UVX) ruff format .; \
	else \
		echo "Neither ruff nor uvx is available" >&2; \
		exit 1; \
	fi

format-check-python:
	@if [ -n "$(RUFF)" ]; then \
		cd scripts && $(RUFF) format --check .; \
	elif [ -n "$(UVX)" ]; then \
		mkdir -p "$(UV_CACHE_DIR)"; \
		cd scripts && UV_CACHE_DIR="$(UV_CACHE_DIR)" $(UVX) ruff format --check .; \
	else \
		echo "Neither ruff nor uvx is available" >&2; \
		exit 1; \
	fi

lint-python:
	@if [ -n "$(RUFF)" ]; then \
		cd scripts && $(RUFF) check .; \
	elif [ -n "$(UVX)" ]; then \
		mkdir -p "$(UV_CACHE_DIR)"; \
		cd scripts && UV_CACHE_DIR="$(UV_CACHE_DIR)" $(UVX) ruff check .; \
	else \
		echo "Neither ruff nor uvx is available" >&2; \
		exit 1; \
	fi

scan-security:
	@if [ -z "$(TRIVY)" ]; then echo "trivy not found" >&2; exit 1; fi
	@$(TRIVY) fs --config trivy.yaml .

scan-secrets:
	@if [ -z "$(TRIVY)" ]; then echo "trivy not found" >&2; exit 1; fi
	@$(TRIVY) fs --scanners secret --severity HIGH,CRITICAL .

scan-config:
	@if [ -z "$(TRIVY)" ]; then echo "trivy not found" >&2; exit 1; fi
	@$(TRIVY) config --severity HIGH,CRITICAL .

scan-images:
	@if [ -z "$(TRIVY)" ]; then echo "trivy not found" >&2; exit 1; fi
	@if [ -z "$(DOCKER)" ]; then echo "docker not found" >&2; exit 1; fi
	@$(DOCKER) build -t symphony-go/arpego:local ./arpego
	@$(TRIVY) image --severity HIGH,CRITICAL --ignore-unfixed symphony-go/arpego:local
	@$(DOCKER) build -t symphony-go/libretto:local ./libretto
	@$(TRIVY) image --severity HIGH,CRITICAL --ignore-unfixed symphony-go/libretto:local

compose-validate:
	@if [ -n "$(DOCKER_COMPOSE)" ]; then \
		$(DOCKER_COMPOSE) -f docker-compose.yaml config >/dev/null; \
	elif [ -n "$(DOCKER)" ]; then \
		echo "docker compose plugin not available; install docker-compose to validate compose locally" >&2; \
		exit 1; \
	else \
		echo "docker or docker-compose not found" >&2; \
		exit 1; \
	fi
