package synthetics

import "context"

// SecureCredential represents a Synthetics secure credential.
type SecureCredential struct {
	Key         string `json:"key"`
	Description string `json:"description"`
	Value       string `json:"value"`
	CreatedAt   *Time  `json:"createdAt"`
	LastUpdated *Time  `json:"lastUpdated"`
}

// GetSecureCredentials is used to retrieve all secure credentials from your New Relic account.

// Deprecated: Use entities.GetEntitySearch instead.
func (s *Synthetics) GetSecureCredentials() ([]*SecureCredential, error) {
	return s.GetSecureCredentialsWithContext(context.Background())
}

// GetSecureCredentialsWithContext is used to retrieve all secure credentials from your New Relic account.

// Deprecated: Use entities.GetEntitySearchWithContext instead.
func (s *Synthetics) GetSecureCredentialsWithContext(ctx context.Context) ([]*SecureCredential, error) {
	resp := getSecureCredentialsResponse{}

	_, err := s.client.GetWithContext(ctx, s.config.Region().SyntheticsURL("/v1/secure-credentials"), nil, &resp)
	if err != nil {
		return nil, err
	}

	return resp.SecureCredentials, nil
}

// GetSecureCredential is used to retrieve a specific secure credential from your New Relic account.

// Deprecated: Use entities.GetEntitySearch instead.
func (s *Synthetics) GetSecureCredential(key string) (*SecureCredential, error) {
	return s.GetSecureCredentialWithContext(context.Background(), key)
}

// GetSecureCredentialWithContext is used to retrieve a specific secure credential from your New Relic account.

// Deprecated: Use entities.GetEntitySearchWithContext instead.
func (s *Synthetics) GetSecureCredentialWithContext(ctx context.Context, key string) (*SecureCredential, error) {
	var sc SecureCredential

	_, err := s.client.GetWithContext(ctx, s.config.Region().SyntheticsURL("/v1/secure-credentials", key), nil, &sc)
	if err != nil {
		return nil, err
	}

	return &sc, nil
}

// AddSecureCredential is used to add a secure credential to your New Relic account.

// Deprecated: Use synthetics.SyntheticsCreateSecureCredential instead.
func (s *Synthetics) AddSecureCredential(key, value, description string) (*SecureCredential, error) {
	return s.AddSecureCredentialWithContext(context.Background(), key, value, description)
}

// AddSecureCredentialWithContext is used to add a secure credential to your New Relic account.

// Deprecated: Use synthetics.SyntheticsCreateSecureCredentialWithContext instead.
func (s *Synthetics) AddSecureCredentialWithContext(ctx context.Context, key, value, description string) (*SecureCredential, error) {
	sc := &SecureCredential{
		Key:         key,
		Value:       value,
		Description: description,
	}

	_, err := s.client.PostWithContext(ctx, s.config.Region().SyntheticsURL("/v1/secure-credentials"), nil, sc, nil)
	if err != nil {
		return nil, err
	}

	return sc, nil
}

// UpdateSecureCredential is used to update a secure credential in your New Relic account.

// Deprecated: Use synthetics.SyntheticsUpdateSecureCredential instead
func (s *Synthetics) UpdateSecureCredential(key, value, description string) (*SecureCredential, error) {
	return s.UpdateSecureCredentialWithContext(context.Background(), key, value, description)
}

// UpdateSecureCredentialWithContext is used to update a secure credential in your New Relic account.

// Deprecated: Use synthetics.SyntheticsUpdateSecureCredentialWithContext instead
func (s *Synthetics) UpdateSecureCredentialWithContext(ctx context.Context, key, value, description string) (*SecureCredential, error) {
	sc := &SecureCredential{
		Key:         key,
		Value:       value,
		Description: description,
	}

	_, err := s.client.PutWithContext(ctx, s.config.Region().SyntheticsURL("/v1/secure-credentials", key), nil, sc, nil)

	if err != nil {
		return nil, err
	}

	return sc, nil
}

// DeleteSecureCredential deletes a secure credential from your New Relic account.

// Deprecated: Use synthetics.SyntheticsDeleteSecureCredential instead
func (s *Synthetics) DeleteSecureCredential(key string) error {
	return s.DeleteSecureCredentialWithContext(context.Background(), key)
}

// DeleteSecureCredentialWithContext deletes a secure credential from your New Relic account.

// Deprecated: Use synthetics.SyntheticsDeleteSecureCredentialWithContext instead
func (s *Synthetics) DeleteSecureCredentialWithContext(ctx context.Context, key string) error {
	_, err := s.client.DeleteWithContext(ctx, s.config.Region().SyntheticsURL("/v1/secure-credentials", key), nil, nil)
	if err != nil {
		return err
	}

	return nil
}

type getSecureCredentialsResponse struct {
	SecureCredentials []*SecureCredential `json:"secureCredentials"`
	Count             int                 `json:"count"`
}
