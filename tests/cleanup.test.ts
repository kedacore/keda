import * as sh from 'shelljs'
import test from 'ava'

test.before('setup shelljs', () => {
  sh.config.silent = true
})

test('Remove KEDA', t => {
  let result = sh.exec('(cd .. && make undeploy)')
  if (result.code !== 0) {
    t.fail('error removing keda. ' + result)
  }
  t.pass('KEDA undeployed successfully using make undeploy command')
})
