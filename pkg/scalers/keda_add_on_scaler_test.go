package scalers

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
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
	if t == nil {
		return nil
	}
	// Start with a shallow copy of the top-level struct.
	copy := *t
	// Deep copy the Status field and its nested structures.
	if t.Status != nil {
		statusCopy := *t.Status
		if t.Status.AddOnMetadata != nil {
			addOnCopy := *t.Status.AddOnMetadata
			if addOnCopy.Metadata != nil {
				metadataCopy := make(map[string]string, len(addOnCopy.Metadata))
				for k, v := range addOnCopy.Metadata {
					metadataCopy[k] = v
				}
				addOnCopy.Metadata = metadataCopy
			}
			statusCopy.AddOnMetadata = &addOnCopy
		}
		copy.Status = &statusCopy
	}
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
					ServerAddress: "http://test-scaler.default.svc.cluster.local:6000",
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
					ServerAddress: "http://test-scaler.default.svc.cluster.local:6000",
					Metadata:      map[string]string{"key": "value"},
					UsePushScaler: false,
				},
			},
		},
	},
}

func TestKedaAddOnScaler(t *testing.T) {
	for idx, testData := range kedaAddOnScalerTestDataset {
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
					TriggerIndex:            idx,
					TriggerMetadata:         testData.metadata,
					GlobalHTTPTimeout:       1000 * time.Millisecond,
					ScalableObjectNamespace: testData.resource.Namespace,
				},
			)

			require.NoError(t, err, "Unexpected error creating scaler")

			if testData.resource.Status.AddOnMetadata.UsePushScaler {
				if es, ok := s.(*externalPushScaler); ok {
					assert.Equal(t, idx, es.metadata.triggerIndex)
				} else {
					assert.Fail(t, fmt.Sprintf("Expected an externalPushScaler, got %T", s))
				}
			}

			if !testData.resource.Status.AddOnMetadata.UsePushScaler {
				if es, ok := s.(*externalScaler); ok {
					assert.Equal(t, idx, es.metadata.triggerIndex)
				} else {
					assert.Fail(t, fmt.Sprintf("Expected an externalScaler, got %T", s))
				}
			}
		})
	}
}
