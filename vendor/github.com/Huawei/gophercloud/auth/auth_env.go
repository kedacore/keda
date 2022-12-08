package auth

import (
	"fmt"
	"os"

	"github.com/Huawei/gophercloud"
	"github.com/Huawei/gophercloud/auth/token"
	"github.com/Huawei/gophercloud/auth/aksk"
)

var nilTokenOptions = token.TokenOptions{}
var nilAKSKOptions = aksk.AKSKOptions{}
/*
TokenOptionsFromEnv fills out an token.TokenOptions structure with the
settings found on the various OpenStack OS_* environment variables.

The following variables provide sources of truth: OS_AUTH_URL, OS_USERNAME,
OS_PASSWORD, OS_TENANT_ID, and OS_TENANT_NAME.

Of these, OS_USERNAME, OS_PASSWORD, and OS_AUTH_URL must have settings,
or an error will result.  OS_TENANT_ID, OS_TENANT_NAME, OS_PROJECT_ID, and
OS_PROJECT_NAME are optional.

OS_TENANT_ID and OS_TENANT_NAME are mutually exclusive to OS_PROJECT_ID and
OS_PROJECT_NAME. If OS_PROJECT_ID and OS_PROJECT_NAME are set, they will
still be referred as "tenant" in Gophercloud.

To use this function, first set the OS_* environment variables (for example,
by sourcing an `openrc` file),or use os.Setenv() function :
	os.Setenv("OS_AUTH_URL", "https://iam.xxx.yyy.com/v3")
	os.Setenv("OS_USERNAME", "{your user name}")
	os.Setenv("OS_PASSWORD", "{your password }")

then:

	opts, err := auth.TokenOptionsFromEnv()
	if err != nil {
		if ue, ok := err.(*gophercloud.UnifiedError); ok {
			fmt.Println("ErrCode:", ue.ErrorCode())
			fmt.Println("Message:", ue.Message())
		}
		return
	}
	provider, err := openstack.AuthenticatedClient(opts)
Now use the provider, you can initialize the serviceClient.
*/
func TokenOptionsFromEnv() (token.TokenOptions, error) {
	authURL := os.Getenv("OS_AUTH_URL")
	username := os.Getenv("OS_USERNAME")
	userID := os.Getenv("OS_USERID")
	password := os.Getenv("OS_PASSWORD")
	tenantID := os.Getenv("OS_TENANT_ID")
	tenantName := os.Getenv("OS_TENANT_NAME")
	domainID := os.Getenv("OS_DOMAIN_ID")
	domainName := os.Getenv("OS_DOMAIN_NAME")

	// If OS_PROJECT_ID is set, overwrite tenantID with the value.
	if v := os.Getenv("OS_PROJECT_ID"); v != "" {
		tenantID = v
	}

	// If OS_PROJECT_NAME is set, overwrite tenantName with the value.
	if v := os.Getenv("OS_PROJECT_NAME"); v != "" {
		tenantName = v
	}

	if authURL == "" {
		message := fmt.Sprintf(gophercloud.CE_MissingInputMessage, "authURL")
		err := gophercloud.NewSystemCommonError(gophercloud.CE_MissingInputCode, message)
		return nilTokenOptions, err
	}

	if username == "" && userID == "" {
		message := fmt.Sprintf(gophercloud.CE_MissingInputMessage, "username")
		err := gophercloud.NewSystemCommonError(gophercloud.CE_MissingInputCode, message)
		return nilTokenOptions, err
	}

	if password == "" {
		message := fmt.Sprintf(gophercloud.CE_MissingInputMessage, "password")
		err := gophercloud.NewSystemCommonError(gophercloud.CE_MissingInputCode, message)
		return nilTokenOptions, err
	}

	to := token.TokenOptions{
		IdentityEndpoint: authURL,
		UserID:           userID,
		Username:         username,
		Password:         password,
		TenantID:         tenantID,
		TenantName:       tenantName,
		DomainID:         domainID,
		DomainName:       domainName,
	}

	return to, nil
}

/*
AKSKOptionsFromEnv fills out an aksk.AKSKOptions structure with the
settings found on the various HWCLOUD_* environment variables.

The following variables provide sources of truth: HWCLOUD_AUTH_URL, HWCLOUD_ACCESS_KEY,
HWCLOUD_SECRET_KEY, HWCLOUD_ACCESS_KEY_STS_TOKEN, and HWCLOUD_PROJECT_ID,HWCLOUD_DOMAIN_ID,HWCLOUD_REGION,HWCLOUD_DOMAIN_NAME.

Of these, HWCLOUD_AUTH_URL, HWCLOUD_ACCESS_KEY, and HWCLOUD_SECRET_KEY must have settings,
or an error will result.The rest of others are optional.

To use this function, first set the HWCLOUD_* environment variables (for example,
by sourcing an `openrc` file), or use os.Setenv() function :
	os.Setenv("HWCLOUD_AUTH_URL", "https://iam.xxx.yyy.com/v3")
	os.Setenv("HWCLOUD_ACCESS_KEY", "{your AK string}")
	os.Setenv("HWCLOUD_SECRET_KEY", "{your SK string}")

then:

	opts, err := auth.AKSKOptionsFromEnv()
	if err != nil {
		if ue, ok := err.(*gophercloud.UnifiedError); ok {
			fmt.Println("ErrCode:", ue.ErrorCode())
			fmt.Println("Message:", ue.Message())
		}
		return
	}
	provider, err := openstack.AuthenticatedClient(opts)
Now use the provider, you can initialize the serviceClient.
*/
func AKSKOptionsFromEnv() (aksk.AKSKOptions, error) {

	authURL := os.Getenv("HWCLOUD_AUTH_URL")
	ak := os.Getenv("HWCLOUD_ACCESS_KEY")
	sk := os.Getenv("HWCLOUD_SECRET_KEY")
	seToken := os.Getenv("HWCLOUD_ACCESS_KEY_STS_TOKEN")
	projectID := os.Getenv("HWCLOUD_PROJECT_ID")
	domainID := os.Getenv("HWCLOUD_DOMAIN_ID")
	region := os.Getenv("HWCLOUD_REGION")
	cloudName := os.Getenv("HWCLOUD_DOMAIN_NAME")

	// If HWCLOUD_CLOUD_NAME is set, overwrite HWCLOUD_DOMAIN_NAME with the value.
	if v := os.Getenv("HWCLOUD_CLOUD_NAME"); v != "" {
		cloudName = v
	}

	if authURL == "" {
		message := fmt.Sprintf(gophercloud.CE_MissingInputMessage, "authURL")
		err := gophercloud.NewSystemCommonError(gophercloud.CE_MissingInputCode, message)
		return nilAKSKOptions, err
	}

	if ak == "" {
		message := fmt.Sprintf(gophercloud.CE_MissingInputMessage, "AccessKey")
		err := gophercloud.NewSystemCommonError(gophercloud.CE_MissingInputCode, message)
		return nilAKSKOptions, err
	}

	if sk == "" {
		message := fmt.Sprintf(gophercloud.CE_MissingInputMessage, "SecretKey")
		err := gophercloud.NewSystemCommonError(gophercloud.CE_MissingInputCode, message)
		return nilAKSKOptions, err
	}

	akskOptions := aksk.AKSKOptions{
		IdentityEndpoint: authURL,
		AccessKey:        ak,
		SecretKey:        sk,
		SecurityToken:    seToken,
		ProjectID:        projectID,
		DomainID:         domainID,
		Region:           region,
		Cloud:            cloudName,
	}

	return akskOptions, nil
}
