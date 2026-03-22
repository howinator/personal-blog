REGISTRY ?= zot.ui.sparky.best
BLOG_IMAGE := $(REGISTRY)/personal-blog
SHA := $(shell git rev-parse --short HEAD)
HOMESERVER_DIR ?= ../homeserver/hosting

.PHONY: build push login deploy \
        sync \
        dev-static dev dev-down \
        test test-js \
        sync-plots \
        maintenance-on maintenance-off

# --- Blog ---

sync:
	cd scripts/build-sessions && CC_STATS_BLOG_ROOT="$$(cd ../../site && pwd)" go run .

build: sync
	podman build --platform linux/amd64 -f Containerfile -t $(BLOG_IMAGE):$(SHA) -t $(BLOG_IMAGE):latest .

push: build
	podman push $(BLOG_IMAGE):$(SHA)
	podman push $(BLOG_IMAGE):latest

deploy: push
	cd $(HOMESERVER_DIR) && \
		pulumi config set homeserver:blogTag $(SHA) && \
		pulumi up

# --- Local dev ---

login:
	podman login $(REGISTRY)

dev-static:
	hugo server -s site

dev:
	docker compose up --build

dev-down:
	docker compose down

# --- Testing & Linting ---

test: test-js

test-js:
	pnpm test
# --- Maintenance mode ---

maintenance-on:
	./scripts/maintenance.sh on

maintenance-off:
	./scripts/maintenance.sh off

# --- Plots ---

sync-plots:
	uv sync --project scripts/plots
