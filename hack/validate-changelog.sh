#!/bin/bash

SCRIPT_ROOT=$(dirname "${BASH_SOURCE[0]}")/..

# Define filename
filename="$SCRIPT_ROOT/CHANGELOG.md"

# Check if file exists
if [[ ! -f "$filename" ]]; then
    echo "Error: $filename does not exist."
    exit 1
fi

# Read content between "## Unreleased" and "## v" into variable
unreleased=$(awk '/## Unreleased/{flag=1;next}/## v[0-9\.]+/{flag=0}flag' "$filename")

# Check if "Unreleased" section exists
if [[ -z "$unreleased" ]]; then
    echo "Error: No 'Unreleased' section found in $filename."
    exit 1
fi

# Define a function to extract and sort sections
function extract_and_check() {
  local section=$1
  local content=$(awk "/### $section/{flag=1;next}/### /{flag=0}flag" <<< "$unreleased" | grep '^- \*\*')

  # Skip if content does not exist
  if [[ -z "$content" ]]; then
    return
  fi

  # Separate and sort the **General**: lines
  local sorted_general_lines=$(echo "$content" | grep '^- \*\*General\*\*:' | sort)

  # Sort the remaining lines
  local sorted_content=$(echo "$content" | grep -v '^- \*\*General\*\*:' | sort)

  # Concatenate the sorted **General**: lines at the top of the sorted_content
  sorted_content=$(printf "%s\n%s" "$sorted_general_lines" "$sorted_content")

  # Check pattern and throw error if wrong pattern found
  while IFS= read -r line; do
      echo "Error: Wrong pattern found in $section section, line: $line"
      exit 1
  done < <(grep -Pv '^(-\s\*\*[^*]+\*\*: .*\(\[#(\d+)\]\(https:\/\/github\.com\/kedacore\/keda\/(pull|issues)\/\2\)\))$' <<< "$content")

  if [ "$content" != "$sorted_content" ]; then
      echo "Error: The $section section is not sorted correctly. Correct order:"
      echo "$sorted_content"
      exit 1
  fi
}

# Separate content into different sections and check sorting
extract_and_check "New"
extract_and_check "Improvements"
extract_and_check "Fixes"
extract_and_check "Deprecations"
extract_and_check "Other"
