import * as sh from 'shelljs'
import test from 'ava'

const workloadIdentityNamespace = "azure-workload-identity-system"
const federatedIdentityCredentialName = "keda-e2e-federated-credential"
const AZURE_AD_OBJECT_ID = process.env['AZURE_SP_OBJECT_ID']
const RUN_WORKLOAD_IDENTITY_TESTS = process.env['AZURE_RUN_WORKLOAD_IDENTITY_TESTS']

test.before('setup shelljs', () => {
  sh.config.silent = false // TODO - Revert after PR workflow runs successfully
})

test.serial('Remove KEDA', t => {
  let result = sh.exec('(cd .. && make undeploy)')
  if (result.code !== 0) {
    t.fail('error removing keda. ' + result)
  }
  t.pass('KEDA undeployed successfully using make undeploy command')
})

test.serial('remove azure workload identity kubernetes components', t => {
  if (!RUN_WORKLOAD_IDENTITY_TESTS || RUN_WORKLOAD_IDENTITY_TESTS == 'false') {
    t.pass('nothing to clean')
    return
  }

  t.is(0,
    sh.exec(`helm uninstall workload-identity-webhook --namespace ${workloadIdentityNamespace}`).code,
    'should be able to uninstall workload identity webhook'
  )

  sh.exec(`kubectl delete ns ${workloadIdentityNamespace}`)

  let uri = `https://graph.microsoft.com/beta/applications/${AZURE_AD_OBJECT_ID}/federatedIdentityCredentials/${federatedIdentityCredentialName}`
  t.is(0,
    sh.exec(`az rest --method DELETE --uri ${uri}`).code,
    "should be able to delete federated identity credential"
  )
})
