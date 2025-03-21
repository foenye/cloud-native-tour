go_vendor:
ifeq (, $(wildcard ./vendor))
	go mod tidy;
	go mod vendor;
endif

register-gen: go_vendor
ifeq (, $(shell which ./register-gen))
	go build ./vendor/k8s.io/code-generator/cmd/register-gen
endif

gen-register: register-gen
	register-gen \
	--output-file zz_generated.register.go \
	--go-header-file ./hack/boilerplate.go.txt \
	github.com/foenye/cloud-native-tour/codegen/pkg/apis/greeting/v1
