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
	"k8s.io/apimachinery/pkg/api/errors"
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
	K8sClusterDomain      string
	CAName                string
	CAOrganization        string
	ValidatingWebhookName string
	APIServiceName        string
	Logger                logr.Logger
	Ready                 chan struct{}
	EnableWebhookPatching bool
}

// AddCertificateRotation registers all needed services to generate the certificates and patches needed resources with the caBundle
func (cm CertManager) AddCertificateRotation(ctx context.Context, mgr manager.Manager) error {
	rotatorHooks := []rotator.WebhookInfo{
		{
			Name: cm.APIServiceName,
			Type: rotator.APIService,
		},
	}

	if cm.EnableWebhookPatching {
		rotatorHooks = append(rotatorHooks,
			rotator.WebhookInfo{
				Name: cm.ValidatingWebhookName,
				Type: rotator.Validating,
			},
		)
	} else {
		cm.Logger.V(1).Info("Webhook patching is disabled, skipping webhook certificates")
	}

	err := cm.ensureSecret(ctx, mgr, cm.SecretName)
	if err != nil {
		return err
	}
	var extraDNSNames []string
	extraDNSNames = append(extraDNSNames, getDNSNames(cm.OperatorService, cm.K8sClusterDomain)...)
	extraDNSNames = append(extraDNSNames, getDNSNames(cm.MetricsServerService, cm.K8sClusterDomain)...)
	if cm.EnableWebhookPatching {
		extraDNSNames = append(extraDNSNames, getDNSNames(cm.WebhookService, cm.K8sClusterDomain)...)
	}

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
func getDNSNames(service, k8sClusterDomain string) []string {
	namespace := kedautil.GetPodNamespace()
	return []string{
		service,
		fmt.Sprintf("%s.%s", service, namespace),
		fmt.Sprintf("%s.%s.svc", service, namespace),
		fmt.Sprintf("%s.%s.svc.%s", service, namespace, k8sClusterDomain),
	}
}

// ensureSecret ensures that the secret used for storing TLS certificates exists
func (cm CertManager) ensureSecret(ctx context.Context, mgr manager.Manager, secretName string) error {
	secret := &corev1.Secret{}
	kedaNamespace := kedautil.GetPodNamespace()
	objKey := client.ObjectKey{
		Namespace: kedaNamespace,
		Name:      secretName,
	}
	create := false
	err := mgr.GetAPIReader().Get(ctx, objKey, secret)
	if err != nil {
		if errors.IsNotFound(err) {
			create = true
		} else {
			cm.Logger.Error(err, "unable to check secret")
			return err
		}
	}

	if create {
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
