REGISTRY ?= zot.ui.sparky.best
BLOG_IMAGE := $(REGISTRY)/personal-blog
CC_LIVE_IMAGE := $(REGISTRY)/cc-live
SHA := $(shell git rev-parse --short HEAD)
HOMESERVER_DIR ?= ../homeserver

.PHONY: build push login deploy \
        build-cc-live push-cc-live deploy-cc-live \
        build-daemon \
        build-all push-all deploy-all \
        dev dev-down dev-heartbeat

# --- Blog ---

build:
	podman build --platform linux/amd64 -f Containerfile -t $(BLOG_IMAGE):$(SHA) -t $(BLOG_IMAGE):latest .

push: build
	podman push $(BLOG_IMAGE):$(SHA)
	podman push $(BLOG_IMAGE):latest

deploy: push
	cd $(HOMESERVER_DIR) && \
		pulumi config set homeserver:blogTag $(SHA) && \
		pulumi up

# --- cc-live ---

build-cc-live:
	podman build --platform linux/amd64 -f services/cc-live/Containerfile -t $(CC_LIVE_IMAGE):$(SHA) -t $(CC_LIVE_IMAGE):latest services/cc-live/

push-cc-live: build-cc-live
	podman push $(CC_LIVE_IMAGE):$(SHA)
	podman push $(CC_LIVE_IMAGE):latest

deploy-cc-live: push-cc-live
	cd $(HOMESERVER_DIR) && \
		pulumi config set homeserver:ccLiveTag $(SHA) && \
		pulumi up

# --- cc-live daemon (local) ---

build-daemon:
	mkdir -p $(HOME)/.cc-live
	cd scripts/cc-live && go build -o $(HOME)/.cc-live/cc-live-daemon .

# --- All ---

build-all: build build-cc-live

push-all: push push-cc-live

deploy-all: push-all
	cd $(HOMESERVER_DIR) && \
		pulumi config set homeserver:blogTag $(SHA) && \
		pulumi config set homeserver:ccLiveTag $(SHA) && \
		pulumi up

# --- Local dev ---

login:
	podman login $(REGISTRY)

dev:
	docker compose up --build

dev-down:
	docker compose down

dev-heartbeat:
	curl -sf -X POST http://localhost:8000/api/live/heartbeat \
		-H "Authorization: Bearer dev-secret"
