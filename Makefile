localenv: kind kubectl ko kustomize
	./hack/setup-kind-with-registry.sh
kind: # find or download kind if necessary
ifeq (, $(shell which kind))
	go install sigs.k8s.io/kind@latest
endif

kubectl: # find or download kubectl if necessary
ifeq (, $(shell which kubectl))
	curl -LO https://dl.k8s.io/release/v1.31.3/bin/linux/amd64/kubectl
	sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl
	rm kubectl
endif

ko: # find or download ko if necessary
ifeq (, $(shell which ko))
	go install github.com/google/ko@latest
endif

kustomize: # find or download kustomize if necessary
ifeq (, $(shell which kustomize))
	go install sigs.k8s.io/kustomize/kustomize/v5@v5.5.0
endif