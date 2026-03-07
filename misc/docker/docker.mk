DOCKER_IMAGE_NAME := xolo

DOCKER_IMAGE_CHANNEL=$(shell git rev-parse --abbrev-ref HEAD)
COMMIT_TIMESTAMP=$(shell git show -s --format=%ct)
DOCKER_IMAGE_TAG ?= $(shell TZ=Europe/Paris date -d "@$(COMMIT_TIMESTAMP)" +%Y.%-m.%-d)-$(DOCKER_IMAGE_CHANNEL).$(shell date -d "@${COMMIT_TIMESTAMP}" +%-H%M).$(shell git rev-parse --short HEAD)

docker-image:
	docker build -f misc/docker/Dockerfile -t $(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_TAG) .

docker-run:
	docker run -it --rm  --name xolo --env-file .env $(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_TAG)

docker-release: docker-image
	docker tag $(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_TAG) $(DOCKER_IMAGE_NAME):latest
	docker push $(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_TAG)
	docker push $(DOCKER_IMAGE_NAME):latest