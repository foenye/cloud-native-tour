# [make install error](https://github.com/kubernetes-sigs/kubebuilder/issues/1140)

**The CustomResourceDefinition "..." is invalid: metadata.annotations: Too long: must have at most 262144 characters**

- Addon `--server-side` options:

```makefile
.PHONY: install
install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | $(KUBECTL) apply --server-side -f -
```

# kind load docker-image

***ERROR: failed to detect containerd snapshotter***

- reinstall kind

```shell
rm -rf $GOBIN/kind
make localenv
```