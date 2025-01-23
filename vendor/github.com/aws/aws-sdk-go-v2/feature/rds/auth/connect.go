package auth

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
)

const (
	signingID        = "rds-db"
	emptyPayloadHash = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
)

// BuildAuthTokenOptions is the optional set of configuration properties for BuildAuthToken
type BuildAuthTokenOptions struct{}

// BuildAuthToken will return an authorization token used as the password for a DB
// connection.
//
// * endpoint - Endpoint consists of the port needed to connect to the DB. <host>:<port>
// * region - Region is the location of where the DB is
// * dbUser - User account within the database to sign in with
// * creds - Credentials to be signed with
//
// The following example shows how to use BuildAuthToken to create an authentication
// token for connecting to a MySQL database in RDS.
//
//	authToken, err := BuildAuthToken(dbEndpoint, awsRegion, dbUser, awsCreds)
//
//	// Create the MySQL DNS string for the DB connection
//	// user:password@protocol(endpoint)/dbname?<params>
//	connectStr = fmt.Sprintf("%s:%s@tcp(%s)/%s?allowCleartextPasswords=true&tls=rds",
//	   dbUser, authToken, dbEndpoint, dbName,
//	)
//
//	// Use db to perform SQL operations on database
//	db, err := sql.Open("mysql", connectStr)
//
// See http://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/UsingWithRDS.IAMDBAuth.html
// for more information on using IAM database authentication with RDS.
func BuildAuthToken(ctx context.Context, endpoint, region, dbUser string, creds aws.CredentialsProvider, optFns ...func(options *BuildAuthTokenOptions)) (string, error) {
	_, port := validateURL(endpoint)
	if port == "" {
		return "", fmt.Errorf("the provided endpoint is missing a port, or the provided port is invalid")
	}

	o := BuildAuthTokenOptions{}

	for _, fn := range optFns {
		fn(&o)
	}

	if creds == nil {
		return "", fmt.Errorf("credetials provider must not ne nil")
	}

	// the scheme is arbitrary and is only needed because validation of the URL requires one.
	if !(strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://")) {
		endpoint = "https://" + endpoint
	}

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return "", err
	}
	values := req.URL.Query()
	values.Set("Action", "connect")
	values.Set("DBUser", dbUser)
	req.URL.RawQuery = values.Encode()

	signer := v4.NewSigner()

	credentials, err := creds.Retrieve(ctx)
	if err != nil {
		return "", err
	}

	// Expire Time: 15 minute
	query := req.URL.Query()
	query.Set("X-Amz-Expires", "900")
	req.URL.RawQuery = query.Encode()

	signedURI, _, err := signer.PresignHTTP(ctx, credentials, req, emptyPayloadHash, signingID, region, time.Now().UTC())
	if err != nil {
		return "", err
	}

	url := signedURI
	if strings.HasPrefix(url, "http://") {
		url = url[len("http://"):]
	} else if strings.HasPrefix(url, "https://") {
		url = url[len("https://"):]
	}

	return url, nil
}

func validateURL(hostPort string) (host, port string) {
	colon := strings.LastIndexByte(hostPort, ':')
	if colon != -1 {
		host, port = hostPort[:colon], hostPort[colon+1:]
	}
	if !validatePort(port) {
		port = ""
		return
	}
	if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		host = host[1 : len(host)-1]
	}

	return
}

func validatePort(port string) bool {
	if _, err := strconv.Atoi(port); err == nil {
		return true
	}
	return false
}
