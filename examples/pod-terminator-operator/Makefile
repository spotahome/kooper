
CODE_GENERATOR_IMAGE := ghcr.io/slok/kube-code-generator:v1.27.0
CRD_GENERATOR_IMAGE :=  ghcr.io/slok/kube-code-generator:v1.27.0
DIRECTORY := $(PWD)
ROOT_DIRECTORY := $(DIRECTORY)/../..
CODE_GENERATOR_PACKAGE := github.com/spotahome/kooper/examples/pod-terminator-operator/v2

generate: generate-client generate-crd

generate-client:
	docker run --rm -it \
		-v $(DIRECTORY):/go/src/$(CODE_GENERATOR_PACKAGE) \
		-e PROJECT_PACKAGE=$(CODE_GENERATOR_PACKAGE) \
		-e CLIENT_GENERATOR_OUT=$(CODE_GENERATOR_PACKAGE)/client/k8s \
		-e APIS_ROOT=$(CODE_GENERATOR_PACKAGE)/apis \
		-e GROUPS_VERSION="chaos:v1alpha1" \
		-e GENERATION_TARGETS="deepcopy,client" \
		$(CODE_GENERATOR_IMAGE)

generate-crd:
	docker run -it --rm \
    	-v $(ROOT_DIRECTORY):/src \
    	-e GO_PROJECT_ROOT=/src/examples/pod-terminator-operator \
    	-e CRD_TYPES_PATH=/src/examples/pod-terminator-operator/apis \
    	-e CRD_OUT_PATH=/src/examples/pod-terminator-operator/manifests \
    	$(CRD_GENERATOR_IMAGE) update-crd.sh