#!/usr/bin/env bash

# Copyright 2017 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
SCRIPT_ROOT="${SCRIPT_DIR}/.."
CODEGEN_PKG=${CODEGEN_PKG:-$(cd "${SCRIPT_ROOT}"; ls -d -1 ./vendor/k8s.io/code-generator 2>/dev/null || echo ../code-generator)}

source "${CODEGEN_PKG}/kube_codegen.sh"

THIS_PKG="github.com/yeahfo/cloud-native-tour/api"

kube::codegen::gen_helpers \
    --boilerplate "${SCRIPT_ROOT}/hack/boilerplate.go.txt" \
    "${SCRIPT_ROOT}"

kube::codegen::gen_register \
      --boilerplate "${SCRIPT_ROOT}/hack/boilerplate.go.txt" \
       "${SCRIPT_ROOT}/hello.yeahfo.github.io"

kube::codegen::gen_register \
      --boilerplate "${SCRIPT_ROOT}/hack/boilerplate.go.txt" \
       "${SCRIPT_ROOT}/transformation"

# UPDATE_API_KNOWN_VIOLATIONS=true ./hack/update-codegen.sh
API_KNOWN_VIOLATIONS_DIR=${SCRIPT_ROOT}/hack
if [[ -n "${API_KNOWN_VIOLATIONS_DIR:-}" ]]; then
    report_filename="${API_KNOWN_VIOLATIONS_DIR}/codegen_violation_exceptions.list"
    if [[ "${UPDATE_API_KNOWN_VIOLATIONS:-}" == "true" ]]; then
        update_report="--update-report"
    fi
fi

kube::codegen::gen_openapi \
    --output-dir "${SCRIPT_ROOT}/generated/openapi" \
    --output-pkg "k8s.io/${THIS_PKG}/generated/openapi" \
    --report-filename "${report_filename:-"/dev/null"}" \
    ${update_report:+"${update_report}"} \
    --boilerplate "${SCRIPT_ROOT}/hack/boilerplate.go.txt" \
    "${SCRIPT_ROOT}"

kube::codegen::gen_client \
    --with-watch \
    --with-applyconfig \
    --output-dir "${SCRIPT_ROOT}/generated" \
    --output-pkg "${THIS_PKG}/generated" \
    --boilerplate "${SCRIPT_ROOT}/hack/boilerplate.go.txt" \
    --prefers-protobuf \
    "${SCRIPT_ROOT}"


#go-to-protobuf \
#    --packages github.com/yeahfo/cloud-native-tour/code-generator-examples/apiserver/apis/core/v1 \
#    --apimachinery-packages=+k8s.io/apimachinery/pkg/util/intstr,+k8s.io/apimachinery/pkg/api/resource,+k8s.io/apimachinery/pkg/runtime/schema,+k8s.io/apimachinery/pkg/runtime,k8s.io/apimachinery/pkg/apis/meta/v1,k8s.io/api/core/v1,k8s.io/api/policy/v1beta1 \
#    --go-header-file "${SCRIPT_ROOT}/hack/boilerplate.go.txt"

#kube::codegen::gen_client \
#    --with-watch \
#    --with-applyconfig \
#    --output-dir "${SCRIPT_ROOT}/crd" \
#    --output-pkg "${THIS_PKG}/crd" \
#    --boilerplate "${SCRIPT_ROOT}/hack/boilerplate.go.txt" \
#    "${SCRIPT_ROOT}/crd/apis"
#
#kube::codegen::gen_client \
#    --with-watch \
#    --with-applyconfig \
#    --output-dir "${SCRIPT_ROOT}/single" \
#    --output-pkg "${THIS_PKG}/single" \
#    --boilerplate "${SCRIPT_ROOT}/hack/boilerplate.go.txt" \
#    --one-input-api "api" \
#    "${SCRIPT_ROOT}/single"
#
#kube::codegen::gen_client \
#    --with-watch \
#    --with-applyconfig \
#    --output-dir "${SCRIPT_ROOT}/MixedCase" \
#    --output-pkg "${THIS_PKG}/MixedCase" \
#    --boilerplate "${SCRIPT_ROOT}/hack/boilerplate.go.txt" \
#    "${SCRIPT_ROOT}/MixedCase/apis"
#
#kube::codegen::gen_client \
#    --with-watch \
#    --with-applyconfig \
#    --output-dir "${SCRIPT_ROOT}/HyphenGroup" \
#    --output-pkg "${THIS_PKG}/HyphenGroup" \
#    --boilerplate "${SCRIPT_ROOT}/hack/boilerplate.go.txt" \
#    "${SCRIPT_ROOT}/HyphenGroup/apis"