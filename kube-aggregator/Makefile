IMAGE := golang:1.24.1-alpine
PROTOC_VERSION := 30.2
KUBE_VERSION := 1.32.3
CONTROLLER_GEN_VERSION := 0.17.2

.PHONY: gen
gen:
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