```shell
docker run -it --rm -v $PWD:/app ghcr.io/slok/kube-code-generator:v0.3.2 --apis-in ./ --go-gen-out ./generated --crd-gen-out ./generated/manifests
```