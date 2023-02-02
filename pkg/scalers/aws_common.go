package scalers

import (
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
)

// ErrAwsNoAccessKey is returned when awsAccessKeyID is missing.
var ErrAwsNoAccessKey = errors.New("awsAccessKeyID not found")

type awsAuthorizationMetadata struct {
	awsRoleArn string

	awsAccessKeyID     string
	awsSecretAccessKey string
	awsSessionToken    string

	podIdentityOwner bool
}

type awsConfigMetadata struct {
	awsRegion        string
	awsEndpoint      string
	awsAuthorization awsAuthorizationMetadata
}

func getAwsConfig(awsRegion string, awsEndpoint string, awsAuthorization awsAuthorizationMetadata) (*session.Session, *aws.Config) {
	metadata := &awsConfigMetadata{
		awsRegion:        awsRegion,
		awsEndpoint:      awsEndpoint,
		awsAuthorization: awsAuthorization}

	sess := session.Must(session.NewSession(&aws.Config{
		Region:   aws.String(metadata.awsRegion),
		Endpoint: aws.String(metadata.awsEndpoint),
	}))

	if !metadata.awsAuthorization.podIdentityOwner {
		return sess, &aws.Config{
			Region:   aws.String(metadata.awsRegion),
			Endpoint: aws.String(metadata.awsEndpoint),
		}
	}

	creds := credentials.NewStaticCredentials(metadata.awsAuthorization.awsAccessKeyID, metadata.awsAuthorization.awsSecretAccessKey, "")

	if metadata.awsAuthorization.awsRoleArn != "" {
		creds = stscreds.NewCredentials(sess, metadata.awsAuthorization.awsRoleArn)
	}

	return sess, &aws.Config{
		Region:      aws.String(metadata.awsRegion),
		Endpoint:    aws.String(metadata.awsEndpoint),
		Credentials: creds,
	}
}

func getAwsAuthorization(authParams, metadata, resolvedEnv map[string]string) (awsAuthorizationMetadata, error) {
	meta := awsAuthorizationMetadata{}

	if metadata["identityOwner"] == "operator" {
		meta.podIdentityOwner = false
	} else if metadata["identityOwner"] == "" || metadata["identityOwner"] == "pod" {
		meta.podIdentityOwner = true
		switch {
		case authParams["awsRoleArn"] != "":
			meta.awsRoleArn = authParams["awsRoleArn"]
		case (authParams["awsAccessKeyID"] != "" || authParams["awsAccessKeyId"] != "") && authParams["awsSecretAccessKey"] != "":
			meta.awsAccessKeyID = authParams["awsAccessKeyID"]
			if meta.awsAccessKeyID == "" {
				meta.awsAccessKeyID = authParams["awsAccessKeyId"]
			}
			meta.awsSecretAccessKey = authParams["awsSecretAccessKey"]
			meta.awsSessionToken = authParams["awsSessionToken"]
		default:
			if metadata["awsAccessKeyID"] != "" {
				meta.awsAccessKeyID = metadata["awsAccessKeyID"]
			} else if metadata["awsAccessKeyIDFromEnv"] != "" {
				meta.awsAccessKeyID = resolvedEnv[metadata["awsAccessKeyIDFromEnv"]]
			}

			if len(meta.awsAccessKeyID) == 0 {
				return meta, ErrAwsNoAccessKey
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
