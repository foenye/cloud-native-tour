IMAGE := golang:1.24.1-alpine
PROTOC_VERSION := 30.2
KUBE_VERSION := 1.32.3
CONTROLLER_GEN_VERSION := 0.17.2

.PHONY: env
env: protoc-bin kube-src


.PHONY: protoc-bin
protoc-bin:
ifeq (, $(wildcard /tmp/protoc.zip))
	curl -fsSL ${GITHUB_RAW}https://github.com/protocolbuffers/protobuf/releases/download/v$(PROTOC_VERSION)/protoc-$(PROTOC_VERSION)-linux-x86_64.zip -o /tmp/protoc.zip
endif

.PHONY: kube-src
kube-src:
ifeq (, $(wildcard /tmp/kube-src.tar.gz))
	curl -fsSL ${GITHUB_RAW}/https://github.com/kubernetes/kubernetes/archive/v$(KUBE_VERSION).tar.gz -o /tmp/kube-src.tar.gz
endif

.PHONY: gen
gen: env
	docker run --rm \
    	-v ./:/go/src/$(PROJECT_MOD) \
    	-e PROJECT_MOD=$(PROJECT_MOD) \
    	-e API_PACKAGES="$(PROJECT_MOD)/pkg/apis/registration/v1 \
    		$(PROJECT_MOD)/pkg/apis/registration/v1beta1" \
    	kube-code-generator



PROJECT_MOD := github.com/foenye/cloud-native-tour/kube-aggregator

.PHONY: gen-protobuf
gen-protobuf:
	docker run --rm \
	-v ./:/go/src/$(PROJECT_MOD) \
	-e PROJECT_MOD=$(PROJECT_MOD) \
	-e API_PACKAGES="$(PROJECT_MOD)/pkg/apis/registration/v1 \
		$(PROJECT_MOD)/pkg/apis/registration/v1beta1" \
	kube-code-generator protobuf

gen-crd:
	docker run --rm \
	-v ./:/go/src/$(PROJECT_MOD) \
	-e PROJECT_MOD=$(PROJECT_MOD) \
	-e CRD_VERSION=v1beta1 \
	-e CRD_TYPE_PATH=pkg/apis/registration/v1beta1 \
	kube-code-generator crd
# 	PROJECT_MOD=github.com/foenye/cloud-native-tour/kube-aggregator
#	docker run --rm -it \
#	-v ./:/go/src/${PROJECT_MOD} \
#	-e PROJECT_MOD=${PROJECT_MOD} \
#	-e API_PACKAGES="${PROJECT_MOD}/pkg/apis/registration/v1 \
#		${PROJECT_MOD}/pkg/apis/registration/v1beta1" \
#	kube-code-generator bash