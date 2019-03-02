package e2e

import (
	"testing"

	// Mysteriously required to support GCP auth (required by k8s libs).
	// Apparently just importing it is enough. @_@ side effects @_@.
	// https://github.com/kubernetes/client-go/issues/242
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	pkgTest "github.com/knative/pkg/test"
	"github.com/knative/serving/test"
)

const (
	configName               = "prod"
	routeName                = "noodleburg"
	helloWorldExpectedOutput = "Hello World! How about some tasty noodles?"
)

// Setup creates the client objects needed in the e2e tests.
func Setup(t *testing.T) *test.Clients {
	clients, err := test.NewClients(
		pkgTest.Flags.Kubeconfig,
		pkgTest.Flags.Cluster,
		test.ServingNamespace)
	if err != nil {
		t.Fatalf("Couldn't initialize clients: %v", err)
	}
	return clients
}

// CreateRouteAndConfig will create Route and Config objects using clients.
// The Config object will serve requests to a container started from the image at imagePath.
func CreateRouteAndConfig(t *testing.T, clients *test.Clients, image string, options *test.Options) (test.ResourceNames, error) {
	svcName := test.ObjectNameForTest(t)
	names := test.ResourceNames{
		Config: svcName,
		Route:  svcName,
		Image:  image,
	}

	if _, err := test.CreateConfiguration(t, clients, names, options); err != nil {
		return test.ResourceNames{}, err
	}
	_, err := test.CreateRoute(t, clients, names)
	return names, err
}
