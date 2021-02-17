build-and-publish-image:
	docker buildx build --platform linux/amd64,linux/arm64,linux/arm/v7 --tag tommady/tianen:latest --push .

deploy:
	kubectl kustomize ./kustomize/base/. | kubectl apply -f -
