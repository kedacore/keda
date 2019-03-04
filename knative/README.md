# Knative PodAutoscaler for Kore

This is pretty barebones at the moment, but just to jot down a few
notes until I work out better instructions.

Create a secret called `koredemo` that contains your Azure Storage
account connection string using a command like:

```bash
oc create secret generic koredemo --from-literal="connection=<your connection string here>"
```

Then, start the Kore Knative PodAutoscaler, telling it to watch all
namespaces for Knative PodAutoscalers:

```bash
WATCH_NAMESPACE="" go run knative/cmd/manager/main.go
```

Create a queue called `demo` in your Azure Storage account. Or, use a
different queue name and modify the
`knative/deploy/knative_service.yaml` to use the desired queue name.

Then deploy a Knative Service with annotations to tell it to use the
Kore autoscaler as well as the secret name and key with the connection
string and the queue name in use:

```bash
kubectl deploy knative/deploy/knative_service.yaml
```

If everything worked, you'll see a Kore `ScaledObject` instance when
the Knative Service gets deployed. If you also have Kore itself
running, you'll see the pods for the Knative Service scale to and from
0 as messages are added and deleted from the Azure Storage Queue using
the web console.

The sample Knative Service here does not actually consume from the
queue. It's just used to test the scaling integration. We can and
should replace that with an Azure Function that consumes from the
queue.
