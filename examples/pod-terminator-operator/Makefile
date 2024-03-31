
IMAGE_GEN=ghcr.io/slok/kube-code-generator:v0.1.0

default: generate

generate: 
	@docker run --rm -v ${PWD}:/app "${IMAGE_GEN}" \
        --apis-in ./apis \
        --go-gen-out ./client/k8s \
        --crd-gen-out ./manifests

generate-debug: 
	@docker run --entrypoint /bin/bash --rm -it -v ${PWD}:/app "${IMAGE_GEN}"
