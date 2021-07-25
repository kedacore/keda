import * as sh from "shelljs";

export function waitForRollout(type: 'deployment' | 'statefulset', name: string, namespace: string): number {
    return sh.exec(`kubectl rollout status ${type}/${name} -n ${namespace}`).code
}
