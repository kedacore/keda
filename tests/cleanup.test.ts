import * as sh from 'shelljs';
import chalk from 'chalk';
import test from 'ava';

test.before('setup shelljs', () => {
    sh.config.silent = true;
});

test('Remove Keda', t => {
    const resources = [
        'customresourcedefinition.apiextensions.k8s.io/scaledobjects.keda.k8s.io',
        'serviceaccount/keda-operator',
        'clusterrolebinding.rbac.authorization.k8s.io/keda-operator-service-account-role-binding',
        'clusterrolebinding.rbac.authorization.k8s.io/keda:system:auth-delegator',
        'deployment.apps/keda-operator',
        'clusterrolebinding.rbac.authorization.k8s.io/custom-metrics-resource-reader',
        'service/keda-operator',
        'apiservice.apiregistration.k8s.io/v1beta1.external.metrics.k8s.io',
        'clusterrole.rbac.authorization.k8s.io/custom-metrics-resource-reader',
        'clusterrolebinding.rbac.authorization.k8s.io/keda-hpa-controller-custom-metrics'
    ];

    for (const resource of resources) {
        const result = sh.exec(`kubectl delete ${resource} --namespace keda`);
        if (result.code !== 0) {
            t.log(chalk.red(`error deleting ${resource}. ${result}`));
        }
    }

    let result = sh.exec('kubectl delete rolebinding.rbac.authorization.k8s.io/keda-auth-reader --namespace kube-system');
    if (result.code !== 0) {
        t.log(chalk.red(`error deleting rolebinding. ${result}`));
    }

    result = sh.exec('kubectl delete namespace keda');
    if (result.code !== 0) {
        t.log(chalk.red(`error deleting keda namespace. ${result}`));
    }

    t.pass();
});