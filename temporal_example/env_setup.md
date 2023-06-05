# Environment setup instructions

## Temporal Go App example

### Requirements

- pulumi
- az cli
- golang

### Deployment

- Login to AZ CLI

- Init dev env

```bash
pulumi stack init dev
```

- Deploy dev env

```bash
pulumi up
```

### Launching Temporal App Workflow

After deployment, endpoints will be printed out.
Use these endpoint to launch and view workflows.

```bash
Outputs:
    starterEndpoint: output<string>
    webEndpoint    : output<string>
```    


## KEDA Scaler 

### Requirements

- kubectl
- docker
- golang

### Deployment

- Open provided Dev Container

- AZ Login

- Get k8s credentials

```bash
az aks get-credentials --resource-group <rg-name> --name <rg-name>-aks
```

- Login to Docker Registry

```bash
sudo docker login <rg-name>.azurecr.io 
```

- Build KEDA and publish image

```bash
sudo IMAGE_REGISTRY=<rg-name>.azurecr.io IMAGE_REPO=<rg-name> make publish
```

- Deploy KEDA resources

```bash
IMAGE_REGISTRY=<rg-name>.azurecr.io IMAGE_REPO=<rg-name> make publish
```

### Deploy Temporal KEDA scaleable object

- Edit values in: temporal_example/temporal_scaledObject.yml

```yml
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: temporal-scaledobject
  namespace: temporal
spec:
  scaleTargetRef:
    name: <Workflow-app-pod-id>
  pollingInterval: 20
  cooldownPeriod:  200
  minReplicaCount: 1
  maxReplicaCount: 50
  triggers:
  - type: temporal
    metadata:
      address: "<temporal internal ip>:7233"
      threshold:           '10'
      #activationThreshold: '50'
```

- Deploy Scalable Object

```bash
kubectl apply -f temporal_example/temporal_scaledObject.yml
```

### Test scaler

Using parallel:

```bash
seq 1000 |  parallel -n0 -j2 "curl http://<endpoint>:8080/async?name=scaledemo"
```
