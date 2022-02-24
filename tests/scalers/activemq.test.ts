import * as fs from 'fs'
import * as sh from 'shelljs'
import * as tmp from 'tmp'
import test from 'ava'
import {createNamespace, waitForRollout} from './helpers'

const activeMQNamespace = 'activemq-test'
const activemqConf = '/opt/apache-activemq-5.16.3/conf'
const activemqHome = '/opt/apache-activemq-5.16.3'
const activeMQPath = 'bin/activemq'
const activeMQUsername = 'admin'
const activeMQPassword = 'admin'
const destinationName = 'testQ'
const nginxDeploymentName = 'nginx-deployment'

test.before(t => {
	// install ActiveMQ
  createNamespace(activeMQNamespace)
	const activeMQTmpFile = tmp.fileSync()
	fs.writeFileSync(activeMQTmpFile.name, activeMQDeployYaml)

	t.is(0, sh.exec(`kubectl apply --namespace ${activeMQNamespace} -f ${activeMQTmpFile.name}`).code, 'creating ActiveMQ deployment should work.')
	t.is(0, waitForRollout('deployment', "activemq", activeMQNamespace))

	const activeMQPod = sh.exec(`kubectl get pods --selector=app=activemq-app -n ${activeMQNamespace} -o jsonpath='{.items[0].metadata.name'}`).stdout

        // ActiveMQ ready check
	let activeMQReady
	for (let i = 0; i < 30; i++) {
		activeMQReady = sh.exec(`kubectl exec -n ${activeMQNamespace} ${activeMQPod} -- curl -u ${activeMQUsername}:${activeMQPassword} -s http://localhost:8161/api/jolokia/exec/org.apache.activemq:type=Broker,brokerName=localhost,service=Health/healthStatus | sed -e 's/[{}]/''/g' | awk -v RS=',"' -F: '/^status/ {print $2}'`)
		if (activeMQReady != 200) {
			sh.exec('sleep 5s')
		}
		else {
			break
		}
	}

	// deploy Nginx, scaledobject etc.
	const nginxTmpFile = tmp.fileSync()
	fs.writeFileSync(nginxTmpFile.name, nginxDeployYaml)

	t.is(0, sh.exec(`kubectl apply --namespace ${activeMQNamespace} -f ${nginxTmpFile.name}`).code, 'creating Nginx deployment should work.')
        t.is(0, waitForRollout('deployment', "nginx-deployment", activeMQNamespace))
})

test.serial('Deployment should have 0 replicas on start', t => {
	const replicaCount = sh.exec(`kubectl get deploy/${nginxDeploymentName} --namespace ${activeMQNamespace} -o jsonpath="{.spec.replicas}"`).stdout
	t.is(replicaCount, '0', 'replica count should start out as 0')
})

test.serial('Deployment should scale to 5 (the max) with 1000 messages on the queue then back to 0', t => {
	const activeMQPod = sh.exec(`kubectl get pods --selector=app=activemq-app -n ${activeMQNamespace} -o jsonpath='{.items[0].metadata.name'}`).stdout

	// produce 1000 messages to ActiveMQ
	t.is(
		0,
		sh.exec(`kubectl exec -n ${activeMQNamespace} ${activeMQPod} -- ${activeMQPath} producer --destination ${destinationName} --messageCount 1000`).code,
		'produce 1000 message to the ActiveMQ queue'
	)

	let replicaCount = '0'
	const maxReplicaCount = '5'

	for (let i = 0; i < 30 && replicaCount !== maxReplicaCount; i++) {
		replicaCount = sh.exec(`kubectl get deploy/${nginxDeploymentName} --namespace ${activeMQNamespace} -o jsonpath="{.spec.replicas}"`).stdout
		if (replicaCount !== maxReplicaCount) {
			sh.exec('sleep 2s')
		}
	}
	t.is(maxReplicaCount, replicaCount, `Replica count should be ${maxReplicaCount} after 60 seconds`)
	sh.exec('sleep 30s')

	// consume all messages from ActiveMQ
	t.is(
		0,
		sh.exec(`kubectl exec -n ${activeMQNamespace} ${activeMQPod} -- ${activeMQPath} consumer --destination ${destinationName} --messageCount 1000`).code,
		'consume all messages'
	)

	for (let i = 0; i < 50 && replicaCount !== '0'; i++) {
		replicaCount = sh.exec(
			`kubectl get deploy/${nginxDeploymentName} --namespace ${activeMQNamespace} -o jsonpath="{.spec.replicas}"`).stdout
		if (replicaCount !== '0') {
			sh.exec('sleep 5s')
		}
	}
	t.is('0', replicaCount, 'Replica count should be 0 after 3 minutes')

})

test.after.always((t) => {
     t.is(0, sh.exec(`kubectl delete namespace ${activeMQNamespace}`).code, 'Should delete ActiveMQ namespace')
})

const activeMQDeployYaml = `
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: activemq-app
  name: activemq
spec:
  replicas: 1
  revisionHistoryLimit: 10
  selector:
    matchLabels:
      app: activemq-app
  template:
    metadata:
      labels:
        app: activemq-app
    spec:
      containers:
      - image: symptoma/activemq:5.16.3
        imagePullPolicy: IfNotPresent
        name: activemq
        ports:
        - containerPort: 61616
          name: jmx
          protocol: TCP
        - containerPort: 8161
          name: ui
          protocol: TCP
        - containerPort: 61616
          name: openwire
          protocol: TCP
        - containerPort: 5672
          name: amqp
          protocol: TCP
        - containerPort: 61613
          name: stomp
          protocol: TCP
        - containerPort: 1883
          name: mqtt
          protocol: TCP
        resources:
        volumeMounts:
        - name: activemq-config
          mountPath: /opt/apache-activemq-5.16.3/webapps/api/WEB-INF/classes/jolokia-access.xml
          subPath: jolokia-access.xml
        - name: remote-access-cm
          mountPath: /opt/apache-activemq-5.16.3/conf/jetty.xml
          subPath: jetty.xml
      volumes:
      - name: activemq-config
        configMap:
          name: activemq-config
          items:
          - key: jolokia-access.xml
            path: jolokia-access.xml
      - name: remote-access-cm
        configMap:
          name: remote-access-cm
          items:
          - key: jetty.xml
            path: jetty.xml
---
apiVersion: v1
kind: Service
metadata:
  name: activemq
spec:
  type: ClusterIP
  selector:
    app: activemq-app
  ports:
  - name: dashboard
    port: 8161
    targetPort: 8161
    protocol: TCP
  - name: openwire
    port: 61616
    targetPort: 61616
    protocol: TCP
  - name: amqp
    port: 5672
    targetPort: 5672
    protocol: TCP
  - name: stomp
    port: 61613
    targetPort: 61613
    protocol: TCP
  - name: mqtt
    port: 1883
    targetPort: 1883
    protocol: TCP
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: activemq-config
data:
  jolokia-access.xml: |
    <?xml version="1.0" encoding="UTF-8"?>
    <restrict>
      <remote>
        <host>0.0.0.0/0</host>
      </remote>

      <deny>
        <mbean>
          <name>com.sun.management:type=DiagnosticCommand</name>
          <attribute>*</attribute>
          <operation>*</operation>
        </mbean>
        <mbean>
          <name>com.sun.management:type=HotSpotDiagnostic</name>
          <attribute>*</attribute>
          <operation>*</operation>
        </mbean>
      </deny>
    </restrict>
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: remote-access-cm
data:
  jetty.xml: |
    <!--
        Licensed to the Apache Software Foundation (ASF) under one or more contributor
        license agreements. See the NOTICE file distributed with this work for additional
        information regarding copyright ownership. The ASF licenses this file to You under
        the Apache License, Version 2.0 (the "License"); you may not use this file except in
        compliance with the License. You may obtain a copy of the License at
        http://www.apache.org/licenses/LICENSE-2.0 Unless required by applicable law or
        agreed to in writing, software distributed under the License is distributed on an
        "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
        implied. See the License for the specific language governing permissions and
        limitations under the License.
    -->
    <!--
        An embedded servlet engine for serving up the Admin consoles, REST and Ajax APIs and
        some demos Include this file in your configuration to enable ActiveMQ web components
        e.g. <import resource="jetty.xml"/>
    -->
    <beans xmlns="http://www.springframework.org/schema/beans" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
    xsi:schemaLocation="http://www.springframework.org/schema/beans http://www.springframework.org/schema/beans/spring-beans.xsd">

    <bean id="httpConfig" class="org.eclipse.jetty.server.HttpConfiguration">
        <property name="sendServerVersion" value="false"/>
    </bean>

    <bean id="securityLoginService" class="org.eclipse.jetty.security.HashLoginService">
        <property name="name" value="ActiveMQRealm" />
        <property name="config" value="${activemqConf}/jetty-realm.properties" />
    </bean>

    <bean id="securityConstraint" class="org.eclipse.jetty.util.security.Constraint">
        <property name="name" value="BASIC" />
        <property name="roles" value="user,admin" />
        <!-- set authenticate=false to disable login -->
        <property name="authenticate" value="true" />
    </bean>
    <bean id="adminSecurityConstraint" class="org.eclipse.jetty.util.security.Constraint">
        <property name="name" value="BASIC" />
        <property name="roles" value="admin" />
         <!-- set authenticate=false to disable login -->
        <property name="authenticate" value="true" />
    </bean>
    <bean id="securityConstraintMapping" class="org.eclipse.jetty.security.ConstraintMapping">
        <property name="constraint" ref="securityConstraint" />
        <property name="pathSpec" value="/*,/api/*,/admin/*,*.jsp" />
    </bean>
    <bean id="adminSecurityConstraintMapping" class="org.eclipse.jetty.security.ConstraintMapping">
        <property name="constraint" ref="adminSecurityConstraint" />
        <property name="pathSpec" value="*.action" />
    </bean>

    <bean id="rewriteHandler" class="org.eclipse.jetty.rewrite.handler.RewriteHandler">
        <property name="rules">
            <list>
                <bean id="header" class="org.eclipse.jetty.rewrite.handler.HeaderPatternRule">
                  <property name="pattern" value="*"/>
                  <property name="name" value="X-FRAME-OPTIONS"/>
                  <property name="value" value="SAMEORIGIN"/>
                </bean>
                <bean id="header" class="org.eclipse.jetty.rewrite.handler.HeaderPatternRule">
                  <property name="pattern" value="*"/>
                  <property name="name" value="X-XSS-Protection"/>
                  <property name="value" value="1; mode=block"/>
                </bean>
                <bean id="header" class="org.eclipse.jetty.rewrite.handler.HeaderPatternRule">
                  <property name="pattern" value="*"/>
                  <property name="name" value="X-Content-Type-Options"/>
                  <property name="value" value="nosniff"/>
                </bean>
            </list>
        </property>
    </bean>

    <bean id="secHandlerCollection" class="org.eclipse.jetty.server.handler.HandlerCollection">
        <property name="handlers">
            <list>
            <ref bean="rewriteHandler"/>
                <bean class="org.eclipse.jetty.webapp.WebAppContext">
                    <property name="contextPath" value="/admin" />
                    <property name="resourceBase" value="${activemqHome}/webapps/admin" />
                    <property name="logUrlOnStart" value="true" />
                </bean>
                <bean class="org.eclipse.jetty.webapp.WebAppContext">
                    <property name="contextPath" value="/api" />
                    <property name="resourceBase" value="${activemqHome}/webapps/api" />
                    <property name="logUrlOnStart" value="true" />
                </bean>
                <bean class="org.eclipse.jetty.server.handler.ResourceHandler">
                    <property name="directoriesListed" value="false" />
                    <property name="welcomeFiles">
                        <list>
                            <value>index.html</value>
                        </list>
                    </property>
                    <property name="resourceBase" value="${activemqHome}/webapps/" />
                </bean>
                <bean id="defaultHandler" class="org.eclipse.jetty.server.handler.DefaultHandler">
                    <property name="serveIcon" value="false" />
                </bean>
            </list>
        </property>
    </bean>
    <bean id="securityHandler" class="org.eclipse.jetty.security.ConstraintSecurityHandler">
        <property name="loginService" ref="securityLoginService" />
        <property name="authenticator">
            <bean class="org.eclipse.jetty.security.authentication.BasicAuthenticator" />
        </property>
        <property name="constraintMappings">
            <list>
                <ref bean="adminSecurityConstraintMapping" />
                <ref bean="securityConstraintMapping" />
            </list>
        </property>
        <property name="handler" ref="secHandlerCollection" />
    </bean>

    <bean id="contexts" class="org.eclipse.jetty.server.handler.ContextHandlerCollection">
    </bean>

    <bean id="jettyPort" class="org.apache.activemq.web.WebConsolePort" init-method="start">
             <!-- the default port number for the web console -->
        <property name="host" value="0.0.0.0"/>
        <property name="port" value="8161"/>
    </bean>

    <bean id="Server" depends-on="jettyPort" class="org.eclipse.jetty.server.Server"
        destroy-method="stop">

        <property name="handler">
            <bean id="handlers" class="org.eclipse.jetty.server.handler.HandlerCollection">
                <property name="handlers">
                    <list>
                        <ref bean="contexts" />
                        <ref bean="securityHandler" />
                    </list>
                </property>
            </bean>
        </property>

    </bean>

    <bean id="invokeConnectors" class="org.springframework.beans.factory.config.MethodInvokingFactoryBean">
    <property name="targetObject" ref="Server" />
    <property name="targetMethod" value="setConnectors" />
    <property name="arguments">
    <list>
               <bean id="Connector" class="org.eclipse.jetty.server.ServerConnector">
                   <constructor-arg ref="Server"/>
                   <constructor-arg>
                       <list>
                           <bean id="httpConnectionFactory" class="org.eclipse.jetty.server.HttpConnectionFactory">
                               <constructor-arg ref="httpConfig"/>
                           </bean>
                       </list>
                   </constructor-arg>
                   <!-- see the jettyPort bean -->
                   <property name="host" value="#{systemProperties['jetty.host']}" />
                   <property name="port" value="#{systemProperties['jetty.port']}" />
               </bean>
                <!--
                    Enable this connector if you wish to use https with web console
                -->
                <!-- bean id="SecureConnector" class="org.eclipse.jetty.server.ServerConnector">
                    <constructor-arg ref="Server" />
                    <constructor-arg>
                        <bean id="handlers" class="org.eclipse.jetty.util.ssl.SslContextFactory">

                            <property name="keyStorePath" value="${activemqConf}/broker.ks" />
                            <property name="keyStorePassword" value="password" />
                        </bean>
                    </constructor-arg>
                    <property name="port" value="8162" />
                </bean -->
            </list>
    </property>
    </bean>

    <bean id="configureJetty" class="org.springframework.beans.factory.config.MethodInvokingFactoryBean">
        <property name="staticMethod" value="org.apache.activemq.web.config.JspConfigurer.configureJetty" />
        <property name="arguments">
            <list>
                <ref bean="Server" />
                <ref bean="secHandlerCollection" />
            </list>
        </property>
    </bean>

    <bean id="invokeStart" class="org.springframework.beans.factory.config.MethodInvokingFactoryBean"
    depends-on="configureJetty, invokeConnectors">
    <property name="targetObject" ref="Server" />
    <property name="targetMethod" value="start" />
    </bean>
    </beans>
`
const nginxDeployYaml = `
apiVersion: v1
kind: Secret
metadata:
  name: activemq-secret
type: Opaque
data:
  activemq-password: YWRtaW4=
  activemq-username: YWRtaW4=
---
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: trigger-auth-activemq
spec:
  secretTargetRef:
    - parameter: username
      name: activemq-secret
      key: activemq-username
    - parameter: password
      name: activemq-secret
      key: activemq-password
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: nginx
  name: ${nginxDeploymentName}
spec:
  replicas: 0
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - image: nginx
        name: nginx
        ports:
        - containerPort: 80
---
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: activemq-scaledobject
  labels:
    deploymentName: ${nginxDeploymentName}
spec:
  scaleTargetRef:
    name: ${nginxDeploymentName}
  pollingInterval: 5
  cooldownPeriod:  5
  minReplicaCount: 0
  maxReplicaCount: 5
  triggers:
    - type: activemq
      metadata:
        managementEndpoint: "activemq.${activeMQNamespace}:8161"
        destinationName: "testQ"
        brokerName: "localhost"
      authenticationRef:
        name: trigger-auth-activemq
`
