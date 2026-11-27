package scalers

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	rdsauth "github.com/aws/aws-sdk-go-v2/feature/rds/auth"
	"github.com/go-logr/logr"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	awsutils "github.com/kedacore/keda/v2/pkg/scalers/aws"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

type parsePostgreSQLMetadataTestData struct {
	metadata map[string]string
}

var testPostgreSQLMetadata = []parsePostgreSQLMetadataTestData{
	// connection with username and password
	{metadata: map[string]string{"query": "test_query", "targetQueryValue": "5", "connectionFromEnv": "test_connection_string"}},
	// connection with username
	{metadata: map[string]string{"query": "test_query", "targetQueryValue": "5", "connectionFromEnv": "test_connection_string2"}},
	// connection with activationTargetQueryValue
	{metadata: map[string]string{"query": "test_query", "targetQueryValue": "5", "activationTargetQueryValue": "3", "connectionFromEnv": "test_connection_string2"}},
	// connection without username and password
	{metadata: map[string]string{"query": "test_query", "targetQueryValue": "5", "connection": "postgresql://localhost:5432"}},
	// connection with password + metricname
	{metadata: map[string]string{"query": "test_query", "targetQueryValue": "5", "connection": "postgresql://username:password@localhost:5432", "metricName": "scaler_sql_data2"}},
	// dbName
	{metadata: map[string]string{"query": "test_query", "targetQueryValue": "5", "host": "test_host", "port": "test_port", "userName": "test_user_name", "dbName": "test_db_name", "sslmode": "test_ssl_mode"}},
	// dbName + metricName
	{metadata: map[string]string{"query": "test_query", "targetQueryValue": "5", "host": "test_host", "port": "test_port", "userName": "test_user_name", "dbName": "test_db_name", "sslmode": "test_ssl_mode", "metricName": "scaler_sql_data"}},
}

type postgreSQLMetricIdentifier struct {
	metadataTestData *parsePostgreSQLMetadataTestData
	resolvedEnv      map[string]string
	authParam        map[string]string
	scaleIndex       int
	name             string
}

var postgreSQLMetricIdentifiers = []postgreSQLMetricIdentifier{
	{&testPostgreSQLMetadata[0], map[string]string{"test_connection_string": "postgresql://localhost:5432"}, nil, 0, "s0-postgresql"},
	{&testPostgreSQLMetadata[1], map[string]string{"test_connection_string2": "postgresql://test@localhost"}, nil, 1, "s1-postgresql"},
}

func TestPostgreSQLGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range postgreSQLMetricIdentifiers {
		meta, _, err := parsePostgreSQLMetadata(logr.Discard(), &scalersconfig.ScalerConfig{ResolvedEnv: testData.resolvedEnv, TriggerMetadata: testData.metadataTestData.metadata, AuthParams: testData.authParam, TriggerIndex: testData.scaleIndex})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockPostgresSQLScaler := postgreSQLScaler{"", meta, nil, kedav1alpha1.AuthPodIdentity{}, logr.Discard()}

		metricSpec := mockPostgresSQLScaler.GetMetricSpecForScaling(context.Background())
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}

type postgreSQLConnectionStringTestData struct {
	metadata         map[string]string
	resolvedEnv      map[string]string
	authParam        map[string]string
	connectionString string
}

var testPostgreSQLConnectionstring = []postgreSQLConnectionStringTestData{
	// from environment
	{metadata: map[string]string{"query": "test_query", "targetQueryValue": "5", "connectionFromEnv": "CONNECTION_ENV"}, resolvedEnv: map[string]string{"CONNECTION_ENV": "test_connection_from_env"}, connectionString: "test_connection_from_env"},
	// from authentication
	{metadata: map[string]string{"query": "test_query", "targetQueryValue": "5"}, authParam: map[string]string{"connection": "test_connection_from_auth"}, connectionString: "test_connection_from_auth"},
	// from meta
	{metadata: map[string]string{"query": "test_query", "targetQueryValue": "5", "host": "localhost", "port": "1234", "dbName": "testDb", "userName": "user", "sslmode": "required"}, connectionString: "host=localhost port=1234 user=user dbname=testDb sslmode=required password="},
	// from meta, multiple hosts
	{metadata: map[string]string{"query": "test_query", "targetQueryValue": "5", "host": "host1,host2", "port": "1234", "dbName": "testDb", "userName": "user", "sslmode": "required"}, connectionString: "host=host1,host2 port=1234 user=user dbname=testDb sslmode=required password="},
}

func TestPostgreSQLConnectionStringGeneration(t *testing.T) {
	for _, testData := range testPostgreSQLConnectionstring {
		meta, _, err := parsePostgreSQLMetadata(logr.Discard(), &scalersconfig.ScalerConfig{ResolvedEnv: testData.resolvedEnv, TriggerMetadata: testData.metadata, AuthParams: testData.authParam, TriggerIndex: 0})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}

		if meta.Connection != testData.connectionString {
			t.Errorf("Error generating connectionString, expected '%s' and get '%s'", testData.connectionString, meta.Connection)
		}
	}
}

func TestEscapePostgreConnectionParameter(t *testing.T) {
	tests := map[string]struct {
		input string
		want  string
	}{
		"simple":              {input: "simple", want: "simple"},
		"space":               {input: "with space", want: "'with space'"},
		"tab":                 {input: "user\tsslmode=disable", want: "'user\tsslmode=disable'"},
		"newline":             {input: "user\nhost=attacker", want: "'user\nhost=attacker'"},
		"single quote":        {input: "user'name", want: "'user\\'name'"},
		"backslash":           {input: `user\name`, want: `'user\\name'`},
		"quote and backslash": {input: `user\'name`, want: `'user\\\'name'`},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := escapePostgreConnectionParameter(tt.input); got != tt.want {
				t.Errorf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestPostgreSQLConnectionStringGenerationEscapesWhitespace(t *testing.T) {
	meta, _, err := parsePostgreSQLMetadata(logr.Discard(), &scalersconfig.ScalerConfig{
		TriggerMetadata: map[string]string{
			"query":            "test_query",
			"targetQueryValue": "5",
			"host":             "db.example",
			"port":             "5432",
			"dbName":           "testdb",
			"userName":         "user\tsslmode=disable",
			"sslmode":          "require",
		},
		AuthParams: map[string]string{
			"password": "secret\nhost=attacker",
		},
	})
	if err != nil {
		t.Fatal("Could not parse metadata:", err)
	}

	expected := "host=db.example port=5432 user='user\tsslmode=disable' dbname=testdb sslmode=require password='secret\nhost=attacker'"
	if meta.Connection != expected {
		t.Errorf("Error generating connectionString, expected %q and got %q", expected, meta.Connection)
	}
}

var testPodIdentityAzureWorkloadPostgreSQLConnectionstring = []postgreSQLConnectionStringTestData{
	// from meta
	{metadata: map[string]string{"query": "test_query", "targetQueryValue": "5", "host": "localhost", "port": "1234", "dbName": "testDb", "userName": "user", "sslmode": "required"}, connectionString: "host=localhost port=1234 user=user dbname=testDb sslmode=required %PASSWORD%"},
}

func TestPodIdentityAzureWorkloadPostgreSQLConnectionStringGeneration(t *testing.T) {
	t.Setenv("AZURE_FEDERATED_TOKEN_FILE", "/dummy/token")
	t.Setenv("AZURE_CLIENT_ID", "dummy-client")
	t.Setenv("AZURE_TENANT_ID", "dummy-tenant")

	identityID := "IDENTITY_ID_CORRESPONDING_TO_USERNAME_FIELD"
	for _, testData := range testPodIdentityAzureWorkloadPostgreSQLConnectionstring {
		meta, _, err := parsePostgreSQLMetadata(logr.Discard(), &scalersconfig.ScalerConfig{ResolvedEnv: testData.resolvedEnv, TriggerMetadata: testData.metadata, PodIdentity: kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderAzureWorkload, IdentityID: &identityID}, AuthParams: testData.authParam, TriggerIndex: 0})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}

		if meta.Connection != testData.connectionString {
			t.Errorf("Error generating connectionString, expected '%s' and get '%s'", testData.connectionString, meta.Connection)
		}
	}
}

type parsePostgresMetadataTestData struct {
	metadata    map[string]string
	authParams  map[string]string
	resolvedEnv map[string]string
	raisesError bool
}

var testPostgresResolvedEnv = map[string]string{
	"POSTGRE_PASSWORD": "pass",
	"POSTGRE_CONN_STR": "test_conn_str",
}

var testPostgresMetadata = []parsePostgresMetadataTestData{
	// No metadata
	{
		metadata:    map[string]string{},
		authParams:  map[string]string{},
		resolvedEnv: map[string]string{},
		raisesError: true,
	},
	// connectionString
	{
		metadata:    map[string]string{"query": "query", "targetQueryValue": "12", "connectionFromEnv": "POSTGRE_CONN_STR"},
		authParams:  map[string]string{},
		resolvedEnv: testPostgresResolvedEnv,
		raisesError: false,
	},
	// Params instead of conn str
	{
		metadata:    map[string]string{"query": "query", "targetQueryValue": "12", "host": "test_host", "port": "test_port", "userName": "test_username", "passwordFromEnv": "POSTGRE_PASSWORD", "dbName": "test_dbname", "sslmode": "require"},
		authParams:  map[string]string{},
		resolvedEnv: testPostgresResolvedEnv,
		raisesError: false,
	},
	// Params from trigger authentication
	{
		metadata:    map[string]string{"query": "query", "targetQueryValue": "12"},
		authParams:  map[string]string{"host": "test_host", "port": "test_port", "userName": "test_username", "password": "POSTGRE_PASSWORD", "dbName": "test_dbname", "sslmode": "disable"},
		resolvedEnv: testPostgresResolvedEnv,
		raisesError: false,
	},
}

func TestParsePostgreSQLMetadata(t *testing.T) {
	for _, testData := range testPostgresMetadata {
		_, _, err := parsePostgreSQLMetadata(logr.Discard(), &scalersconfig.ScalerConfig{ResolvedEnv: testData.resolvedEnv, TriggerMetadata: testData.metadata, AuthParams: testData.authParams})
		if err != nil && !testData.raisesError {
			t.Error("Expected success but got error", err)
		}
		if err == nil && testData.raisesError {
			t.Error("Expected error but got success")
		}
	}
}

// testCAPEM returns a self-signed CA certificate in PEM format
func testCAPEM(t *testing.T) []byte {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test-rds-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

// writeTestCABundle writes a self-signed CA certificate in PEM format and returns its path
func writeTestCABundle(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "rds-ca.pem")
	if err := os.WriteFile(path, testCAPEM(t), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func awsRDSTriggerMetadata(overrides map[string]string) map[string]string {
	metadata := map[string]string{
		"query":            "SELECT 1",
		"targetQueryValue": "5",
		"host":             "mydb.abc123.us-east-2.rds.amazonaws.com",
		"port":             "5432",
		"userName":         "keda_user",
		"dbName":           "mydb",
		"sslmode":          "verify-full",
		"awsRegion":        "us-east-2",
	}
	for k, v := range overrides {
		if v == "" {
			delete(metadata, k)
			continue
		}
		metadata[k] = v
	}
	return metadata
}

func TestParsePostgreSQLMetadataAwsIAM(t *testing.T) {
	caBundle := writeTestCABundle(t)
	caPEM := string(testCAPEM(t))
	roleArn := "arn:aws:iam::123456789012:role/keda-rds"
	keda := "keda"
	workload := "workload"

	tests := map[string]struct {
		metadata    map[string]string
		authParams  map[string]string
		resolvedEnv map[string]string
		podIdentity kedav1alpha1.AuthPodIdentity
		defaultCA   string
		wantErr     string
		wantConn    string
	}{
		"roleArn on pod identity": {
			metadata:    awsRDSTriggerMetadata(map[string]string{"sslrootcert": caBundle}),
			podIdentity: kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderAws, RoleArn: &roleArn},
			wantConn:    "host=mydb.abc123.us-east-2.rds.amazonaws.com port=5432 user=keda_user dbname=mydb sslmode=verify-full sslrootcert=" + caBundle,
		},
		"roleArn resolved from workload identity owner": {
			metadata:    awsRDSTriggerMetadata(map[string]string{"sslrootcert": caBundle}),
			authParams:  map[string]string{"awsRoleArn": roleArn},
			podIdentity: kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderAws, IdentityOwner: &workload},
		},
		"explicit keda identity owner": {
			metadata:    awsRDSTriggerMetadata(map[string]string{"sslrootcert": caBundle}),
			podIdentity: kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderAws, IdentityOwner: &keda},
		},
		"connection params from trigger authentication": {
			metadata:    map[string]string{"query": "SELECT 1", "targetQueryValue": "5"},
			authParams:  map[string]string{"host": "mydb", "port": "5432", "userName": "keda_user", "dbName": "mydb", "sslmode": "verify-full", "sslrootcert": caBundle, "awsRegion": "us-east-2"},
			podIdentity: kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderAws, RoleArn: &roleArn},
			wantConn:    "host=mydb port=5432 user=keda_user dbname=mydb sslmode=verify-full sslrootcert=" + caBundle,
		},
		"default CA bundle path": {
			metadata:    awsRDSTriggerMetadata(nil),
			podIdentity: kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderAws, RoleArn: &roleArn},
			defaultCA:   caBundle,
			wantConn:    "host=mydb.abc123.us-east-2.rds.amazonaws.com port=5432 user=keda_user dbname=mydb sslmode=verify-full sslrootcert=" + caBundle,
		},
		"ca PEM from trigger authentication": {
			metadata:    awsRDSTriggerMetadata(nil),
			authParams:  map[string]string{"ca": caPEM},
			podIdentity: kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderAws, RoleArn: &roleArn},
			defaultCA:   filepath.Join(t.TempDir(), "missing.pem"),
			wantConn:    "host=mydb.abc123.us-east-2.rds.amazonaws.com port=5432 user=keda_user dbname=mydb sslmode=verify-full",
		},
		"ca and sslrootcert both set": {
			metadata:    awsRDSTriggerMetadata(map[string]string{"sslrootcert": caBundle}),
			authParams:  map[string]string{"ca": caPEM},
			podIdentity: kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderAws, RoleArn: &roleArn},
			wantErr:     "ca and sslrootcert are mutually exclusive",
		},
		"ca without valid certificates": {
			metadata:    awsRDSTriggerMetadata(nil),
			authParams:  map[string]string{"ca": "not a certificate"},
			podIdentity: kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderAws, RoleArn: &roleArn},
			wantErr:     "ca does not contain any valid PEM encoded certificates",
		},
		"no role and no explicit identity owner": {
			metadata:    awsRDSTriggerMetadata(map[string]string{"sslrootcert": caBundle}),
			podIdentity: kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderAws},
			wantErr:     "requires either roleArn or identityOwner",
		},
		"default CA bundle missing": {
			metadata:    awsRDSTriggerMetadata(nil),
			podIdentity: kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderAws, RoleArn: &roleArn},
			defaultCA:   filepath.Join(t.TempDir(), "missing.pem"),
			wantErr:     "RDS CA bundle not found",
		},
		"sslrootcert missing": {
			metadata:    awsRDSTriggerMetadata(map[string]string{"sslrootcert": filepath.Join(t.TempDir(), "missing.pem")}),
			podIdentity: kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderAws, RoleArn: &roleArn},
			wantErr:     "RDS CA bundle not found",
		},
		"sslmode not verify-full": {
			metadata:    awsRDSTriggerMetadata(map[string]string{"sslmode": "require", "sslrootcert": caBundle}),
			podIdentity: kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderAws, RoleArn: &roleArn},
			wantErr:     "sslmode must be verify-full",
		},
		"missing awsRegion": {
			metadata:    awsRDSTriggerMetadata(map[string]string{"awsRegion": "", "sslrootcert": caBundle}),
			podIdentity: kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderAws, RoleArn: &roleArn},
			wantErr:     "no awsRegion given",
		},
		"password set": {
			metadata:    awsRDSTriggerMetadata(map[string]string{"sslrootcert": caBundle}),
			authParams:  map[string]string{"password": "secret"},
			podIdentity: kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderAws, RoleArn: &roleArn},
			wantErr:     "password must not be set",
		},
		"connection string set": {
			metadata:    map[string]string{"query": "SELECT 1", "targetQueryValue": "5", "awsRegion": "us-east-2"},
			authParams:  map[string]string{"connection": "postgresql://keda_user@mydb:5432/mydb"},
			podIdentity: kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderAws, RoleArn: &roleArn},
			wantErr:     "connection is not supported",
		},
		"multiple hosts": {
			metadata:    awsRDSTriggerMetadata(map[string]string{"host": "host1,host2", "sslrootcert": caBundle}),
			podIdentity: kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderAws, RoleArn: &roleArn},
			wantErr:     "multiple hosts are not supported",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if tt.defaultCA != "" {
				previous := defaultRDSRootCertPath
				defaultRDSRootCertPath = tt.defaultCA
				t.Cleanup(func() { defaultRDSRootCertPath = previous })
			}

			meta, podIdentity, err := parsePostgreSQLMetadata(logr.Discard(), &scalersconfig.ScalerConfig{
				TriggerMetadata:  tt.metadata,
				AuthParams:       tt.authParams,
				ResolvedEnv:      tt.resolvedEnv,
				PodIdentity:      tt.podIdentity,
				TriggerUniqueKey: "test-" + name,
			})
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatal("Could not parse metadata:", err)
			}
			if podIdentity.Provider != kedav1alpha1.PodIdentityProviderAws {
				t.Errorf("expected aws pod identity, got %q", podIdentity.Provider)
			}
			if !meta.awsAuthorization.UsingPodIdentity {
				t.Error("expected AWS authorization to use pod identity")
			}
			if strings.Contains(meta.Connection, "password") {
				t.Errorf("connection string must not contain a password, got %q", meta.Connection)
			}
			if tt.wantConn != "" && meta.Connection != tt.wantConn {
				t.Errorf("expected connection string %q, got %q", tt.wantConn, meta.Connection)
			}
		})
	}
}

func TestPostgreSQLAwsIAMSignsTokenPerConnection(t *testing.T) {
	// Reserve a local port, close it, and then attempt to connect to it to trigger the BeforeConnect hook
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	host, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	listener.Close()

	roleArn := "arn:aws:iam::123456789012:role/keda-rds"
	meta, _, err := parsePostgreSQLMetadata(logr.Discard(), &scalersconfig.ScalerConfig{
		TriggerMetadata:  awsRDSTriggerMetadata(map[string]string{"host": host, "port": port, "sslrootcert": writeTestCABundle(t)}),
		PodIdentity:      kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderAws, RoleArn: &roleArn},
		TriggerUniqueKey: "test-token-per-connection",
	})
	if err != nil {
		t.Fatal("Could not parse metadata:", err)
	}
	t.Cleanup(func() { awsutils.ClearAwsConfig(meta.awsAuthorization) })

	var calls []string
	previous := buildRDSAuthToken
	buildRDSAuthToken = func(_ context.Context, endpoint, region, dbUser string, _ aws.CredentialsProvider, _ ...func(*rdsauth.BuildAuthTokenOptions)) (string, error) {
		calls = append(calls, endpoint+"|"+region+"|"+dbUser)
		return "signed-token", nil
	}
	t.Cleanup(func() { buildRDSAuthToken = previous })

	db, err := openAwsRDSConnection(context.Background(), meta)
	if err != nil {
		t.Fatal("Could not open connection:", err)
	}
	defer db.Close()

	for range 2 {
		if err := db.PingContext(context.Background()); err == nil {
			t.Fatal("expected ping to fail against a closed port")
		}
	}

	want := net.JoinHostPort(host, port) + "|us-east-2|keda_user"
	if len(calls) != 2 {
		t.Fatalf("expected a token to be signed for each of 2 connection attempts, got %d", len(calls))
	}
	for _, call := range calls {
		if call != want {
			t.Errorf("expected token request %q, got %q", want, call)
		}
	}
}

func TestPostgreSQLAwsIAMUsesCAFromAuthParams(t *testing.T) {
	roleArn := "arn:aws:iam::123456789012:role/keda-rds"
	meta, _, err := parsePostgreSQLMetadata(logr.Discard(), &scalersconfig.ScalerConfig{
		TriggerMetadata:  awsRDSTriggerMetadata(nil),
		AuthParams:       map[string]string{"ca": string(testCAPEM(t))},
		PodIdentity:      kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderAws, RoleArn: &roleArn},
		TriggerUniqueKey: "test-ca-from-auth-params",
	})
	if err != nil {
		t.Fatal("Could not parse metadata:", err)
	}
	if meta.rdsRootCAs == nil {
		t.Fatal("expected the ca auth param to be parsed into a certificate pool")
	}

	connConfig, err := awsRDSConnConfig(meta)
	if err != nil {
		t.Fatal("Could not build connection config:", err)
	}
	if connConfig.TLSConfig == nil {
		t.Fatal("expected TLS to be configured for sslmode=verify-full")
	}
	if connConfig.TLSConfig.RootCAs != meta.rdsRootCAs {
		t.Error("expected the server certificate to be verified against the ca auth param")
	}
	if connConfig.TLSConfig.ServerName != "mydb.abc123.us-east-2.rds.amazonaws.com" {
		t.Errorf("expected the server name to be verified, got %q", connConfig.TLSConfig.ServerName)
	}
}

func TestPostgreSQLAwsIAMConnectionHonoursContext(t *testing.T) {
	roleArn := "arn:aws:iam::123456789012:role/keda-rds"
	meta, podIdentity, err := parsePostgreSQLMetadata(logr.Discard(), &scalersconfig.ScalerConfig{
		TriggerMetadata:  awsRDSTriggerMetadata(map[string]string{"sslrootcert": writeTestCABundle(t)}),
		PodIdentity:      kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderAws, RoleArn: &roleArn},
		TriggerUniqueKey: "test-cancelled-context",
	})
	if err != nil {
		t.Fatal("Could not parse metadata:", err)
	}
	t.Cleanup(func() { awsutils.ClearAwsConfig(meta.awsAuthorization) })

	signed := false
	previous := buildRDSAuthToken
	buildRDSAuthToken = func(context.Context, string, string, string, aws.CredentialsProvider, ...func(*rdsauth.BuildAuthTokenOptions)) (string, error) {
		signed = true
		return "signed-token", nil
	}
	t.Cleanup(func() { buildRDSAuthToken = previous })

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = getConnection(ctx, meta, podIdentity, logr.Discard())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if signed {
		t.Error("expected no token to be signed after the context was cancelled")
	}
}
