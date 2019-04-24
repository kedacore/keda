#! /bin/bash

set -e

echo "Add helm repo"
helm repo add projectkore https://projectkore.azurecr.io/helm/v1/repo \
    --username b514b60c-68cc-4f12-b361-3858878b2479 \
    --password '4jX5vkPTSrUQ96UBbU/B7CQrBoJwT62WSs5WfZtFbB8='

echo "Update helm repos"
helm repo update

echo "Create Tiller's service account"
cat <<END_RBAC_CONFIG | kubectl create -f -
apiVersion: v1
kind: ServiceAccount
metadata:
  name: tiller
  namespace: kube-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: tiller
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin
subjects:
  - kind: ServiceAccount
    name: tiller
    namespace: kube-system
END_RBAC_CONFIG

echo "Init helm"
helm init --service-account tiller --wait
helm install projectkore/kore-edge --name kore-test-release --devel --set logLevel=debug

