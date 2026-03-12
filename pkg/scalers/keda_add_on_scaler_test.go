package scalers

import (
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	"github.com/stretchr/testify/assert"
)

type TestCR struct {
	metav1.ObjectMeta `json:"metadata"`
	metav1.TypeMeta   `json:",inline"`
	Status            *struct {
		AddOnMetadata *struct {
			ServerAddress string            `json:"serverAddress"`
			Metadata      map[string]string `json:"metadata"`
			UsePushScaler bool              `json:"usePushScaler"`
		} `json:"addOnMetadata"`
	} `json:"status"`
}

func (t *TestCR) GetObjectKind() schema.ObjectKind {
	return &t.TypeMeta
}

func (t *TestCR) DeepCopyObject() runtime.Object {
	copy := *t
	return &copy
}

var kedaAddOnCRDGroup = "scalers"
var kedaAddOnCRDVersion = "v1"
var typeMeta = metav1.TypeMeta{
	Kind:       "TestCR",
	APIVersion: kedaAddOnCRDGroup + "/" + kedaAddOnCRDVersion,
}

type kedaAddOnScalerTestData struct {
	name     string
	resource *TestCR
	metadata map[string]string
	isError  bool
}

var kedaAddOnScalerTestDataset = []kedaAddOnScalerTestData{
	{
		name:     "Valid Add-On CR with Push Scaler",
		metadata: map[string]string{"apiVersion": typeMeta.APIVersion, "kind": typeMeta.Kind, "name": "test-addon-push"},
		resource: &TestCR{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-addon-push",
				Namespace: "default",
			},
			TypeMeta: typeMeta,
			Status: &struct {
				AddOnMetadata *struct {
					ServerAddress string            `json:"serverAddress"`
					Metadata      map[string]string `json:"metadata"`
					UsePushScaler bool              `json:"usePushScaler"`
				} `json:"addOnMetadata"`
			}{
				AddOnMetadata: &struct {
					ServerAddress string            `json:"serverAddress"`
					Metadata      map[string]string `json:"metadata"`
					UsePushScaler bool              `json:"usePushScaler"`
				}{
					ServerAddress: "http://test-scaler.default.svc.cluster.local",
					Metadata:      map[string]string{"key": "value"},
					UsePushScaler: true,
				},
			},
		},
	},
	{
		name:     "Valid Add-On CR without Push Scaler",
		metadata: map[string]string{"apiVersion": typeMeta.APIVersion, "kind": typeMeta.Kind, "name": "test-addon-no-push"},
		resource: &TestCR{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-addon-no-push",
				Namespace: "default",
			},
			TypeMeta: typeMeta,
			Status: &struct {
				AddOnMetadata *struct {
					ServerAddress string            `json:"serverAddress"`
					Metadata      map[string]string `json:"metadata"`
					UsePushScaler bool              `json:"usePushScaler"`
				} `json:"addOnMetadata"`
			}{
				AddOnMetadata: &struct {
					ServerAddress string            `json:"serverAddress"`
					Metadata      map[string]string `json:"metadata"`
					UsePushScaler bool              `json:"usePushScaler"`
				}{
					ServerAddress: "http://test-scaler.default.svc.cluster.local",
					Metadata:      map[string]string{"key": "value"},
					UsePushScaler: false,
				},
			},
		},
	},
}

func TestKedaAddOnScaler(t *testing.T) {
	for _, testData := range kedaAddOnScalerTestDataset {
		t.Run(testData.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			gv := schema.GroupVersion{Group: kedaAddOnCRDGroup, Version: kedaAddOnCRDVersion}
			scheme.AddKnownTypes(gv, &TestCR{})
			mapper := meta.NewDefaultRESTMapper([]schema.GroupVersion{gv})
			mapper.Add(gv.WithKind("TestCR"), meta.RESTScopeNamespace)

			clientBuilder := fake.NewClientBuilder().WithScheme(scheme).WithRESTMapper(mapper)
			clientBuilder = clientBuilder.WithRuntimeObjects(testData.resource)

			s, err := NewKedaAddOnScaler(
				t.Context(),
				clientBuilder.Build(),
				&scalersconfig.ScalerConfig{
					TriggerMetadata:         testData.metadata,
					GlobalHTTPTimeout:       1000 * time.Millisecond,
					ScalableObjectNamespace: testData.resource.Namespace,
				},
			)

			if testData.isError {
				assert.Error(t, err, "Expected error creating scaler: %v", err)
			} else {
				assert.NoError(t, err, "Unexpected error creating scaler: %v", err)
			}

			if testData.resource.Status.AddOnMetadata.UsePushScaler {
				if _, ok := s.(*externalPushScaler); !ok {
					t.Errorf("Expected an externalPushScaler, got %T", s)
				}
			}

			if !testData.resource.Status.AddOnMetadata.UsePushScaler {
				if _, ok := s.(*externalScaler); !ok {
					t.Errorf("Expected an externalScaler, got %T", s)
				}
			}
		})
	}
}
