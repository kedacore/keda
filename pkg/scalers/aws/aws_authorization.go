package aws

type AuthorizationMetadata struct {
	AwsRoleArn string

	AwsAccessKeyID     string
	AwsSecretAccessKey string
	AwsSessionToken    string

	PodIdentityOwner bool
	// Pod identity owner is confusing
	// and it'll be removed when we get
	// rid of the old aws podIdentities
	UsingPodIdentity bool

	ScalerUniqueKey string
}
