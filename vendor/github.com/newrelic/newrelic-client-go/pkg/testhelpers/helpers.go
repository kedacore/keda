package testhelpers

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"time"
)

var (
	letters = []rune("abcdefghijklmnopqrstuvwxyz")
)

// RandSeq is used to get a string made up of n random lowercase letters.
func RandSeq(n int) string {
	rand.Seed(time.Now().UnixNano())
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// GetTestUserID returns the integer value for a New Relic user ID from the environment
func GetTestUserID() (int, error) {
	return getEnvInt("NEW_RELIC_TEST_USER_ID")
}

// GetTestAccountID returns the integer value for a New Relic Account ID from the environment
func GetTestAccountID() (int, error) {
	return getEnvInt("NEW_RELIC_ACCOUNT_ID")
}

// getEnvInt helper to DRY up the other env get calls for integers
func getEnvInt(name string) (int, error) {
	if name == "" {
		return 0, fmt.Errorf("failed to get environment value, no name specified")
	}

	id := os.Getenv(name)

	if id == "" {
		return 0, fmt.Errorf("failed to get environment value due to undefined environment variable %s", name)
	}

	n, err := strconv.Atoi(id)
	if err != nil {
		return 0, err
	}

	return n, nil
}
