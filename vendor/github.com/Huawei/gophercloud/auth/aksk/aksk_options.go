package aksk

// AKSKAuthOptions presents the required information for AK/SK auth
type AKSKOptions struct {
	// IdentityEndpoint specifies the HTTP endpoint that is required to work with
	// the Identity API of the appropriate version. While it's ultimately needed by
	// all of the identity services, it will often be populated by a provider-level
	// function.
	//
	// The IdentityEndpoint is typically referred to as the "auth_url" or
	// "OS_AUTH_URL" in the information provided by the cloud operator.
	IdentityEndpoint string `json:"-" required:"true"`

	// user project id
	ProjectID string

	DomainID string `json:"-" required:"true"`

	// region
	Region string

	//Cloud name
	Domain string

	//Cloud name
	Cloud string

	AccessKey string //Access Key
	SecretKey string //Secret key

	SecurityToken string
}

// GetIdentityEndpoint,Implements the method of AuthOptionsProvider
func (opts AKSKOptions) GetIdentityEndpoint() string {
	return opts.IdentityEndpoint
}

//GetProjectId, Implements the method of AuthOptionsProvider
func (opts AKSKOptions) GetProjectId() string {
	return opts.ProjectID
}

// GetDomainId,Implements the method of AuthOptionsProvider
func (opts AKSKOptions) GetDomainId() string {
	return opts.DomainID
}
