PROJECT_IMAGE_NAME ?= word-agent:1.0.0

.PHONY: build-project ensure-project build-deps

build-project:
	docker build -f Dockerfile.project -t $(PROJECT_IMAGE_NAME) .

ensure-project:
	@docker image inspect $(PROJECT_IMAGE_NAME) >/dev/null 2>&1 || $(MAKE) build-project

build-deps: ensure-project
	docker build -f Dockerfile.deps --build-arg PROJECT_IMAGE=$(PROJECT_IMAGE_NAME) -t $(PROJECT_IMAGE_NAME) .
