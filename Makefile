REGISTRY ?= zot.ui.sparky.best
IMAGE := $(REGISTRY)/personal-blog
SHA := $(shell git rev-parse --short HEAD)
HOMESERVER_DIR ?= ../homeserver

.PHONY: build push login deploy

build:
	podman build --platform linux/amd64 -f Containerfile -t $(IMAGE):$(SHA) -t $(IMAGE):latest .

push: build
	podman push $(IMAGE):$(SHA)
	podman push $(IMAGE):latest

login:
	podman login $(REGISTRY)

deploy: push
	cd $(HOMESERVER_DIR) && \
		pulumi config set homeserver:blogTag $(SHA) && \
		pulumi up
