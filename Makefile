# Use git tag/git branch to tag Docker image.
CURRENT_DIR             ?= $(shell pwd)
VERSION                 ?= $(subst /,-,$(shell git describe --tags --always --abbrev=0 --dirty='-dev' || git rev-parse --abbrev-ref HEAD))
DOCKER_USERNAME         ?= kiennt26
DOCKER_IMAGE_NAME       ?= faythe
DOCKER_IMAGE_FULL       ?= $(DOCKER_USERNAME)/$(DOCKER_IMAGE_NAME):$(VERSION)
DOCKER_IMAGE_LATEST     ?= $(DOCKER_USERNAME)/$(DOCKER_IMAGE_NAME):latest
DOCKER_CONTAINER_NAME   ?= faythe
FAYTHE_PORT             ?= 8600
FAYTHE_CONF_PATH        ?= $(CURRENT_DIR)/examples/faythe.yml

build:
	docker build -t "$(DOCKER_IMAGE_FULL)" .

build-latest: build
	docker tag "$(DOCKER_IMAGE_FULL)" "$(DOCKER_IMAGE_LATEST)"

push: build
	docker push "$(DOCKER_IMAGE_FULL)"

push-latest: build-latest
	docker push "$(DOCKER_IMAGE_LATEST)"

run:
	docker rm -f "$(DOCKER_CONTAINER_NAME)" || true
	docker run -d --restart always -p "$(FAYTHE_PORT)":8600 \
	    -v "$(FAYTHE_CONF_PATH)":/etc/faythe/config.yml \
	    --name "$(DOCKER_CONTAINER_NAME)" "$(DOCKER_IMAGE_FULL)"
