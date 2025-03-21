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

