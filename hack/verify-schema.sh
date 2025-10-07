#!/usr/bin/env bash

# Copyright 2025 The KEDA Authors.
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

# set -o errexit
set -o nounset
set -o pipefail

SCRIPT_ROOT=$(dirname "${BASH_SOURCE[0]}")/..

SCHEMAROOT="${SCRIPT_ROOT}/schema/generated"
TMP_DIFFROOT="${SCRIPT_ROOT}/_tmp/schema"
_tmp="${SCRIPT_ROOT}/_tmp"

cleanup() {
  rm -rf "${_tmp}"
}
trap "cleanup" EXIT SIGINT

# Make sure schema json file has correct format
find "$SCHEMAROOT" -name "*.json" | while read file; do
  if jq -e . $file >/dev/null 2>&1; then
      echo "Parsed JSON successfully and got something other than false/null"
  else
      echo "Failed to parse JSON, or got false/null from $file"
      break
  fi

   err_line_content=$(grep -vE '"kedaVersion":.+|"schemaVersion":.+|"scalers": \[|"metadata": \[|"parameters": \[|"optional":.+|"default":.+|"canReadFromEnv":.+|"canReadFromAuth":.+|"metadataVariableReadable":.+|"envVariableReadable":.+|"triggerAuthenticationVariableReadable":.+|"type":.+|"name":.+|"rangeSeparator":.+|"separator":.+|"allowedValue": \[|"deprecatedAnnounce":.+|"deprecated":.+|^[^:]*$' "$file")

    if [ ! -z "$err_line_content" ]; then
        echo "ERROR: error schema format found: $err_line_content in $file"
        exit 1
    fi
done
echo "Schema json files are in correct format"

# Make sure schema yaml file has correct format
find $SCHEMAROOT -name "*.yaml" | while read file; do
   err_line_content=$(grep -vE "kedaVersion:.+|schemaVersion:.+|scalers:|metadata:|parameters:|optional:.+|default:.+|canReadFromEnv:.+|canReadFromAuth:.+|metadataVariableReadable:.+|envVariableReadable:.+|triggerAuthenticationVariableReadable:.+|type:.+|name:.+|rangeSeparator:.+|separator:.+|allowedValue:|deprecatedAnnounce:.+|deprecated:.+|^[^:]*$" "$file")
    if [ ! -z "$err_line_content" ]; then
        echo "ERROR: error schema format found: $err_line_content in $file"
        exit 1
    fi
done
echo "Schema yaml files are in correct format"

cleanup

mkdir -p "${TMP_DIFFROOT}"
cp -a "${SCHEMAROOT}"/* "${TMP_DIFFROOT}"

make generate-scalers-schema
echo "diffing ${SCHEMAROOT} against freshly generated scalers schema"
ret=0
diff -Naup "${SCHEMAROOT}" "${TMP_DIFFROOT}" || ret=$?
cp -a "${TMP_DIFFROOT}"/* "${SCHEMAROOT}"
if [[ $ret -eq 0 ]]
then
  echo "${SCHEMAROOT} up to date."
else
  echo "${SCHEMAROOT} is out of date. Please run 'make generate-scalers-schema'"
  exit 1
fi
