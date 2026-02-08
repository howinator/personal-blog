REGISTRY ?= <tailscale-ip>:5000
IMAGE := $(REGISTRY)/personal-blog
TAG ?= latest

.PHONY: build push login

build:
	docker build -t $(IMAGE):$(TAG) .

push: build
	docker push $(IMAGE):$(TAG)

login:
	docker login $(REGISTRY)
