package gcp

import (
	"context"
	"encoding/base64"
	"fmt"

	"google.golang.org/api/idtoken"
)

func GetCloudID(audience string) (string, error) {
	if audience == "" {
		audience = "akeyless.io"
	}
	signedJWT, err := idtoken.NewTokenSource(context.Background(), audience)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve signed JWT: %w", err)
	}

	t, err := signedJWT.Token()
	if err != nil {
		return "", fmt.Errorf("signed JWT is not available: %w", err)
	}

	if !t.Valid() {
		return "", fmt.Errorf("signed JWT is invalid")
	}

	return base64.StdEncoding.EncodeToString([]byte(t.AccessToken)), nil
}
