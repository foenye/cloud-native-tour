.PHONY: controller-gen
controller-gen: # find or download controller if necessary
ifeq (, $(shell which kind))
	go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest
endif

.PHONY: gen-crd
gen-crd: controller-gen
	controller-gen crd:crdVersions=v1 paths=./pkg/apis/greeting/v1/... output:dir=./config/crd/bases

.PHONY: gen-crd-no-describe
gen-crd-no-describe: controller-gen
	controller-gen crd:crdVersions=v1,maxDescLen=0 paths=./pkg/apis/greeting/v1/... output:dir=./config/crd/bases

.PHONY: patch-crd
patch-crd: controller-gen
	controller-gen schemapatch:manifests=./config/crd/bases paths=./pkg/apis/greeting/v1/... output:dir=./config/crd/bases

PROJECT_MOD := github.com/foenye/cloud-native-tour/crd-getting-started

.PHONY: gen
gen:
	docker run --rm \
	-v ./:/go/src/$(PROJECT_MOD) \
	-e PROJECT_MOD=$(PROJECT_MOD) \
	-e API_ROOT=pkg/apis \
	-e API_PACKAGES="$(PROJECT_MOD)/pkg/apis/greeting/v1 \
		$(PROJECT_MOD)/pkg/apis/greeting/v2" \
	kube-code-generator

.PHONY: gen-crd
gen-crd:
	docker run --rm \
	-v ./:/go/src/$(PROJECT_MOD) \
	-e PROJECT_MOD=$(PROJECT_MOD) \
	-e CRD_VERSION=v1 \
	-e CRD_TYPE_PATH=pkg/apis/greeting/v2 \
	kube-code-generator crd