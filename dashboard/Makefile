IMAGE_NAME ?= ghcr.io/b-harvest/metisian
TAG ?= latest
PLATFORM ?= linux/amd64
VITE_API_HOST ?= localhost

build:
	VITE_API_HOST=$(VITE_API_HOST) npm run build
	docker buildx build --platform=$(PLATFORM) -t $(IMAGE_NAME):$(TAG) --push .

run:
	docker run -d -p 8001:5173 -e VITE_API_HOST=$(VITE_API_HOST):8888 $(IMAGE_NAME):$(TAG)

.PHONY: build run
