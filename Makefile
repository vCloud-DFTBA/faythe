# Use git tag/git branch to tag Docker image.
DOCKER_IMAGE_TAG        ?= $(subst /,-,$(shell git describe --tags --abbrev=0 || git rev-parse --abbrev-ref HEAD))
DOCKER_REPO             ?= ntk148v
DOCKER_IMAGE_NAME       ?= faythe
DOCKER_IMAGE_FULL       ?= $(DOCKER_REPO)/$(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_TAG)
DOCKER_IMAGE_LATEST     ?= $(DOCKER_REPO)/$(DOCKER_IMAGE_NAME):latest
FAYTHE_PORT             ?= 8600
DOCKER_CONTAINER_NAME   ?= faythe

build:
	docker build -t "$(DOCKER_IMAGE_FULL)" .

build-latest: build
	docker tag "$(DOCKER_IMAGE_FULL)" "$(DOCKER_IMAGE_LATEST)"

push:
	docker push "$(DOCKER_IMAGE_FULL)"

push-latest:
	docker push "$(DOCKER_IMAGE_LATEST)"

run: build
	docker rm -f "$(DOCKER_CONTAINER_NAME)" || true
	docker run -d --net host --name "$(DOCKER_CONTAINER_NAME)" "$(DOCKER_IMAGE_FULL)"
