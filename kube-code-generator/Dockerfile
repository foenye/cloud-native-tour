FROM golang:1.24.1-bullseye
ARG KUBE_VERSION="1.32.3"
ARG PROTOC_VERSION="30.2"
ARG CONTROLLER_GEN_VERSION="0.17.2"
ARG GITHUB_RAW=${GITHUB_RAW}
ENV GOPROXY=https://goproxy.cn
ENV PROJECT_MOD=""

RUN sed -i 's#deb.debian.org#mirrors.aliyun.com#g' /etc/apt/sources.list && \
      apt update -qq && \
      apt install -y --no-install-recommends unzip && \
      apt clean && rm -rf /var/lib/apt/lists /var/cache/apt/archives && \
    \
    wget ${GITHUB_RAW}https://github.com/kubernetes/kubernetes/archive/v${KUBE_VERSION}.tar.gz && \
        mkdir -p /go/src/k8s.io/code-generator/ && \
        mkdir -p /go/src/k8s.io/apimachinery/ && \
        mkdir -p /go/src/github.com/gogo/ && \
        mkdir -p /go/src/k8s.io/kubernetes/third_party/protobuf/ && \
        tar zxvf v${KUBE_VERSION}.tar.gz --strip 5 -C /go/src/k8s.io/code-generator/ kubernetes-${KUBE_VERSION}/staging/src/k8s.io/code-generator && \
        tar zxvf v${KUBE_VERSION}.tar.gz --strip 5 -C /go/src/k8s.io/apimachinery/ kubernetes-${KUBE_VERSION}/staging/src/k8s.io/apimachinery && \
        tar zxvf v${KUBE_VERSION}.tar.gz --strip 4 -C /go/src/github.com/gogo/ kubernetes-${KUBE_VERSION}/vendor/github.com/gogo && \
        tar zxvf v${KUBE_VERSION}.tar.gz --strip 3 -C /go/src/k8s.io/kubernetes/third_party/protobuf/ kubernetes-${KUBE_VERSION}/third_party/protobuf && \
        rm v${KUBE_VERSION}.tar.gz && \
    \
    wget ${GITHUB_RAW}https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-linux-x86_64.zip && \
        unzip protoc-${PROTOC_VERSION}-linux-x86_64.zip -d protoc && \
        cp protoc/bin/protoc /usr/local/bin && \
        rm -rf protoc* && \
        cd /go/src/k8s.io/code-generator/ && \
        go build -o /usr/bin/protoc-gen-gogo ./cmd/go-to-protobuf/protoc-gen-gogo && \
        go build -o /usr/bin/go-to-protobuf  ./cmd/go-to-protobuf && \
        GOBIN=/usr/bin go install golang.org/x/tools/cmd/goimports@latest && \
        cd - && \
    \
    GOBIN=/usr/bin go install sigs.k8s.io/controller-tools/cmd/controller-gen@v${CONTROLLER_GEN_VERSION} && \
      rm -rf /go/pkg

COPY protobuf /usr/local/bin/protobuf
COPY generation /usr/local/bin/generation
COPY register /usr/local/bin/register
COPY helpers /usr/local/bin/helpers
COPY client /usr/local/bin/client
COPY openapi /usr/local/bin/openapi
COPY aggregator_violation_exceptions.list /k8s-openapi/aggregator_violation_exceptions.list
COPY crd /usr/local/bin/crd

CMD ["generation"]

# docker buildx build --build-arg GITHUB_RAW=$GITHUB_RAW -t kube-code-generator .