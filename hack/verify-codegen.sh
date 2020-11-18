#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_ROOT=$(dirname "${BASH_SOURCE[0]}")/..

DIFFROOT="${SCRIPT_ROOT}/pkg"
TMP_DIFFROOT="${SCRIPT_ROOT}/_tmp/pkg"
_tmp="${SCRIPT_ROOT}/_tmp"

cleanup() {
  rm -rf "${_tmp}"
}
trap "cleanup" EXIT SIGINT

cleanup

mkdir -p "${TMP_DIFFROOT}"
cp -a "${DIFFROOT}"/* "${TMP_DIFFROOT}"

"${SCRIPT_ROOT}/hack/update-codegen.sh"

# (Zbynek): Kubebuilder project layout has api under 'api/v1alpha1'
# client-go codegen expects group name in the path ie. 'api/keda/v1alpha'
# Because there's no way how to modify any of these settings,
# we need to hack things a little bit (replace the name of package)
find "${DIFFROOT}/generated" -type f -name "*.go" |\
xargs sed -i "s#github.com/kedacore/keda/v2/api/keda/v1alpha1#github.com/kedacore/keda/v2/api/v1alpha1#g"

echo "diffing ${DIFFROOT} against freshly generated codegen"
ret=0
diff -Naupr "${DIFFROOT}" "${TMP_DIFFROOT}" || ret=$?
cp -a "${TMP_DIFFROOT}"/* "${DIFFROOT}"
if [[ $ret -eq 0 ]]
then
  echo "${DIFFROOT} up to date."
else
  echo "${DIFFROOT} is out of date. Please run make clientset-generate"
  exit 1
fi
