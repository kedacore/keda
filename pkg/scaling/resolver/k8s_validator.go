package resolver

import (
	"fmt"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

var parser = jwt.NewParser()

func readKubernetesServiceAccountProjectedToken(path string) ([]byte, error) {
	jwt, err := os.ReadFile(path)
	if err != nil {
		return []byte{}, err
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
