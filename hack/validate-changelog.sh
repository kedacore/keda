#!/bin/bash

SCRIPT_ROOT=$(dirname "${BASH_SOURCE[0]}")/..

# Define filename
filename="$SCRIPT_ROOT/CHANGELOG.md"

# Check if file exists
if [[ ! -f "$filename" ]]; then
    echo "Error: $filename does not exist."
    exit 1
fi

# Storing the version to be checked
mapfile -t versions < <(awk '/## History/{flag=1;next}/## /{flag=0}flag' "$filename" | grep -o '\[[^]]*\]' | grep -v "v1." | sed 's/[][]//g')

# Define a function to extract and sort sections
function extract_and_check() {
  local section=$1
  local content_block=$2
  local content=$(awk "/### $section/{flag=1;next}/### /{flag=0}flag" <<< "$content_block" | grep '^- \*\*')

  # Skip if content does not exist
  if [[ -z "$content" ]]; then
    return
  fi

  # Separate and sort the **General**: lines
  local sorted_general_lines=$(echo "$content" | grep '^- \*\*General\*\*:' | sort)

  # Sort the remaining lines
  local sorted_content=$(echo "$content" | grep -v '^- \*\*General\*\*:' | sort)

  # Check if sorted_general_lines is not empty, then concatenate
  if [[ -n "$sorted_general_lines" ]]; then
      sorted_content=$(printf "%s\n%s" "$sorted_general_lines" "$sorted_content")
  fi

  # Check pattern and throw error if wrong pattern found
  while IFS= read -r line; do
      echo "Error: Wrong pattern found in section: $section , line: $line"
      exit 1
  done < <(grep -Pv '^(-\s\*\*[^*]+\*\*: .*\(\[#(\d+)\]\(https:\/\/github\.com\/kedacore\/(keda|charts|governance)\/(pull|issues|discussions)\/\2\)(?:\|\[#(\d+)\]\(https:\/\/github\.com\/kedacore\/(keda|charts|governance)\/(pull|issues|discussions)\/\5\)){0,}\))$' <<< "$content")

  if [ "$content" != "$sorted_content" ]; then
      echo "Error: Section: $section is not sorted correctly. Correct order:"
      echo "$sorted_content"
      exit 1
  fi
}


# Extract release sections, including "Unreleased", and check them
for version in "${versions[@]}"; do
  release_content=$(awk "/## $version/{flag=1;next}/## v[0-9\.]+/{flag=0}flag" "$filename")


  if [[ -z "$release_content" ]]; then
    echo "No content found for $version Skipping."
    continue
  fi

  echo "Checking section: $version"

  # Separate content into different sections and check sorting for each release
  extract_and_check "New" "$release_content"
  extract_and_check "Experimental" "$release_content"
  extract_and_check "Improvements" "$release_content"
  extract_and_check "Fixes" "$release_content"
  extract_and_check "Deprecations" "$release_content"
  extract_and_check "Other" "$release_content"

done
