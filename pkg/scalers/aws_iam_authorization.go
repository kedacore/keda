package scalers

import "fmt"

const (
	awsAccessKeyIDEnvVar     = "AWS_ACCESS_KEY_ID"
	awsSecretAccessKeyEnvVar = "AWS_SECRET_ACCESS_KEY"
	awsSessionTokenEnvVar    = "AWS_SESSION_TOKEN"
)

type awsAuthorizationMetadata struct {
	awsRoleArn string

	awsAccessKeyID     string
	awsSecretAccessKey string
	awsSessionToken    string
}

func getAwsAuthorization(authParams, metadata, resolvedEnv map[string]string) (awsAuthorizationMetadata, error) {
	meta := awsAuthorizationMetadata{}

	if authParams["awsRoleArn"] != "" {
		meta.awsRoleArn = authParams["awsRoleArn"]
	} else if authParams["awsAccessKeyId"] != "" && authParams["awsSecretAccessKey"] != "" {
		meta.awsAccessKeyID = authParams["awsAccessKeyId"]
		meta.awsSecretAccessKey = authParams["awsSecretAccessKey"]
	} else {
		var keyName string
		if keyName = metadata["awsAccessKeyID"]; keyName == "" {
			keyName = awsAccessKeyIDEnvVar
		}
		if val, ok := resolvedEnv[keyName]; ok && val != "" {
			meta.awsAccessKeyID = val
		} else {
			return meta, fmt.Errorf("'%s' doesn't exist in the deployment environment", keyName)
		}

		if keyName = metadata["awsSecretAccessKey"]; keyName == "" {
			keyName = awsSecretAccessKeyEnvVar
		}
		if val, ok := resolvedEnv[keyName]; ok && val != "" {
			meta.awsSecretAccessKey = val
		} else {
			return meta, fmt.Errorf("'%s' doesn't exist in the deployment environment", keyName)
		}
	}

	return meta, nil
}
