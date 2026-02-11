REGISTRY ?= zot.ui.sparky.best
IMAGE := $(REGISTRY)/personal-blog
TAG ?= latest

.PHONY: build push login

build:
	docker build --platform linux/amd64 -t $(IMAGE):$(TAG) .

push: build
	docker push $(IMAGE):$(TAG)

login:
	docker login $(REGISTRY)
