# TriggerAuthentication specification

This specification describes the `TriggerAuthentication` custom resource definition which is used to define how KEDA should authenticate to a given trigger.

This allows you to define once how to authenticate and use it for multiple triggers across different teams, without them knowing where the secrets are.

```yaml
apiVersion: keda.k8s.io/v1alpha1
kind: TriggerAuthentication
metadata:
metadata:
  name: keda-trigger-auth-azure-queue-secret
  namespace: keda
spec:
  podIdentity:
      provider: none | azure | gcp | spiffe # Optional. Default: none
  secretTargetRef: # Optional.
  - parameter: connectionString # Required.
    name: my-keda-secret-entity # Required.
    key: azure-storage-connectionstring # Required.
  env: # Optional.
  - parameter: region # Required.
    name: my-env-var # Required.
    containerName: my-container # Optional. Default: scaleTargetRef.containerName of ScaledObject
```

In order to determine what set of parameters you need to define we recommend reading the specification for the trigger type that you need.

Based on the requirements you can mix and match the authentication providers in order to configure all required parameters.

### Environment variable(s)

You can pull information via one or more environment variables by providing the `name` of the variable for a given `containerName`.

```yaml
env: # Optional.
  - parameter: region # Required.
    name: my-env-var # Required.
    containerName: my-container # Optional. Default: scaleTargetRef.containerName of ScaledObject
```

**Assumptions:** `containerName` is in the same deployment as the configured `scaleTargetRef.deploymentName` in the ScaledObject, unless specified otherwise.

### Secret(s)

You can pull one or more secrets into the trigger by defining the `name` of the Kubernetes Secret and the `key` to use.

```yaml
secretTargetRef: # Optional.
  - parameter: connectionString # Required.
    name: my-keda-secret-entity # Required.
    key: azure-storage-connectionstring # Required.
```

**Assumptions:** `namespace` is in the same deployment as the configured `scaleTargetRef.deploymentName` in the ScaledObject, unless specified otherwise.

### Pod Authentication Providers

Several service providers allow you to assign an identity to a pod. By using that identity, you can defer authentication to the pod & the service provider, rather than configuring secrets.

Currently we support the following:

```yaml
podIdentity:
  provider: none | azure # Optional. Default: false
```

#### Azure Pod Identity

Azure Pod Identity is an implementation of [Azure AD Pod Identity](https://github.com/Azure/aad-pod-identity) which let's you bind an [Azure Managed Identity](https://docs.microsoft.com/en-us/azure/active-directory/managed-identities-azure-resources/) to a Pod in a Kubernetes cluster as delegated access.

You can tell KEDA to use Azure AD Pod Identity via `podIdentity.provider`.

- https://docs.microsoft.com/en-us/azure/active-directory/managed-identities-azure-resources/

```yaml
podIdentity:
  provider: azure # Optional. Default: false
```
