/*
Copyright 2017 The Knative Authors

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

package webhook

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/knative/pkg/apis"
	"github.com/knative/pkg/apis/duck"
	"github.com/knative/pkg/kmp"
	"github.com/knative/pkg/logging"
	"github.com/knative/pkg/logging/logkey"
	perrors "github.com/pkg/errors"

	"github.com/markbates/inflect"
	"github.com/mattbaird/jsonpatch"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	v1beta1 "k8s.io/api/extensions/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	clientadmissionregistrationv1beta1 "k8s.io/client-go/kubernetes/typed/admissionregistration/v1beta1"
)

const (
	secretServerKey  = "server-key.pem"
	secretServerCert = "server-cert.pem"
	secretCACert     = "ca-cert.pem"
)

var (
	deploymentKind      = v1beta1.SchemeGroupVersion.WithKind("Deployment")
	errMissingNewObject = errors.New("the new object may not be nil")
)

// ControllerOptions contains the configuration for the webhook
type ControllerOptions struct {
	// WebhookName is the name of the webhook we create to handle
	// mutations before they get stored in the storage.
	WebhookName string

	// ServiceName is the service name of the webhook.
	ServiceName string

	// DeploymentName is the service name of the webhook.
	DeploymentName string

	// SecretName is the name of k8s secret that contains the webhook
	// server key/cert and corresponding CA cert that signed them. The
	// server key/cert are used to serve the webhook and the CA cert
	// is provided to k8s apiserver during admission controller
	// registration.
	SecretName string

	// Namespace is the namespace in which everything above lives.
	Namespace string

	// Port where the webhook is served. Per k8s admission
	// registration requirements this should be 443 unless there is
	// only a single port for the service.
	Port int

	// RegistrationDelay controls how long admission registration
	// occurs after the webhook is started. This is used to avoid
	// potential races where registration completes and k8s apiserver
	// invokes the webhook before the HTTP server is started.
	RegistrationDelay time.Duration

	// ClientAuthType declares the policy the webhook server will follow for
	// TLS Client Authentication.
	// The default value is tls.NoClientCert.
	ClientAuth tls.ClientAuthType
}

// ResourceCallback defines a signature for resource specific (Route, Configuration, etc.)
// handlers that can validate and mutate an object. If non-nil error is returned, object creation
// is denied. Mutations should be appended to the patches operations.
type ResourceCallback func(patches *[]jsonpatch.JsonPatchOperation, old GenericCRD, new GenericCRD) error

// ResourceDefaulter defines a signature for resource specific (Route, Configuration, etc.)
// handlers that can set defaults on an object. If non-nil error is returned, object creation
// is denied. Mutations should be appended to the patches operations.
type ResourceDefaulter func(patches *[]jsonpatch.JsonPatchOperation, crd GenericCRD) error

// AdmissionController implements the external admission webhook for validation of
// pilot configuration.
type AdmissionController struct {
	Client   kubernetes.Interface
	Options  ControllerOptions
	Handlers map[schema.GroupVersionKind]GenericCRD
	Logger   *zap.SugaredLogger
}

// GenericCRD is the interface definition that allows us to perform the generic
// CRD actions like deciding whether to increment generation and so forth.
type GenericCRD interface {
	apis.Defaultable
	apis.Validatable
	runtime.Object
}

// GetAPIServerExtensionCACert gets the Kubernetes aggregate apiserver
// client CA cert used by validator.
//
// NOTE: this certificate is provided kubernetes. We do not control
// its name or location.
func getAPIServerExtensionCACert(cl kubernetes.Interface) ([]byte, error) {
	const name = "extension-apiserver-authentication"
	c, err := cl.CoreV1().ConfigMaps(metav1.NamespaceSystem).Get(name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	const caFileName = "requestheader-client-ca-file"
	pem, ok := c.Data[caFileName]
	if !ok {
		return nil, fmt.Errorf("cannot find %s in ConfigMap %s: ConfigMap.Data is %#v", caFileName, name, c.Data)
	}
	return []byte(pem), nil
}

// MakeTLSConfig makes a TLS configuration suitable for use with the server
func makeTLSConfig(serverCert, serverKey, caCert []byte, clientAuthType tls.ClientAuthType) (*tls.Config, error) {
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)
	cert, err := tls.X509KeyPair(serverCert, serverKey)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    caCertPool,
		ClientAuth:   clientAuthType,
	}, nil
}

func getOrGenerateKeyCertsFromSecret(ctx context.Context, client kubernetes.Interface,
	options *ControllerOptions) (serverKey, serverCert, caCert []byte, err error) {
	logger := logging.FromContext(ctx)
	secret, err := client.CoreV1().Secrets(options.Namespace).Get(options.SecretName, metav1.GetOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, nil, nil, err
		}
		logger.Info("Did not find existing secret, creating one")
		newSecret, err := generateSecret(ctx, options)
		if err != nil {
			return nil, nil, nil, err
		}
		secret, err = client.CoreV1().Secrets(newSecret.Namespace).Create(newSecret)
		if err != nil && !apierrors.IsAlreadyExists(err) {
			return nil, nil, nil, err
		}
		// Ok, so something else might have created, try fetching it one more time
		secret, err = client.CoreV1().Secrets(options.Namespace).Get(options.SecretName, metav1.GetOptions{})
		if err != nil {
			return nil, nil, nil, err
		}
	}

	var ok bool
	if serverKey, ok = secret.Data[secretServerKey]; !ok {
		return nil, nil, nil, errors.New("server key missing")
	}
	if serverCert, ok = secret.Data[secretServerCert]; !ok {
		return nil, nil, nil, errors.New("server cert missing")
	}
	if caCert, ok = secret.Data[secretCACert]; !ok {
		return nil, nil, nil, errors.New("ca cert missing")
	}
	return serverKey, serverCert, caCert, nil
}

// validate checks whether "new" and "old" implement HasImmutableFields and checks them,
// it then delegates validation to apis.Validatable on "new".
func validate(old GenericCRD, new GenericCRD) error {
	if immutableNew, ok := new.(apis.Immutable); ok && old != nil {
		// Copy the old object and set defaults so that we don't reject our own
		// defaulting done earlier in the webhook.
		old = old.DeepCopyObject().(GenericCRD)
		old.SetDefaults()

		immutableOld, ok := old.(apis.Immutable)
		if !ok {
			return fmt.Errorf("unexpected type mismatch %T vs. %T", old, new)
		}
		if err := immutableNew.CheckImmutableFields(immutableOld); err != nil {
			return err
		}
	}
	// Can't just `return new.Validate()` because it doesn't properly nil-check.
	if err := new.Validate(); err != nil {
		return err
	}
	return nil
}

func setAnnotations(patches duck.JSONPatch, new, old GenericCRD, ui *authenticationv1.UserInfo) (duck.JSONPatch, error) {
	// Nowhere to set the annotations.
	if new == nil {
		return patches, nil
	}
	na, ok := new.(apis.Annotatable)
	if !ok {
		return patches, nil
	}
	var oa apis.Annotatable
	if old != nil {
		oa = old.(apis.Annotatable)
	}
	b, a := new.DeepCopyObject().(apis.Annotatable), na

	a.AnnotateUserInfo(oa, ui)
	patch, err := duck.CreatePatch(b, a)
	if err != nil {
		return nil, err
	}
	return append(patches, patch...), nil
}

// setDefaults simply leverages apis.Defaultable to set defaults.
func setDefaults(patches duck.JSONPatch, crd GenericCRD) (duck.JSONPatch, error) {
	before, after := crd.DeepCopyObject(), crd
	after.SetDefaults()

	patch, err := duck.CreatePatch(before, after)
	if err != nil {
		return nil, err
	}

	return append(patches, patch...), nil
}

func configureCerts(ctx context.Context, client kubernetes.Interface, options *ControllerOptions) (*tls.Config, []byte, error) {
	var apiServerCACert []byte
	if options.ClientAuth >= tls.VerifyClientCertIfGiven {
		var err error
		apiServerCACert, err = getAPIServerExtensionCACert(client)
		if err != nil {
			return nil, nil, err
		}
	}

	serverKey, serverCert, caCert, err := getOrGenerateKeyCertsFromSecret(ctx, client, options)
	if err != nil {
		return nil, nil, err
	}
	tlsConfig, err := makeTLSConfig(serverCert, serverKey, apiServerCACert, options.ClientAuth)
	if err != nil {
		return nil, nil, err
	}
	return tlsConfig, caCert, nil
}

// Run implements the admission controller run loop.
func (ac *AdmissionController) Run(stop <-chan struct{}) error {
	logger := ac.Logger
	ctx := logging.WithLogger(context.TODO(), logger)
	tlsConfig, caCert, err := configureCerts(ctx, ac.Client, &ac.Options)
	if err != nil {
		logger.Errorw("could not configure admission webhook certs", zap.Error(err))
		return err
	}

	server := &http.Server{
		Handler:   ac,
		Addr:      fmt.Sprintf(":%v", ac.Options.Port),
		TLSConfig: tlsConfig,
	}

	logger.Info("Found certificates for webhook...")
	if ac.Options.RegistrationDelay != 0 {
		logger.Infof("Delaying admission webhook registration for %v", ac.Options.RegistrationDelay)
	}

	select {
	case <-time.After(ac.Options.RegistrationDelay):
		cl := ac.Client.AdmissionregistrationV1beta1().MutatingWebhookConfigurations()
		if err := ac.register(ctx, cl, caCert); err != nil {
			logger.Errorw("failed to register webhook", zap.Error(err))
			return err
		}
		logger.Info("Successfully registered webhook")
	case <-stop:
		return nil
	}

	serverBootstrapErrCh := make(chan struct{})
	go func() {
		if err := server.ListenAndServeTLS("", ""); err != nil {
			logger.Errorw("ListenAndServeTLS for admission webhook returned error", zap.Error(err))
			close(serverBootstrapErrCh)
		}
	}()

	select {
	case <-stop:
		return server.Close()
	case <-serverBootstrapErrCh:
		return errors.New("webhook server bootstrap failed")
	}
}

// Register registers the external admission webhook for pilot
// configuration types.
func (ac *AdmissionController) register(
	ctx context.Context, client clientadmissionregistrationv1beta1.MutatingWebhookConfigurationInterface, caCert []byte) error { // nolint: lll
	logger := logging.FromContext(ctx)
	failurePolicy := admissionregistrationv1beta1.Fail

	var rules []admissionregistrationv1beta1.RuleWithOperations
	for gvk := range ac.Handlers {
		plural := strings.ToLower(inflect.Pluralize(gvk.Kind))

		rules = append(rules, admissionregistrationv1beta1.RuleWithOperations{
			Operations: []admissionregistrationv1beta1.OperationType{
				admissionregistrationv1beta1.Create,
				admissionregistrationv1beta1.Update,
			},
			Rule: admissionregistrationv1beta1.Rule{
				APIGroups:   []string{gvk.Group},
				APIVersions: []string{gvk.Version},
				Resources:   []string{plural},
			},
		})
	}

	// Sort the rules by Group, Version, Kind so that things are deterministically ordered.
	sort.Slice(rules, func(i, j int) bool {
		lhs, rhs := rules[i], rules[j]
		if lhs.APIGroups[0] != rhs.APIGroups[0] {
			return lhs.APIGroups[0] < rhs.APIGroups[0]
		}
		if lhs.APIVersions[0] != rhs.APIVersions[0] {
			return lhs.APIVersions[0] < rhs.APIVersions[0]
		}
		return lhs.Resources[0] < rhs.Resources[0]
	})

	webhook := &admissionregistrationv1beta1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: ac.Options.WebhookName,
		},
		Webhooks: []admissionregistrationv1beta1.Webhook{{
			Name:  ac.Options.WebhookName,
			Rules: rules,
			ClientConfig: admissionregistrationv1beta1.WebhookClientConfig{
				Service: &admissionregistrationv1beta1.ServiceReference{
					Namespace: ac.Options.Namespace,
					Name:      ac.Options.ServiceName,
				},
				CABundle: caCert,
			},
			FailurePolicy: &failurePolicy,
		}},
	}

	// Set the owner to our deployment.
	deployment, err := ac.Client.ExtensionsV1beta1().Deployments(ac.Options.Namespace).Get(ac.Options.DeploymentName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to fetch our deployment: %v", err)
	}
	deploymentRef := metav1.NewControllerRef(deployment, deploymentKind)
	webhook.OwnerReferences = append(webhook.OwnerReferences, *deploymentRef)

	// Try to create the webhook and if it already exists validate webhook rules.
	_, err = client.Create(webhook)
	if err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("failed to create a webhook: %v", err)
		}
		logger.Info("Webhook already exists")
		configuredWebhook, err := client.Get(ac.Options.WebhookName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("error retrieving webhook: %v", err)
		}
		if ok, err := kmp.SafeEqual(configuredWebhook.Webhooks, webhook.Webhooks); err != nil {
			return fmt.Errorf("error diffing webhooks: %v", err)
		} else if !ok {
			logger.Info("Updating webhook")
			// Set the ResourceVersion as required by update.
			webhook.ObjectMeta.ResourceVersion = configuredWebhook.ObjectMeta.ResourceVersion
			if _, err := client.Update(webhook); err != nil {
				return fmt.Errorf("failed to update webhook: %s", err)
			}
		} else {
			logger.Info("Webhook is already valid")
		}
	} else {
		logger.Info("Created a webhook")
	}
	return nil
}

// ServeHTTP implements the external admission webhook for mutating
// serving resources.
func (ac *AdmissionController) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := ac.Logger
	logger.Infof("Webhook ServeHTTP request=%#v", r)

	// Verify the content type is accurate.
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		http.Error(w, "invalid Content-Type, want `application/json`", http.StatusUnsupportedMediaType)
		return
	}

	var review admissionv1beta1.AdmissionReview
	if err := json.NewDecoder(r.Body).Decode(&review); err != nil {
		http.Error(w, fmt.Sprintf("could not decode body: %v", err), http.StatusBadRequest)
		return
	}

	logger = logger.With(
		zap.String(logkey.Kind, fmt.Sprint(review.Request.Kind)),
		zap.String(logkey.Namespace, review.Request.Namespace),
		zap.String(logkey.Name, review.Request.Name),
		zap.String(logkey.Operation, fmt.Sprint(review.Request.Operation)),
		zap.String(logkey.Resource, fmt.Sprint(review.Request.Resource)),
		zap.String(logkey.SubResource, fmt.Sprint(review.Request.SubResource)),
		zap.String(logkey.UserInfo, fmt.Sprint(review.Request.UserInfo)))
	reviewResponse := ac.admit(logging.WithLogger(r.Context(), logger), review.Request)
	var response admissionv1beta1.AdmissionReview
	if reviewResponse != nil {
		response.Response = reviewResponse
		response.Response.UID = review.Request.UID
	}

	logger.Infof("AdmissionReview for %#v: %s/%s response=%#v",
		review.Request.Kind, review.Request.Namespace, review.Request.Name, reviewResponse)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, fmt.Sprintf("could encode response: %v", err), http.StatusInternalServerError)
		return
	}
}

func makeErrorStatus(reason string, args ...interface{}) *admissionv1beta1.AdmissionResponse {
	result := apierrors.NewBadRequest(fmt.Sprintf(reason, args...)).Status()
	return &admissionv1beta1.AdmissionResponse{
		Result:  &result,
		Allowed: false,
	}
}

func (ac *AdmissionController) admit(ctx context.Context, request *admissionv1beta1.AdmissionRequest) *admissionv1beta1.AdmissionResponse {
	logger := logging.FromContext(ctx)
	switch request.Operation {
	case admissionv1beta1.Create, admissionv1beta1.Update:
	default:
		logger.Infof("Unhandled webhook operation, letting it through %v", request.Operation)
		return &admissionv1beta1.AdmissionResponse{Allowed: true}
	}

	patchBytes, err := ac.mutate(ctx, request)
	if err != nil {
		return makeErrorStatus("mutation failed: %v", err)
	}
	logger.Infof("Kind: %q PatchBytes: %v", request.Kind, string(patchBytes))

	return &admissionv1beta1.AdmissionResponse{
		Patch:   patchBytes,
		Allowed: true,
		PatchType: func() *admissionv1beta1.PatchType {
			pt := admissionv1beta1.PatchTypeJSONPatch
			return &pt
		}(),
	}
}

func (ac *AdmissionController) mutate(ctx context.Context, req *admissionv1beta1.AdmissionRequest) ([]byte, error) {
	kind := req.Kind
	newBytes := req.Object.Raw
	oldBytes := req.OldObject.Raw
	// Why, oh why are these different types...
	gvk := schema.GroupVersionKind{
		Group:   kind.Group,
		Version: kind.Version,
		Kind:    kind.Kind,
	}

	logger := logging.FromContext(ctx)
	handler, ok := ac.Handlers[gvk]
	if !ok {
		logger.Errorf("Unhandled kind: %v", gvk)
		return nil, fmt.Errorf("unhandled kind: %v", gvk)
	}

	// nil values denote absence of `old` (create) or `new` (delete) objects.
	var oldObj, newObj GenericCRD

	if len(newBytes) != 0 {
		newObj = handler.DeepCopyObject().(GenericCRD)
		newDecoder := json.NewDecoder(bytes.NewBuffer(newBytes))
		if err := newDecoder.Decode(&newObj); err != nil {
			return nil, fmt.Errorf("cannot decode incoming new object: %v", err)
		}
	}
	if len(oldBytes) != 0 {
		oldObj = handler.DeepCopyObject().(GenericCRD)
		oldDecoder := json.NewDecoder(bytes.NewBuffer(oldBytes))
		if err := oldDecoder.Decode(&oldObj); err != nil {
			return nil, fmt.Errorf("cannot decode incoming old object: %v", err)
		}
	}
	var patches duck.JSONPatch

	// Add these before defaulting fields, otherwise defaulting may cause an illegal patch because
	// it expects the round tripped through Golang fields to be present already.
	rtp, err := roundTripPatch(newBytes, newObj)
	if err != nil {
		return nil, fmt.Errorf("cannot create patch for round tripped newBytes: %v", err)
	}
	patches = append(patches, rtp...)

	if patches, err = setDefaults(patches, newObj); err != nil {
		logger.Errorw("Failed the resource specific defaulter", zap.Error(err))
		// Return the error message as-is to give the defaulter callback
		// discretion over (our portion of) the message that the user sees.
		return nil, err
	}

	if patches, err = setAnnotations(patches, newObj, oldObj, &req.UserInfo); err != nil {
		logger.Errorw("Failed the resource annotator", zap.Error(err))
		return nil, perrors.Wrap(err, "error setting annotations")
	}

	// None of the validators will accept a nil value for newObj.
	if newObj == nil {
		return nil, errMissingNewObject
	}
	if err := validate(oldObj, newObj); err != nil {
		logger.Errorw("Failed the resource specific validation", zap.Error(err))
		// Return the error message as-is to give the validation callback
		// discretion over (our portion of) the message that the user sees.
		return nil, err
	}
	return json.Marshal(patches)
}

// roundTripPatch generates the JSONPatch that corresponds to round tripping the given bytes through
// the Golang type (JSON -> Golang type -> JSON). Because it is not always true that
// bytes == json.Marshal(json.Unmarshal(bytes)).
//
// For example, if bytes did not contain a 'spec' field and the Golang type specifies its 'spec'
// field without omitempty, then by round tripping through the Golang type, we would have added
// `'spec': {}`.
func roundTripPatch(bytes []byte, unmarshalled interface{}) (duck.JSONPatch, error) {
	if unmarshalled == nil {
		return duck.JSONPatch{}, nil
	}
	marshaledBytes, err := json.Marshal(unmarshalled)
	if err != nil {
		return nil, fmt.Errorf("cannot marshal interface: %v", err)
	}
	return jsonpatch.CreatePatch(bytes, marshaledBytes)
}

func generateSecret(ctx context.Context, options *ControllerOptions) (*corev1.Secret, error) {
	serverKey, serverCert, caCert, err := CreateCerts(ctx, options.ServiceName, options.Namespace)
	if err != nil {
		return nil, err
	}
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      options.SecretName,
			Namespace: options.Namespace,
		},
		Data: map[string][]byte{
			secretServerKey:  serverKey,
			secretServerCert: serverCert,
			secretCACert:     caCert,
		},
	}, nil
}
