#! /bin/bash

set -e # uo pipefail

function assert_env_var {
    value=$(eval "echo \$$1")
    if [ -z "$value" ]; then
        echo -e "Error: env var $1 is not defined"
        exit 1
    fi
}

function assert_command_exists {
    if ! command -v $1 > /dev/null; then
        echo -e "Error: command $1 doesn't exist"
        exit 1
    fi
}

required_env_vars=(AZURE_SP_ID AZURE_SP_KEY AZURE_SP_TENANT AZURE_SUBSCRIPTION AZURE_RESOURCE_GROUP AKS_NAME TEST_STORAGE_CONNECTION)
for var_name in "${required_env_vars[@]}"
do
    assert_env_var $var_name
done


required_commands=(az kubectl helm)
for var_name in "${required_commands[@]}"
do
    assert_command_exists $var_name
done