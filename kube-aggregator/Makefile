IMAGE := golang:1.24.1-alpine
PROTOC_VERSION := 30.2
KUBE_VERSION := 1.32.3
CONTROLLER_GEN_VERSION := 0.17.2

.PHONY: env
env: protoc-bin kube-src


.PHONY: protoc-bin
protoc-bin:
ifeq (, $(wildcard /tmp/protoc.zip))
	curl -fsSL https://ghraw.foenye.ip-ddns.com/https://github.com/protocolbuffers/protobuf/releases/download/v$(PROTOC_VERSION)/protoc-$(PROTOC_VERSION)-linux-x86_64.zip -o /tmp/protoc.zip
endif

.PHONY: kube-src
kube-src:
ifeq (, $(wildcard /tmp/kube-src.tar.gz))
	curl -fsSL https://ghraw.foenye.ip-ddns.com/https://github.com/kubernetes/kubernetes/archive/v$(KUBE_VERSION).tar.gz -o /tmp/kube-src.tar.gz
endif

.PHONY: setup
setup: env
	docker run --rm -it \
		-e KUBE_VERSION=$(KUBE_VERSION) \
		-e CONTROLLER_GEN_VERSION=$(CONTROLLER_GEN_VERSION) \
		-e GOPROXY=https://goproxy.cn \
		-v /tmp/kube-src.tar.gz:/kube-src.tar.gz \
		-v /tmp/protoc.zip:/protoc.zip \
		-v ./:/go/src/github.com/foenye/cloud-native-tour/kube-aggregator \
		$(IMAGE) \
		sh -c '/go/src/github.com/foenye/cloud-native-tour/kube-aggregator/hack/setup.sh && \
			   sed -i 's#dl-cdn.alpinelinux.org#mirrors.tuna.tsinghua.edu.cn#g' /etc/apk/repositories && \
			   apk add --no-cache --purge -uU bash && bash'



PROJECT_MOD := github.com/foenye/cloud-native-tour/kube-aggregator

.PHONY: gen-protoc
gen-protoc:
	docker run --rm \
	-v ./:/go/src/$(PROJECT_MOD) \
	-e PROJECT_MOD=$(PROJECT_MOD) \
	-e API_PACKAGES="$(PROJECT_MOD)/pkg/apis/registration/v1 \
		$(PROJECT_MOD)/pkg/apis/registration/v1beta1" \
	kube-code-generator

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