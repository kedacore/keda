//go:build e2e
// +build e2e

package activemq_test

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"

	. "github.com/kedacore/keda/v2/tests/helper"
)

// Load environment variables from .env file
var _ = godotenv.Load("../../.env")

const (
	testName = "activemq-test"
)

var (
	testNamespace       = fmt.Sprintf("%s-ns", testName)
	deploymentName      = fmt.Sprintf("%s-deployment", testName)
	scaledObjectName    = fmt.Sprintf("%s-so", testName)
	secretName          = fmt.Sprintf("%s-secret", testName)
	activemqUser        = "admin"
	activemqPassword    = "admin"
	activemqConf        = "/opt/apache-activemq-5.16.3/conf"
	activemqHome        = "/opt/apache-activemq-5.16.3"
	activemqPath        = "bin/activemq"
	activemqDestination = "testQ"
	activemqPodName     = "activemq-0"
	minReplicaCount     = 0
	maxReplicaCount     = 2
)

type templateData struct {
	TestNamespace          string
	DeploymentName         string
	ScaledObjectName       string
	SecretName             string
	ActiveMQPasswordBase64 string
	ActiveMQUserBase64     string
	ActiveMQConf           string
	ActiveMQHome           string
	ActiveMQDestination    string
}

const (
	secretTemplate = `apiVersion: v1
kind: Secret
metadata:
  name: {{.SecretName}}
  namespace: {{.TestNamespace}}
data:
  activemq-password: {{.ActiveMQPasswordBase64}}
  activemq-username: {{.ActiveMQUserBase64}}
`

	triggerAuthenticationTemplate = `apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: keda-trigger-auth-activemq-secret
  namespace: {{.TestNamespace}}
spec:
  secretTargetRef:
    - parameter: username
      name: {{.SecretName}}
      key: activemq-username
    - parameter: password
      name: {{.SecretName}}
      key: activemq-password
`

	deploymentTemplate = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.DeploymentName}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.DeploymentName}}
spec:
  replicas: 0
  selector:
    matchLabels:
      app: {{.DeploymentName}}
  template:
    metadata:
      labels:
        app: {{.DeploymentName}}
    spec:
      containers:
      - name: nginx
        image: ghcr.io/nginx/nginx-unprivileged:1.26
        ports:
        - containerPort: 80
`

	activemqSteatefulTemplate = `apiVersion: apps/v1
kind: StatefulSet
metadata:
  labels:
    app: activemq-app
  name: activemq
  namespace: {{.TestNamespace}}
spec:
  replicas: 1
  serviceName: activemq
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
        volumeMounts:
        - name: remote-access-cm
          mountPath: /opt/apache-activemq-5.16.3/webapps/api/WEB-INF/classes/jolokia-access.xml
          subPath: jolokia-access.xml
        - name: config
          mountPath: /opt/apache-activemq-5.16.3/conf/jetty.xml
          subPath: jetty.xml
      volumes:
      - name: remote-access-cm
        configMap:
          name: activemq-config
          items:
          - key: jolokia-access.xml
            path: jolokia-access.xml
      - name: config
        configMap:
          name: activemq-config
          items:
          - key: jetty.xml
            path: jetty.xml
`
	activemqServiceTemplate = `apiVersion: v1
kind: Service
metadata:
  name: activemq
  namespace: {{.TestNamespace}}
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
`
	activemqConfigTemplate = `apiVersion: v1
kind: ConfigMap
metadata:
  name: activemq-config
  namespace: {{.TestNamespace}}
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
        <property name="config" value="{{.ActiveMQConf}}/jetty-realm.properties" />
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
                    <property name="resourceBase" value="{{.ActiveMQHome}}/webapps/admin" />
                    <property name="logUrlOnStart" value="true" />
                </bean>
                <bean class="org.eclipse.jetty.webapp.WebAppContext">
                    <property name="contextPath" value="/api" />
                    <property name="resourceBase" value="{{.ActiveMQHome}}/webapps/api" />
                    <property name="logUrlOnStart" value="true" />
                </bean>
                <bean class="org.eclipse.jetty.server.handler.ResourceHandler">
                    <property name="directoriesListed" value="false" />
                    <property name="welcomeFiles">
                        <list>
                            <value>index.html</value>
                        </list>
                    </property>
                    <property name="resourceBase" value="{{.ActiveMQHome}}/webapps/" />
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

                            <property name="keyStorePath" value="{{.ActiveMQConf}}/broker.ks" />
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

	scaledObjectTemplate = `
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: {{.ScaledObjectName}}
  namespace: {{.TestNamespace}}
  labels:
    app: {{.DeploymentName}}
spec:
  scaleTargetRef:
    name: {{.DeploymentName}}
  minReplicaCount: 0
  maxReplicaCount: 2
  pollingInterval: 1
  cooldownPeriod:  1
  triggers:
    - type: activemq
      metadata:
        managementEndpoint: "activemq.{{.TestNamespace}}:8161"
        destinationName: "{{.ActiveMQDestination}}"
        brokerName: "localhost"
        activationTargetQueueSize: "500"
      authenticationRef:
        name: keda-trigger-auth-activemq-secret
`
)

func TestActiveMQScaler(t *testing.T) {
	kc := GetKubernetesClient(t)
	data, templates := getTemplateData()
	t.Cleanup(func() {
		DeleteKubernetesResources(t, testNamespace, data, templates)
	})

	// Create kubernetes resources
	CreateKubernetesResources(t, kc, testNamespace, data, templates)

	// setup activemq
	setupActiveMQ(t, kc)

	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)

	// test scaling
	testActivation(t, kc)
	testScaleOut(t, kc)
	testScaleIn(t, kc)
}

func setupActiveMQ(t *testing.T, kc *kubernetes.Clientset) {
	require.True(t, WaitForStatefulsetReplicaReadyCount(t, kc, "activemq", testNamespace, 1, 60, 3),
		"activemq should be up")
	err := checkIfActiveMQStatusIsReady(t, activemqPodName)
	require.NoErrorf(t, err, "%s", err)
}

func checkIfActiveMQStatusIsReady(t *testing.T, name string) error {
	t.Log("--- checking activemq status ---")
	time.Sleep(time.Second * 10)
	for i := 0; i < 60; i++ {
		out, errOut, _ := ExecCommandOnSpecificPod(t, name, testNamespace, fmt.Sprintf("%s query –objname type=Broker,brokerName=localhost,Service=Health", activemqPath))
		t.Logf("Output: %s, Error: %s", out, errOut)
		if !strings.Contains(out, "CurrentStatus = Good") {
			time.Sleep(time.Second * 10)
			continue
		}
		return nil
	}
	return errors.New("activemq is not ready")
}

func testActivation(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing activation ---")
	_, _, err := ExecCommandOnSpecificPod(t, activemqPodName, testNamespace, fmt.Sprintf("%s producer --destination %s --messageCount 100", activemqPath, activemqDestination))
	assert.NoErrorf(t, err, "cannot enqueue messages - %s", err)
	AssertReplicaCountNotChangeDuringTimePeriod(t, kc, deploymentName, testNamespace, minReplicaCount, 60)
}

func testScaleOut(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale out ---")
	_, _, err := ExecCommandOnSpecificPod(t, activemqPodName, testNamespace, fmt.Sprintf("%s producer --destination %s --messageCount 900", activemqPath, activemqDestination))
	assert.NoErrorf(t, err, "cannot enqueue messages - %s", err)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, maxReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", maxReplicaCount)
}

func testScaleIn(t *testing.T, kc *kubernetes.Clientset) {
	t.Log("--- testing scale in ---")
	_, _, err := ExecCommandOnSpecificPod(t, activemqPodName, testNamespace, fmt.Sprintf("%s consumer --destination %s --messageCount 1000", activemqPath, activemqDestination))
	assert.NoErrorf(t, err, "cannot enqueue messages - %s", err)
	assert.True(t, WaitForDeploymentReplicaReadyCount(t, kc, deploymentName, testNamespace, minReplicaCount, 60, 3),
		"replica count should be %d after 3 minutes", minReplicaCount)
}

func getTemplateData() (templateData, []Template) {
	return templateData{
			TestNamespace:          testNamespace,
			DeploymentName:         deploymentName,
			ScaledObjectName:       scaledObjectName,
			SecretName:             secretName,
			ActiveMQPasswordBase64: base64.StdEncoding.EncodeToString([]byte(activemqPassword)),
			ActiveMQUserBase64:     base64.StdEncoding.EncodeToString([]byte(activemqUser)),
			ActiveMQConf:           activemqConf,
			ActiveMQHome:           activemqHome,
			ActiveMQDestination:    activemqDestination,
		}, []Template{
			{Name: "secretTemplate", Config: secretTemplate},
			{Name: "triggerAuthenticationTemplate", Config: triggerAuthenticationTemplate},
			{Name: "activemqServiceTemplate", Config: activemqServiceTemplate},
			{Name: "activemqConfigTemplate", Config: activemqConfigTemplate},
			{Name: "activemqSteatefulTemplate", Config: activemqSteatefulTemplate},
			{Name: "deploymentTemplate", Config: deploymentTemplate},
			{Name: "scaledObjectTemplate", Config: scaledObjectTemplate},
		}
}
