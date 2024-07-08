#!/usr/bin/env bash

# Copyright 2017 The Kubernetes Authors.
# Copyright 2023 The KEDA Authors.
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

SCRIPT_ROOT=$(dirname "${BASH_SOURCE[0]}")/..

DIFFROOT="${SCRIPT_ROOT}/config"
TMP_DIFFROOT="${SCRIPT_ROOT}/_tmp/config"
_tmp="${SCRIPT_ROOT}/_tmp"

cleanup() {
  rm -rf "${_tmp}"
}
trap "cleanup" EXIT SIGINT

yaml2json() {
  python3 -c 'import json, sys, yaml ; y=yaml.safe_load(sys.stdin.read()) ; json.dump(y, sys.stdout)'
}

if ! python3 -c "import yaml" >/dev/null 2>&1; then
  echo "Python module 'yaml' required for this script."
  exit 1
fi

# Make sure all the CRDs are listed in the kustomize resource list
declare -A crds
declare -A crs
while read -r filename; do
  crds["$filename"]=1
done < <(sed -n '/^resources:$/,/^[^-]/ s#^- ##p' config/crd/kustomization.yaml)
bad_crd_resource_list=0
for f in config/crd/bases/*.yaml; do
  key="bases/$(basename "$f")"
  if [ ! -v "crds[${key}]" ]; then
    echo "ERROR: CRD file $f is not listed in the resources section of config/crd/kustomization.yaml"
    bad_crd_resource_list=1
  else
    crs[$key]="$(yaml2json < $f | jq -r '.spec.names.singular as $k | (.spec.group | sub("\\..*"; "")) as $g | .spec.versions[] | ($g+"_"+.name+"_"+$k)')"
  fi
done

# Make sure all sample CRs are listed in the kustomize resource list (part 1)
declare -A crslist
while read -r filename; do
  if ! test -f "$filename"; then
    crslist["$filename"]=1
  fi
done < <(sed -n '/^resources:$/,/^[^-]/ s#^- ##p' config/samples/kustomization.yaml)

# Make sure there is a sample CR for each CRD version
for key in ${!crs[@]}; do
  for gvk in ${crs[$key]}; do
    if [ ! -f "config/samples/${gvk}.yaml" ]; then
      echo "ERROR: CRD config/crd/$key does not have a sample CR config/samples/$gvk.yaml"
      bad_crd_resource_list=1
    fi
    # Make sure all sample CRs are listed in the kustomize resource list (part 2)
    if [ ! -v "crslist[${gvk}.yaml]" ]; then
      echo "ERROR: CR config/samples/${gvk}.yaml is not listed in the resources section of config/samples/kustomization.yaml"
      bad_crd_resource_list=1
    fi
  done
done

cleanup

mkdir -p "${TMP_DIFFROOT}"
cp -a "${DIFFROOT}"/* "${TMP_DIFFROOT}"

make manifests
echo "diffing ${DIFFROOT} against freshly generated manifests"
ret=0
diff -Naupr "${DIFFROOT}" "${TMP_DIFFROOT}" || ret=$?
cp -a "${TMP_DIFFROOT}"/* "${DIFFROOT}"
if [[ $ret -eq 0 ]]
then
  echo "${DIFFROOT} up to date."
else
  echo "${DIFFROOT} is out of date. Please run 'make manifests'"
  exit 1
fi

if [ "$bad_crd_resource_list" != 0 ]; then
  echo "Check failed due to previous errors. See output above"
  exit 1
fi
