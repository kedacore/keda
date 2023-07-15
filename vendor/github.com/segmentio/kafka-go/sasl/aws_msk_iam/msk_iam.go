package aws_msk_iam

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"time"

	sigv4 "github.com/aws/aws-sdk-go/aws/signer/v4"
	"github.com/segmentio/kafka-go/sasl"
)

const (
	// These constants come from https://github.com/aws/aws-msk-iam-auth#details and
	// https://github.com/aws/aws-msk-iam-auth/blob/main/src/main/java/software/amazon/msk/auth/iam/internals/AWS4SignedPayloadGenerator.java.
	signVersion      = "2020_10_22"
	signService      = "kafka-cluster"
	signAction       = "kafka-cluster:Connect"
	signVersionKey   = "version"
	signHostKey      = "host"
	signUserAgentKey = "user-agent"
	signActionKey    = "action"
	queryActionKey   = "Action"
)

var signUserAgent = fmt.Sprintf("kafka-go/sasl/aws_msk_iam/%s", runtime.Version())

// Mechanism implements sasl.Mechanism for the AWS_MSK_IAM mechanism, based on the official java implementation:
// https://github.com/aws/aws-msk-iam-auth
type Mechanism struct {
	// The sigv4.Signer to use when signing the request. Required.
	Signer *sigv4.Signer
	// The region where the msk cluster is hosted, e.g. "us-east-1". Required.
	Region string
	// The time the request is planned for. Optional, defaults to time.Now() at time of authentication.
	SignTime time.Time
	// The duration for which the presigned request is active. Optional, defaults to 5 minutes.
	Expiry time.Duration
}

func (m *Mechanism) Name() string {
	return "AWS_MSK_IAM"
}

// Start produces the authentication values required for AWS_MSK_IAM. It produces the following json as a byte array,
// making use of the aws-sdk to produce the signed output.
// 	{
// 	  "version" : "2020_10_22",
// 	  "host" : "<broker host>",
// 	  "user-agent": "<user agent string from the client>",
// 	  "action": "kafka-cluster:Connect",
// 	  "x-amz-algorithm" : "<algorithm>",
// 	  "x-amz-credential" : "<clientAWSAccessKeyID>/<date in yyyyMMdd format>/<region>/kafka-cluster/aws4_request",
// 	  "x-amz-date" : "<timestamp in yyyyMMdd'T'HHmmss'Z' format>",
// 	  "x-amz-security-token" : "<clientAWSSessionToken if any>",
// 	  "x-amz-signedheaders" : "host",
// 	  "x-amz-expires" : "<expiration in seconds>",
// 	  "x-amz-signature" : "<AWS SigV4 signature computed by the client>"
// 	}
func (m *Mechanism) Start(ctx context.Context) (sess sasl.StateMachine, ir []byte, err error) {
	saslMeta := sasl.MetadataFromContext(ctx)
	if saslMeta == nil {
		return nil, nil, errors.New("missing sasl metadata")
	}

	query := url.Values{
		queryActionKey: {signAction},
	}

	signUrl := url.URL{
		Scheme:   "kafka",
		Host:     saslMeta.Host,
		Path:     "/",
		RawQuery: query.Encode(),
	}

	req, err := http.NewRequest("GET", signUrl.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	signTime := m.SignTime
	if signTime.IsZero() {
		signTime = time.Now()
	}

	expiry := m.Expiry
	if expiry == 0 {
		expiry = 5 * time.Minute
	}

	header, err := m.Signer.Presign(req, nil, signService, m.Region, expiry, signTime)
	if err != nil {
		return nil, nil, err
	}
	signedMap := map[string]string{
		signVersionKey:   signVersion,
		signHostKey:      signUrl.Host,
		signUserAgentKey: signUserAgent,
		signActionKey:    signAction,
	}
	// The protocol requires lowercase keys.
	for key, vals := range header {
		signedMap[strings.ToLower(key)] = vals[0]
	}
	for key, vals := range req.URL.Query() {
		signedMap[strings.ToLower(key)] = vals[0]
	}

	signedJson, err := json.Marshal(signedMap)
	return m, signedJson, err
}

func (m *Mechanism) Next(ctx context.Context, challenge []byte) (bool, []byte, error) {
	// After the initial step, the authentication is complete
	// kafka will return error if it rejected the credentials, so we'll only
	// arrive here on success.
	return true, nil, nil
}
