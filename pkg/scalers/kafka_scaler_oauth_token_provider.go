package scalers

import (
	"context"

	"github.com/IBM/sarama"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

type TokenProvider struct {
	tokenSource oauth2.TokenSource
	extensions  map[string]string
}

func OAuthBearerTokenProvider(clientID, clientSecret, tokenURL string, scopes []string, extensions map[string]string) sarama.AccessTokenProvider {
	cfg := clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     tokenURL,
		Scopes:       scopes,
	}

	return &TokenProvider{
		tokenSource: cfg.TokenSource(context.Background()),
		extensions:  extensions,
	}
}

func (t *TokenProvider) Token() (*sarama.AccessToken, error) {
	token, err := t.tokenSource.Token()
	if err != nil {
		return nil, err
	}

	return &sarama.AccessToken{Token: token.AccessToken, Extensions: t.extensions}, nil
}
