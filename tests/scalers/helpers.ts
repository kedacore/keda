import * as sh from "shelljs";
import * as tmp from 'tmp';
import * as fs from 'fs';

export function waitForRollout(type: 'deployment' | 'statefulset', name: string, namespace: string, timeoutSeconds = 180): number {
    return sh.exec(`kubectl rollout status ${type}/${name} -n ${namespace} --timeout ${timeoutSeconds}s`).code
}

export function sleep(duration: number) {
    return new Promise(resolve => setTimeout(resolve, duration));
}

export async function waitForDeploymentReplicaCount(target: number, name: string, namespace: string, iterations = 10, interval = 3000): Promise<boolean> {
    for (let i = 0; i < iterations; i++) {
        let replicaCountStr = sh.exec(`kubectl get deployment.apps/${name} --namespace ${namespace} -o jsonpath="{.spec.replicas}"`).stdout
        try {
            const replicaCount = parseInt(replicaCountStr, 10)
            if (replicaCount === target) {
                return true
            }
        } catch { }

        await sleep(interval)
    }
    return false
}

export async function createNamespace(namespace: string) {
    const namespaceFile = tmp.fileSync()
    fs.writeFileSync(namespaceFile.name, namespaceTemplate.replace('{{NAMESPACE}}', namespace))
    sh.exec(`kubectl apply -f ${namespaceFile.name}`)
}

export function createYamlFile(yaml: string) {
    const tmpFile = tmp.fileSync()
    fs.writeFileSync(tmpFile.name, yaml)
    return tmpFile.name
}

const namespaceTemplate = `apiVersion: v1
kind: Namespace
metadata:
  labels:
    type: e2e
  name: {{NAMESPACE}}`
