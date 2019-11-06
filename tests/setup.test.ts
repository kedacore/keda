import * as sh from 'shelljs';
import * as k8s from '@kubernetes/client-node';
import chalk from 'chalk';
import test from 'ava';

const kc = new k8s.KubeConfig()
kc.loadFromDefault();

test.before('configure shelljs', () => {
    sh.config.silent = true;
});

test.serial('Verify all commands', t => {
    for (const command of ['kubectl']) {
        if (!sh.which(command)) {
            t.fail(`${command} is required for setup`);
        }
    }
    t.pass();
});

test.serial('Verify environment variables', t => {
    const cluster = kc.getCurrentCluster();
    t.truthy(cluster, 'Make sure kubectl is logged into a cluster.');
});

test.serial('Deploy Keda', t => {
    let result = sh.exec('kubectl get namespace keda');
    if (result.code !== 0 && result.stderr.indexOf('not found') !== -1) {
        t.log('creating keda namespace');
        result = sh.exec('kubectl create namespace keda');
        if (result.code !== 0) {
            t.fail('error creating keda namespace');
        }
    }

    if (sh.exec('kubectl apply -f ../deploy/crds/keda.k8s.io_scaledobjects_crd.yaml').code !== 0) {
        t.fail('error deploying keda. ' + result);
    }
    if (sh.exec('kubectl apply -f ../deploy/crds/keda.k8s.io_triggerauthentications_crd.yaml').code !== 0) {
        t.fail('error deploying keda. ' + result);
    }
    if (sh.exec('kubectl apply -f ../deploy/').code !== 0) {
        t.fail('error deploying keda. ' + result);
    }
    t.pass('Keda deployed successfully using crds and yaml');
});

test.serial('verifyKeda', t => {
    let result = sh.exec('kubectl scale deployment.apps/keda-operator --namespace keda --replicas=0');
    if (result.code !== 0) {
        t.fail(`error scaling keda to 0. ${result}`);
    }

    result = sh.exec('kubectl set image deployment.apps/keda-operator --namespace keda keda-operator=kedacore/keda:master');
    if (result.code !== 0) {
        t.fail(`error updating keda image. ${result}`);
    }

    result = sh.exec('kubectl scale deployment.apps/keda-operator --namespace keda --replicas=1');
    if (result.code !== 0) {
        t.fail(`error scaling keda to 1. ${result}`);
    }

    let success = false;
    for (let i = 0; i < 20; i++) {
        let result = sh.exec('kubectl get deployment.apps/keda-operator --namespace keda -o jsonpath="{.status.readyReplicas}"');
        const parsed = parseInt(result.stdout, 10);
        if (isNaN(parsed) || parsed != 1) {
            t.log(`Keda is not ready. sleeping`);
            sh.exec('sleep 1s');
        } else if (parsed == 1) {
            t.log('keda is running 1 pod');
            success = true;
            break;
        }
    }

    t.true(success, 'expected keda deployment to start 1 pod successfully with kedacore/keda:master');
});
