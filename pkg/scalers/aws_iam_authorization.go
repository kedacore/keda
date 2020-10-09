package scalers

import "fmt"

type awsAuthorizationMetadata struct {
	awsRoleArn string

	awsAccessKeyID     string
	awsSecretAccessKey string

	podIdentityOwner bool
}

func getAwsAuthorization(authParams, metadata, resolvedEnv map[string]string) (awsAuthorizationMetadata, error) {
	meta := awsAuthorizationMetadata{}

	if metadata["identityOwner"] == "operator" {
		meta.podIdentityOwner = false
	} else if metadata["identityOwner"] == "" || metadata["identityOwner"] == "pod" {
		meta.podIdentityOwner = true
		if authParams["awsRoleArn"] != "" {
			meta.awsRoleArn = authParams["awsRoleArn"]
		} else if (authParams["awsAccessKeyID"] != "" || authParams["awsAccessKeyId"] != "") && authParams["awsSecretAccessKey"] != "" {
			meta.awsAccessKeyID = authParams["awsAccessKeyID"]
			if meta.awsAccessKeyID == "" {
				meta.awsAccessKeyID = authParams["awsAccessKeyId"]
			}
			meta.awsSecretAccessKey = authParams["awsSecretAccessKey"]
		} else {
			if metadata["awsAccessKeyID"] != "" {
				meta.awsAccessKeyID = metadata["awsAccessKeyID"]
			} else if metadata["awsAccessKeyIDFromEnv"] != "" {
				meta.awsAccessKeyID = resolvedEnv[metadata["awsAccessKeyID"]]
			}

			if len(meta.awsAccessKeyID) == 0 {
				return meta, fmt.Errorf("awsAccessKeyID not found")
			}

			if metadata["awsSecretAccessKeyFromEnv"] != "" {
				meta.awsSecretAccessKey = resolvedEnv[metadata["awsSecretAccessKeyFromEnv"]]
			}

			if len(meta.awsSecretAccessKey) == 0 {
				return meta, fmt.Errorf("awsSecretAccessKey not found")
			}
		}
	}

	return meta, nil
}
