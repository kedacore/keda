+++
title = "Deploying KEDA"
date = "2017-10-05"
fragment = "content"
weight = 100
+++

## Deploying with a Helm chart

### Add Helm repo
```cli
helm repo add kedacore https://kedacore.azureedge.net/helm
```

### Update Helm repo
```cli
helm repo update
```

### Install keda-edge chart
```cli
helm install kedacore/keda-edge --devel --set logLevel=debug --namespace keda --name keda
```

### Install keda-edge chart with ARM image
```cli
helm install kedacore/keda-edge --devel --set logLevel=debug --namespace keda --name keda --set image.tag=arm
```

## Deploying with the [Azure Functions Core Tools](https://github.com/Azure/azure-functions-core-tools)
```
func kubernetes install --namespace keda
```

## Deploying using the deploy yaml
If you want to try KEDA on minikube or a different Kubernetes deployment without using Helm, you can deploy CRD and yamls under the `/deploy` directory on our GitHub repo.
```
kubectl apply -f deploy/crds/keda.k8s.io_scaledobjects_crd.yaml
kubectl apply -f deploy/crds/keda.k8s.io_triggerauthentications_crd.yaml
kubectl apply -f deploy/
```