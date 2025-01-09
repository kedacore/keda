package testhelpers

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
)

var (
	letters = []rune("abcdefghijklmnopqrstuvwxyz")
)

// Our integration test Dummy App
const IntegrationTestApplicationEntityGUID = "MzgwNjUyNnxBUE18QVBQTElDQVRJT058NTczNDgyNjM4"
const IntegrationTestApplicationEntityGUIDNew = "MzgwNjUyNnxBUE18QVBQTElDQVRJT058NTUzNDQ4MjAy"

// name of the APM Application with the GUID IntegrationTestApplicationEntityGUIDNew
const IntegrationTestApplicationEntityNameNew = "Dummy App Pro Max"

// Our integration test account ID (v2 account)
const IntegrationTestAccountID = 3806526

// RandSeq is used to get a string made up of n random lowercase letters.
func RandSeq(n int) string {
	// commented the following line out, since this no longer works with go1.21 and needs no alternative, as seeding isn't needed
	// rand.Seed(time.Now().UnixNano())
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

// GetNonExistentIDs returns two non-existent IDs that can be used to test errors raised
// when Ingest/User keys which do not exist are updated or deleted.
func GetNonExistentIDs() (string, string) {
	return "00000FF000F0FFFF000F0F0000FF0FF0000000F000F00F0FF000FFFFF0FF00F0", "00000HH000H0HHHH000H0H0000HH0HH0000000H000H00H0HH000HHHHH0HH00H0"
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

// Use this method to generate a random name to be used in integration tests
// for whatever resource you might be trying to create and/or test.
// If no random character count is provided, the default behavior is to append 5 random
// letters to the end of the name.
//
// Example random name: nr-test-xmnvb
func GenerateRandomName(randCharCount int) string {
	if randCharCount == 0 {
		randCharCount = 5
	}

	return fmt.Sprintf("nr-test-%s", RandSeq(randCharCount))
}
