# Useful Commands

## Get K8s objects in namespace

### Temporal

```bash
kubectl get all --namespace temporal
```

### Keda

```bash
kubectl get all --namespace temporal
```

## Deploy the Scaled Object

```bash
kubectl apply -f temporal_example/temporal_scaledObject.yml
```

## Deploy the custom Keda (with Temporal Scaler)

### Build and publish

```bash
sudo IMAGE_REGISTRY=<rg-name>.azurecr.io IMAGE_REPO=<rg-name> make publish
```

### Deploy to K8s from Container Registry

```bash
IMAGE_REGISTRY=<rg-name>.azurecr.io IMAGE_REPO=<rg-name> make deploy
```

## Undeploy from K8s

```bash
IMAGE_REGISTRY=<rg-name>.azurecr.io IMAGE_REPO=<rg-name> make undeploy
```

## Get KEDA operator logs

```bash
kubectl logs -l app=keda-operator -n keda -f
```

## Run start worker calls in parallel

```bash
seq 1000 |  parallel -n0 -j2 "curl http://<endpoint>:8080/async?name=scaletest1"
```