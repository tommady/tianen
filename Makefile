build-and-publish-image:
	docker buildx build \
	--platform linux/amd64,linux/arm64,linux/arm/v7 \
  --tag tommady/tianen:latest \
  --push .

.PHONY: build-and-publish-image
