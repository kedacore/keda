package token

// TokenIdOptions presents the required information for token ID auth
type TokenIdOptions struct {
	// IdentityEndpoint specifies the HTTP endpoint that is required to work with
	// the Identity API of the appropriate version. While it's ultimately needed by
	// all of the identity services, it will often be populated by a provider-level
	// function.
	//
	// The IdentityEndpoint is typically referred to as the "auth_url" or
	// "OS_AUTH_URL" in the information provided by the cloud operator.
	IdentityEndpoint string `json:"-" required:"true"`

	// AuthToken allows users to authenticate (possibly as another user) with an
	// authentication token ID.
	AuthToken string `json:"-"`

	// user project id
	ProjectID string

	DomainID string `json:"-" required:"true"`

}

// GetIdentityEndpoint,Implements the method of AuthOptionsProvider
func (opts TokenIdOptions) GetIdentityEndpoint() string {
	return opts.IdentityEndpoint
}

//GetProjectId, Implements the method of AuthOptionsProvider
func (opts TokenIdOptions) GetProjectId() string {
	return opts.ProjectID
}

// GetDomainId,Implements the method of AuthOptionsProvider
func (opts TokenIdOptions) GetDomainId() string {
	return opts.DomainID
}
