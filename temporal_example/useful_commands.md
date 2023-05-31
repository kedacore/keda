##
```
kubectl get all --namespace temporal
```

```
kubectl apply -f temporal_example/temporal_scaledObject.yml
```

```
IMAGE_REGISTRY=thnuq3k.azurecr.io IMAGE_REPO=thnuq3k make deploy
```

```
IMAGE_REGISTRY=thnuq3k.azurecr.io IMAGE_REPO=thnuq3k make undeploy
```

```
sudo IMAGE_REGISTRY=thnuq3k.azurecr.io IMAGE_REPO=thnuq3k make publish
```

```
kubectl logs -l app=keda-operator -n keda -f
```

```
seq 1000 |  parallel -n0 -j2 "curl http://20.81.100.221:8080/async?name=scaletest1"
```