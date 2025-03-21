##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install:
	$(KUSTOMIZE) build config/crd | $(KUBECTL) apply -f -

.PHONY: uninstall
uninstall:
	$(KUSTOMIZE) build config/crd | $(KUBECTL) delete --ignore-not-found=$(ignore-not-found) -f -


.PHONY: deploy
deploy:
	$(KUSTOMIZE) build config/samples | $(KUBECTL) apply -f -

.PHONY: undeploy
undeploy:
	$(KUSTOMIZE) build config/samples | $(KUBECTL) delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: discovery-v6
discovery-v6:
	$(KUBECTL) get fo --cache-dir $(shell mktemp -d) -v 6

.PHONY: discovery-v8
discovery-v8:
	$(KUBECTL) get fo --cache-dir $(shell mktemp -d) -v 8

.PHONY: proxy-discovery/apis
proxy-discovery/apis:
	$(KUBECTL) proxy &
	sleep 0.2
	curl -H 'Accept: application/yaml;g=apidiscovery.k8s.io;v=v2beta1;as=APIGroupDiscoveryList' localhost:8001/apis
	pkill -f "kubectl proxy"

.PHONY: api-resource-discovery-v6
api-resource-discovery-v6:
	$(KUBECTL) api-resources --cache-dir $(shell mktemp -d) -v 6 | awk 'NR==1 || /pods|fo|deploy/'

.PHONY: proxy/apis
proxy/apis:
	$(KUBECTL) proxy &
	sleep 0.2
	curl -H 'Accept: application/yaml' localhost:8001/apis
	pkill -f "kubectl proxy"

.PHONY: proxy/apis/greeting.foen.ye/v1
proxy/apis/greeting.foen.ye/v1:
	$(KUBECTL) proxy &
	sleep 0.2
	curl -H 'Accept: application/yaml' localhost:8001/apis/greeting.foen.ye/v1
	pkill -f "kubectl proxy"

.PHONY: proxy/openapi/v3/apis/greeting.foen.ye/v1
proxy/openapi/v3/apis/greeting.foen.ye/v1:
	$(KUBECTL) proxy &
	sleep 0.2
	curl -s localhost:8001/openapi/v3/apis/greeting.foen.ye/v1 | jq 'delpaths([path(.), path(..) | select(length >3)])'
	pkill -f "kubectl proxy"

.PHONY: apply-invalid-required
apply-invalid-required:
	cat config/samples/invcalid.required.yaml | $(KUBECTL) apply -f -

.PHONY: apply-invalid-field
apply-invalid-field:
	cat config/samples/invcalid.field.yaml | $(KUBECTL) apply -f -

.PHONY: clean
clean: undeploy uninstall

## Tool Binaries
KUBECTL ?= kubectl
KUSTOMIZE ?= kustomize