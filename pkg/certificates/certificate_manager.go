/*
Copyright 2023 The KEDA Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package certificates

import (
	"context"
	"crypto/x509"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/open-policy-agent/cert-controller/pkg/rotator"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

// +kubebuilder:rbac:groups=apiregistration.k8s.io,resources=apiservices,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=admissionregistration.k8s.io,resources=validatingwebhookconfigurations,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups="",namespace=keda,resources=secrets,verbs=get;list;watch;create;update;patch;delete

type CertManager struct {
	SecretName            string
	CertDir               string
	OperatorService       string
	MetricsServerService  string
	WebhookService        string
	CAName                string
	CAOrganization        string
	ValidatingWebhookName string
	APIServiceName        string
	Logger                logr.Logger
	Ready                 chan struct{}
}

// AddCertificateRotation registers all needed services to generate the certificates and patches needed resources with the caBundle
func (cm CertManager) AddCertificateRotation(ctx context.Context, mgr manager.Manager) error {
	var rotatorHooks = []rotator.WebhookInfo{
		{
			Name: cm.ValidatingWebhookName,
			Type: rotator.Validating,
		},
		{
			Name: cm.APIServiceName,
			Type: rotator.APIService,
		},
	}

	err := cm.ensureSecret(ctx, mgr, cm.SecretName)
	if err != nil {
		return err
	}
	extraDNSNames := []string{}
	extraDNSNames = append(extraDNSNames, getDNSNames(cm.OperatorService)...)
	extraDNSNames = append(extraDNSNames, getDNSNames(cm.WebhookService)...)
	extraDNSNames = append(extraDNSNames, getDNSNames(cm.MetricsServerService)...)

	cm.Logger.V(1).Info("setting up cert rotation")
	err = rotator.AddRotator(mgr, &rotator.CertRotator{
		SecretKey: types.NamespacedName{
			Namespace: kedautil.GetPodNamespace(),
			Name:      cm.SecretName,
		},
		CertDir:                cm.CertDir,
		CAName:                 cm.CAName,
		CAOrganization:         cm.CAOrganization,
		DNSName:                extraDNSNames[0],
		ExtraDNSNames:          extraDNSNames,
		IsReady:                cm.Ready,
		Webhooks:               rotatorHooks,
		RestartOnSecretRefresh: true,
		RequireLeaderElection:  true,
		ExtKeyUsages: &[]x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
			x509.ExtKeyUsageClientAuth,
		},
	})
	return err
}

// getDNSNames  creates all the possible DNS names for a given service
func getDNSNames(service string) []string {
	namespace := kedautil.GetPodNamespace()
	return []string{
		service,
		fmt.Sprintf("%s.%s", service, namespace),
		fmt.Sprintf("%s.%s.svc", service, namespace),
		fmt.Sprintf("%s.%s.svc.local", service, namespace),
		fmt.Sprintf("%s.%s.svc.cluster.local", service, namespace),
	}
}

// ensureSecret ensures that the secret used for storing TLS certificates exists
func (cm CertManager) ensureSecret(ctx context.Context, mgr manager.Manager, secretName string) error {
	secrets := &corev1.SecretList{}
	kedaNamespace := kedautil.GetPodNamespace()
	opt := &client.ListOptions{
		Namespace: kedaNamespace,
	}

	err := mgr.GetAPIReader().List(ctx, secrets, opt)
	if err != nil {
		cm.Logger.Error(err, "unable to check secrets")
		return err
	}

	exists := false
	for _, secret := range secrets.Items {
		if secret.Name == secretName {
			exists = true
			break
		}
	}
	if !exists {
		secret := &corev1.Secret{
			ObjectMeta: v1.ObjectMeta{
				Name:      secretName,
				Namespace: kedaNamespace,
				Labels: map[string]string{
					"app":                         "keda-operator",
					"app.kubernetes.io/name":      "keda-operator",
					"app.kubernetes.io/component": "keda-operator",
					"app.kubernetes.io/part-of":   "keda",
				},
			},
		}
		err = mgr.GetClient().Create(ctx, secret)
		if err != nil {
			cm.Logger.Error(err, "unable to create certificates secret")
			return err
		}
		cm.Logger.V(1).Info(fmt.Sprintf("created the secret %s to store cert-controller certificates", secretName))
	}
	return nil
}
