import * as async from 'async';
import * as azure from 'azure-storage';
import * as fs from 'fs';
import * as sh from 'shelljs';
import * as tmp from 'tmp';
import test from 'ava';
import { TLSSocket } from 'tls';

const defaultNamespace = 'azure-job-queue-test';
const connectionString = process.env['TEST_STORAGE_CONNECTION_STRING'];

test.before(t => {
    if (!connectionString) {
        t.fail('TEST_STORAGE_CONNECTION_STRING environment variable is required for queue tests');
    }

    sh.config.silent = true;
    const base64ConStr = Buffer.from(connectionString).toString('base64');
    const tmpFile = tmp.fileSync();
    fs.writeFileSync(tmpFile.name, deployYaml.replace('{{CONNECTION_STRING_BASE64}}', base64ConStr));
    t.log(sh.exec(`kubectl create namespace ${defaultNamespace}`));
    t.log(sh.exec(`kubectl apply -f ${tmpFile.name} --namespace ${defaultNamespace}`));
    t.is(0, sh.exec(`kubectl apply -f ${tmpFile.name} --namespace ${defaultNamespace}`).code, 'creating a deployment should work.');
});

test.serial('Jobs should be 0 on start', t=> {
    const jobCount = parseInt(sh.exec(`kubectl get jobs --namespace ${defaultNamespace} -o jsonpath="{.items[?(@.status.succeeded==0)].metadata.name}" | wc -w`).stdout, 10);
    t.is(jobCount, 0, 'job count should start out as 0');
});

test.serial.cb('Jobs should scale to 4 with 4 messages on the queue then back to 0', t => {
    // add 10,000 messages
    const queueSvc = azure.createQueueService(connectionString);
    queueSvc.messageEncoder = new azure.QueueMessageEncoder.TextBase64QueueMessageEncoder();
    queueSvc.createQueueIfNotExists('test-queue', err => {
        t.falsy(err, 'unable to create queue');
        async.mapLimit(Array(100).keys(), 200, (n, cb) => queueSvc.createMessage('test-queue', `test ${n}`, cb), () => {
            let jobCount = 0;
            for (let i = 0; i < 10 && jobCount !== 4; i++) {
                jobCount = parseInt(sh.exec(`kubectl get jobs --namespace ${defaultNamespace} -o jsonpath="{.items[?(@.status.active==1)].metadata.name}" | wc -w`).stdout, 10);
                t.log(jobCount);
                if (jobCount !== 4) {
                    sh.exec('sleep 1s');
                }
            }

            t.is(4, jobCount, 'Job count should be 4 after 10 seconds');

            for (let i = 0; i < 50 && jobCount !== 0; i++) {
                jobCount = parseInt(sh.exec(`kubectl get jobs --namespace ${defaultNamespace} -o jsonpath="{.items[?(@.status.active==1)].metadata.name}" | wc -w`).stdout,10);
                t.log(jobCount);
                if (jobCount !== 0) {
                    sh.exec('sleep 5s');
                }
            }

            t.is(0, jobCount, 'Job count should be 0 after 3 minutes')
            t.end();
        });
    });
});

test.after.always.cb('clean up azure-queue deployment', t => {
    const resources = [
        'secret/test-secrets',
        'scaledobject.keda.k8s.io/test-scaledobject-jobs',
    ];
        //'keda-operator'

    for (const resource of resources) {
        sh.exec(`kubectl delete ${resource} --namespace ${defaultNamespace}`);
    }
    sh.exec(`kubectl delete namespace ${defaultNamespace}`);

    // delete test queue
    const queueSvc = azure.createQueueService(connectionString);
    queueSvc.deleteQueueIfExists('test-queue', err => {
        t.falsy(err, 'should delete test queue successfully');
        t.end();
    });
});

const deployYaml = `apiVersion: v1
kind: Secret
metadata:
  name: test-secrets
data:
  AzureWebJobsStorage: {{CONNECTION_STRING_BASE64}}
---
apiVersion: keda.k8s.io/v1alpha1
kind: ScaledObject
metadata:
  name: test-scaledobject-jobs
  namespace: azure-job-queue-test
spec:
  scaleType: job
  pollingInterval: 10 
  maxReplicaCount: 4
  cooldownPeriod: 10
  parallelism: 1
  completions: 1
  activeDeadline: 60
  backoffLimit: 6
  consumerSpec:
    containers:
    - name: consumer-job
      image: sgricci/queue-consumer:latest
      metadata:
        namespace: azure-job-queue-test
        name: consumer-job
      env:
      - name: test
        value: test
      - name: TEST_STORAGE_CONNECTION_STRING
        valueFrom:
          secretKeyRef:
            name: test-secrets
            key: AzureWebJobsStorage
  triggers:
  - type: azure-queue
    metadata:
      queueName: test-queue
      connection: AzureWebJobsStorage`
