FRONTEND_DIR = ./web
BACKEND_DIR = .

# Docker image build
REGISTRY ?= 159.75.178.153:18098
releaseType ?= dev
APP_NAME ?= new-api

.PHONY: all build-frontend start-backend build

all: build-frontend start-backend

build-frontend:
	@echo "Building frontend..."
	@cd $(FRONTEND_DIR) && bun install && DISABLE_ESLINT_PLUGIN='true' VITE_REACT_APP_VERSION=$(cat VERSION) bun run build

start-backend:
	@echo "Starting backend dev server..."
	@cd $(BACKEND_DIR) && go run main.go &

build:
	@if [ "$(releaseType)" = "release" ]; then \
		LATEST_TAG=$$(git tag --sort=-v:refname | head -n 1); \
		if [ -z "$$LATEST_TAG" ]; then LATEST_TAG="none"; fi; \
		printf "Git tag version (latest: %s, e.g. v0.1.0): " "$$LATEST_TAG"; \
		read VERSION; \
		if [ -z "$$VERSION" ]; then echo "tag required"; exit 1; fi; \
		git tag "$$VERSION"; \
		IMAGE_TAG="$(REGISTRY)/$(releaseType)/$(APP_NAME):$$VERSION"; \
	elif [ "$(releaseType)" = "dev" ]; then \
		GIT_BRANCH=$$(git rev-parse --abbrev-ref HEAD); \
		GIT_HASH=$$(git rev-parse --short=7 HEAD); \
		VERSION="$$GIT_BRANCH-$$GIT_HASH"; \
		IMAGE_TAG="$(REGISTRY)/$(releaseType)/$(APP_NAME):$$VERSION"; \
	else \
		echo "releaseType must be dev or release"; \
		exit 1; \
	fi; \
	echo "[docker] building $$IMAGE_TAG"; \
	docker buildx build --platform linux/amd64 --push -t "$$IMAGE_TAG" .
