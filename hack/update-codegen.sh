#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
SCRIPT_ROOT="${SCRIPT_DIR}/.."
CODEGEN_PKG=${CODEGEN_PKG:-$(cd "${SCRIPT_ROOT}"; ls -d -1 ./vendor/k8s.io/code-generator 2>/dev/null || echo ../code-generator)}

source "${CODEGEN_PKG}/kube_codegen.sh"

# At some environments (eg. GitHub Actions), due to $GOPATH setting, the codegen output might not be at the expected path
# in the project repo, therefore we should force the output to the specific directory (see --output-base)
# we need to handle (move) the generated files to the correct location in the repo then
CODEGEN_INPUT="github.com/kedacore/keda/v2/apis"
CODEGEN_OUTPUT_BASE="${SCRIPT_ROOT}"/output
CODEGEN_OUTPUT_GENERATED="${CODEGEN_OUTPUT_BASE}"/github.com/kedacore/keda/v2/pkg/generated

kube::codegen::gen_helpers \
    --input-pkg-root "${CODEGEN_INPUT}" \
    --output-base "${CODEGEN_OUTPUT_BASE}" \
    --boilerplate "${SCRIPT_ROOT}/hack/boilerplate.go.txt"

if [[ -n "${API_KNOWN_VIOLATIONS_DIR:-}" ]]; then
    report_filename="${API_KNOWN_VIOLATIONS_DIR}/codegen_violation_exceptions.list"
    if [[ "${UPDATE_API_KNOWN_VIOLATIONS:-}" == "true" ]]; then
        update_report="--update-report"
    fi
fi

kube::codegen::gen_client \
    --versioned-name "keda:v1alpha1" \
    --input-pkg-root "${CODEGEN_INPUT}" \
    --output-base "${CODEGEN_OUTPUT_BASE}" \
    --output-pkg-root "${CODEGEN_OUTPUT_GENERATED}" \
    --boilerplate "${SCRIPT_ROOT}/hack/boilerplate.go.txt"

# (Zbynek): If v2 is specified in go.mod, codegen unfortunately outputs to 'v2/pkg/generated' instead of 'pkg/generated',
# and since we are using a specific ouput for codegen,  we need to move the generated code around the repo a bit
if [ -d "${CODEGEN_OUTPUT_GENERATED}" ]; then
  rm -rf "${SCRIPT_ROOT}"/pkg/generated
  mv "${CODEGEN_OUTPUT_GENERATED}" "${SCRIPT_ROOT}"/pkg/
  rm -rf "${SCRIPT_ROOT}"/output
fi
