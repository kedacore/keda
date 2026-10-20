package resolver

import (
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

var parser = jwt.NewParser()

const maxProjectedServiceAccountTokenSize = 1 << 20

func readKubernetesServiceAccountProjectedToken(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return []byte{}, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return []byte{}, err
	}
	if !info.Mode().IsRegular() {
		return []byte{}, fmt.Errorf("service account token path %s is not a regular file", path)
	}
	if info.Size() > maxProjectedServiceAccountTokenSize {
		return []byte{}, fmt.Errorf("service account token file %s exceeds maximum size of %d bytes", path, maxProjectedServiceAccountTokenSize)
	}

	jwt, err := io.ReadAll(io.LimitReader(file, maxProjectedServiceAccountTokenSize+1))
	if err != nil {
		return []byte{}, err
	}
	if len(jwt) > maxProjectedServiceAccountTokenSize {
		return []byte{}, fmt.Errorf("service account token file %s exceeds maximum size of %d bytes", path, maxProjectedServiceAccountTokenSize)
	}
	if err = validateK8sSAToken(jwt); err != nil {
		return []byte{}, err
	}
	return jwt, nil
}

func validateK8sSAToken(saToken []byte) error {
	claims := jwt.MapClaims{}
	_, _, err := parser.ParseUnverified(string(saToken), &claims)
	if err != nil {
		return fmt.Errorf("error validating token: %w", err)
	}
	sub, err := claims.GetSubject()
	if err != nil {
		return fmt.Errorf("error getting token sub: %w", err)
	}
	if !strings.HasPrefix(sub, "system:serviceaccount:") {
		return fmt.Errorf("error validating token: subject isn't a service account")
	}

	return nil
}

// validateK8sSATokenAudiences filters outgoing tokens; the recipient verifies signatures.
// Configured audiences should not be accepted by kube-apiserver.
func validateK8sSATokenAudiences(saToken []byte, allowed []string) error {
	if len(allowed) == 0 {
		return fmt.Errorf("at least one approved service account token audience is required")
	}
	if err := validateK8sSAToken(saToken); err != nil {
		return err
	}
	claims := jwt.MapClaims{}
	if _, _, err := parser.ParseUnverified(string(saToken), &claims); err != nil {
		return fmt.Errorf("error parsing token audiences: %w", err)
	}
	actual, err := claims.GetAudience()
	if err != nil {
		return fmt.Errorf("error getting token audiences: %w", err)
	}
	if len(actual) == 0 {
		return fmt.Errorf("service account token has no audience")
	}
	for _, audience := range actual {
		if audience == "" || !slices.Contains(allowed, audience) {
			return fmt.Errorf("service account token contains an audience not approved by the allowed audience list")
		}
	}
	if err := jwt.NewValidator(jwt.WithExpirationRequired()).Validate(claims); err != nil {
		return fmt.Errorf("invalid service account token lifetime: %w", err)
	}
	return nil
}
