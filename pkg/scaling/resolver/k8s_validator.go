package resolver

import (
	"fmt"
	"io"
	"os"
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
