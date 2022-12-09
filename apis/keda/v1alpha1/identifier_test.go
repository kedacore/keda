package v1alpha1

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type testData struct {
	name               string
	expectedIdentifier string
	soName             string
	soNamespace        string
	soKind             string
}

var tests = []testData{
	{
		name:               "all lowercase",
		expectedIdentifier: "scaledobject.namespace.name",
		soName:             "name",
		soNamespace:        "namespace",
		soKind:             "scaledobject",
	},
	{
		name:               "all uppercase",
		expectedIdentifier: "scaledobject.namespace.name",
		soName:             "NAME",
		soNamespace:        "NAMESPACE",
		soKind:             "SCALEDOBJECT",
	},
	{
		name:               "camel case",
		expectedIdentifier: "scaledobject.namespace.name",
		soName:             "name",
		soNamespace:        "namespace",
		soKind:             "scaledobject",
	},
	{
		name:               "missing namespace",
		expectedIdentifier: "scaledobject..name",
		soName:             "name",
		soKind:             "scaledobject",
	},
}

func TestGeneratedIdentifierForScaledObject(t *testing.T) {
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			expectedIdentifier := test.expectedIdentifier
			genericIdentifier := GenerateIdentifier(test.soKind, test.soNamespace, test.soName)

			scaledObject := &ScaledObject{
				ObjectMeta: metav1.ObjectMeta{
					Name:      test.soName,
					Namespace: test.soNamespace,
				},
			}
			scaledObjectIdentifier := scaledObject.GenerateIdentifier()

			withTriggers, err := AsDuckWithTriggers(scaledObject)
			if err != nil {
				t.Errorf("got error while converting to WithTriggers object: %s", err)
			}
			withTriggersIdentifier := withTriggers.GenerateIdentifier()

			if expectedIdentifier != genericIdentifier {
				t.Errorf("genericIdentifier=%q doesn't equal the expectedIdentifier=%q", genericIdentifier, expectedIdentifier)
			}

			if expectedIdentifier != scaledObjectIdentifier {
				t.Errorf("scaledObjectIdentifier=%q doesn't equal the expectedIdentifier=%q", scaledObjectIdentifier, expectedIdentifier)
			}

			if expectedIdentifier != withTriggersIdentifier {
				t.Errorf("withTriggersIdentifier=%q doesn't equal the expectedIdentifier=%q", withTriggersIdentifier, expectedIdentifier)
			}
		})
	}
}
