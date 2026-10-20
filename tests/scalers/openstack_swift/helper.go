//go:build e2e
// +build e2e

package openstack

import (
	"context"
	"strings"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	containers "github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1/containers"
	objects "github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1/objects"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func CreateClient(t *testing.T, authURL, userID, password, projectID string) *gophercloud.ServiceClient {
	opts := gophercloud.AuthOptions{
		IdentityEndpoint: authURL,
		UserID:           userID,
		Password:         password,
		Scope: &gophercloud.AuthScope{
			ProjectID: projectID,
		},
	}
	provider, err := openstack.AuthenticatedClient(context.Background(), opts)
	require.NoErrorf(t, err, "cannot create the provider - %s", err)
	client, err := openstack.NewObjectStorageV1(provider, gophercloud.EndpointOpts{})
	require.NoErrorf(t, err, "cannot create the client - %s", err)
	return client
}

func CreateContainer(t *testing.T, client *gophercloud.ServiceClient, name string) {
	createOpts := containers.CreateOpts{
		ContentType: "application/json",
	}
	_, err := containers.Create(context.Background(), client, name, createOpts).Extract()
	require.NoErrorf(t, err, "cannot create the container - %s", err)
}

func DeleteContainer(t *testing.T, client *gophercloud.ServiceClient, name string) {
	_, err := containers.Delete(context.Background(), client, name).Extract()
	assert.NoErrorf(t, err, "cannot delete the container - %s", err)
}

func CreateObject(t *testing.T, client *gophercloud.ServiceClient, containerName, name string) {
	createOpts := objects.CreateOpts{
		ContentType: "text/plain",
		Content:     strings.NewReader("foo"),
	}
	_, err := objects.Create(context.Background(), client, containerName, name, createOpts).Extract()
	assert.NoErrorf(t, err, "cannot create the object - %s", err)
}

func DeleteObject(t *testing.T, client *gophercloud.ServiceClient, containerName, name string) {
	_, err := objects.Delete(context.Background(), client, containerName, name, objects.DeleteOpts{}).Extract()
	assert.NoErrorf(t, err, "cannot delete the object - %s", err)
}
