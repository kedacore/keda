#! /bin/bash

DIR="$(dirname $0)"

NAMESPACE=test-queue
DEPLOYMENT_NAME=test-deployment
QUEUE_NAME=queue-name

echo "create namespace $NAMESPACE"
kubectl create namespace $NAMESPACE

echo "deploy functions"
CONNECTION_STRING_BASE64=$(echo -n "$TEST_STORAGE_CONNECTION" | base64 -w 0)
sed -i "s/CONNECTION_STRING_BASE64/$CONNECTION_STRING_BASE64/g" $DIR/deploy.yaml

kubectl apply -f $DIR/deploy.yaml --namespace $NAMESPACE

echo "sleep for 20 seconds then confirm that the deployment is still at 0."
sleep 20

replica_count=$(kubectl get deployment $DEPLOYMENT_NAME --namespace $NAMESPACE --output jsonpath='{.spec.replicas}')
echo "Rplica count is $replica_count"

if [ $replica_count != 0 ]; then
    >&2 echo "Replica count should be 0 on the start of the test"
    kubectl get deployment $DEPLOYMENT_NAME --namespace $NAMESPACE
    exit 1
fi

echo "create messages on the queue \"$QUEUE_NAME\""
go run $DIR/zero_to_1_azure_queue.go create $TEST_STORAGE_CONNECTION $QUEUE_NAME

for i in {1..30}
do
    echo "sleeping 5 seconds"
    sleep 5
    replica_count=$(kubectl get deployment $DEPLOYMENT_NAME --namespace $NAMESPACE --output jsonpath='{.spec.replicas}')
    echo "Replica count is $replica_count"
    if [ $replica_count == 1 ]; then
        break
    fi
done

if [ $replica_count != 1 ]; then
    >&2 echo "Replica count should be == 1"
    kubectl get deployment $DEPLOYMENT_NAME --namespace $NAMESPACE
    exit 1
fi

for i in {1..10}
do
    queue_length=$(go run $DIR/zero_to_1_azure_queue.go get-length $TEST_STORAGE_CONNECTION $QUEUE_NAME)
    echo "Queue $QUEUE_NAME length is $queue_length"
    if [ $queue_length != 0 ]; then
        echo "sleeping for 60 seconds"
        sleep 60
    else
        break
    fi
done

if [ $queue_length != 0 ]; then
    echo "Queue length is still not zero"
    exit 1
fi

echo "sleeping 10 seconds"
sleep 10

replica_count=$(kubectl get deployment $DEPLOYMENT_NAME --namespace $NAMESPACE --output jsonpath='{.spec.replicas}')
echo "Replica count is $replica_count"

if [ $replica_count != 0 ]; then
    >&2 echo "Replica count should be == 0"
    kubectl get deployment $DEPLOYMENT_NAME --namespace $NAMESPACE
    exit 1
fi

echo "Test succeeded"
kubectl delete namespace $NAMESPACE

exit 0
