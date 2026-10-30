package scalers

import (
	"context"
	"math"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"cloud.google.com/go/firestore/apiv1/firestorepb"
	"github.com/go-logr/logr"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	v2 "k8s.io/api/autoscaling/v2"

	"github.com/kedacore/keda/v2/pkg/scalers/gcp"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

var testFirestoreResolvedEnv = map[string]string{
	"CREDS_ENV":      `{"type":"service_account","project_id":"my-project"}`,
	"CREDS_FILE_ENV": "/var/secrets/sa.json",
}

var firestoreRequiredFields = map[string]string{
	"projectId":  "my-project",
	"collection": "jobs",
}

var firestoreCredentials = map[string]string{"GoogleApplicationCredentials": `{"type":"service_account"}`}

func firestoreWith(overrides map[string]string) map[string]string {
	return withOverrides(firestoreRequiredFields, overrides)
}

// ── a fake Firestore backend ──────────────────────────────────────────────────

type firestoreReply func(ctx context.Context, req *firestorepb.RunAggregationQueryRequest) (*firestorepb.RunAggregationQueryResponse, error)

// fakeFirestore answers RunAggregationQuery as a test says and records the requests, so that the
// scaler is tested through the real Firestore client without a backend.
type fakeFirestore struct {
	firestorepb.UnimplementedFirestoreServer

	mu       sync.Mutex
	requests []*firestorepb.RunAggregationQueryRequest
	// reply decides the answer. A nil response ends the stream with no result. The default is a count of 0.
	reply firestoreReply
}

func (f *fakeFirestore) RunAggregationQuery(req *firestorepb.RunAggregationQueryRequest, stream firestorepb.Firestore_RunAggregationQueryServer) error {
	f.mu.Lock()
	f.requests = append(f.requests, req)
	reply := f.reply
	f.mu.Unlock()

	if reply == nil {
		reply = countReply(0)
	}
	resp, err := reply(stream.Context(), req)
	if err != nil || resp == nil {
		return err
	}
	return stream.Send(resp)
}

func (f *fakeFirestore) setReply(reply firestoreReply) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reply = reply
}

func (f *fakeFirestore) lastRequest(t *testing.T) *firestorepb.RunAggregationQueryRequest {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.requests) == 0 {
		t.Fatal("the backend received no request")
	}
	return f.requests[len(f.requests)-1]
}

func aggregationResponse(req *firestorepb.RunAggregationQueryRequest, value *firestorepb.Value) *firestorepb.RunAggregationQueryResponse {
	alias := req.GetStructuredAggregationQuery().GetAggregations()[0].GetAlias()
	return &firestorepb.RunAggregationQueryResponse{
		Result: &firestorepb.AggregationResult{AggregateFields: map[string]*firestorepb.Value{alias: value}},
	}
}

// countReply answers with an integer count.
func countReply(count int64) firestoreReply {
	return func(_ context.Context, req *firestorepb.RunAggregationQueryRequest) (*firestorepb.RunAggregationQueryResponse, error) {
		return aggregationResponse(req, &firestorepb.Value{ValueType: &firestorepb.Value_IntegerValue{IntegerValue: count}}), nil
	}
}

// startFakeFirestore serves the fake backend on a local port and points the Firestore client at it with
// FIRESTORE_EMULATOR_HOST, the variable the SDK reads for the emulator.
func startFakeFirestore(t *testing.T) *fakeFirestore {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listening: %v", err)
	}
	fake := &fakeFirestore{}
	server := grpc.NewServer()
	firestorepb.RegisterFirestoreServer(server, fake)
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(server.Stop)
	t.Setenv("FIRESTORE_EMULATOR_HOST", listener.Addr().String())
	return fake
}

// newTestFirestoreScaler creates the real scaler against the fake backend.
func newTestFirestoreScaler(t *testing.T, metadata map[string]string) *gcpFirestoreScaler {
	t.Helper()
	scaler, err := NewGcpFirestoreScaler(&scalersconfig.ScalerConfig{
		TriggerMetadata: firestoreWith(metadata),
		AuthParams:      map[string]string{},
		ResolvedEnv:     testFirestoreResolvedEnv,
	})
	if err != nil {
		t.Fatalf("NewGcpFirestoreScaler: %v", err)
	}
	t.Cleanup(func() { _ = scaler.Close(context.Background()) })
	return scaler.(*gcpFirestoreScaler)
}

// countDocuments makes one count with the given metadata, so that the backend records the query.
func countDocuments(t *testing.T, metadata map[string]string) {
	t.Helper()
	if _, err := newTestFirestoreScaler(t, metadata).count(context.Background()); err != nil {
		t.Fatalf("count: %v", err)
	}
}

// ── metadata ──────────────────────────────────────────────────────────────────

type parseFirestoreMetadataTestData struct {
	desc       string
	authParams map[string]string
	metadata   map[string]string
	isError    bool
}

var testFirestoreMetadataCases = []parseFirestoreMetadataTestData{
	// required fields
	{desc: "all required fields missing", authParams: map[string]string{}, metadata: map[string]string{}, isError: true},
	{desc: "missing projectId", authParams: firestoreCredentials, metadata: withoutKey(firestoreRequiredFields, "projectId"), isError: true},
	{desc: "missing collection", authParams: firestoreCredentials, metadata: withoutKey(firestoreRequiredFields, "collection"), isError: true},

	// authentication methods
	{desc: "credentials inline via authParams", authParams: firestoreCredentials, metadata: firestoreRequiredFields},
	{desc: "credentials from env var (credentialsFromEnv)", authParams: map[string]string{}, metadata: firestoreWith(map[string]string{"credentialsFromEnv": "CREDS_ENV"})},
	{desc: "credentials from env file path (credentialsFromEnvFile)", authParams: map[string]string{}, metadata: firestoreWith(map[string]string{"credentialsFromEnvFile": "CREDS_FILE_ENV"})},
	{desc: "no credentials provided (no pod identity)", authParams: map[string]string{}, metadata: firestoreRequiredFields, isError: true},

	// target and activation values
	{desc: "custom targetValue and activationValue", authParams: firestoreCredentials, metadata: firestoreWith(map[string]string{"targetValue": "20", "activationValue": "3"})},
	{desc: "targetValue zero is invalid", authParams: firestoreCredentials, metadata: firestoreWith(map[string]string{"targetValue": "0"}), isError: true},
	{desc: "targetValue negative is invalid", authParams: firestoreCredentials, metadata: firestoreWith(map[string]string{"targetValue": "-1"}), isError: true},

	// collection
	{desc: "subcollection path", authParams: firestoreCredentials, metadata: firestoreWith(map[string]string{"collection": "users/alice/jobs"})},
	{desc: "collection path of a document", authParams: firestoreCredentials, metadata: firestoreWith(map[string]string{"collection": "users/alice"}), isError: true},
	{desc: "collection with a leading slash", authParams: firestoreCredentials, metadata: firestoreWith(map[string]string{"collection": "/jobs"}), isError: true},
	{desc: "collection with a trailing slash", authParams: firestoreCredentials, metadata: firestoreWith(map[string]string{"collection": "jobs/"}), isError: true},
	{desc: "collection with an empty segment", authParams: firestoreCredentials, metadata: firestoreWith(map[string]string{"collection": "users//jobs"}), isError: true},
	{desc: "collection that is a dot", authParams: firestoreCredentials, metadata: firestoreWith(map[string]string{"collection": "."}), isError: true},
	{desc: "collection that is two dots", authParams: firestoreCredentials, metadata: firestoreWith(map[string]string{"collection": ".."}), isError: true},
	{desc: "collection id reserved by Firestore", authParams: firestoreCredentials, metadata: firestoreWith(map[string]string{"collection": "__jobs__"}), isError: true},
	{desc: "reserved segment inside a path", authParams: firestoreCredentials, metadata: firestoreWith(map[string]string{"collection": "users/__alice__/jobs"}), isError: true},
	{desc: "collection that only starts with underscores", authParams: firestoreCredentials, metadata: firestoreWith(map[string]string{"collection": "__jobs"})},

	// filter
	{desc: "filter with the default operator", authParams: firestoreCredentials, metadata: firestoreWith(map[string]string{"filterField": "status", "filterValue": "pending"})},
	{desc: "nested filter field", authParams: firestoreCredentials, metadata: firestoreWith(map[string]string{"filterField": "owner.name", "filterValue": "alice"})},
	{desc: "filter operator that takes a list is not supported", authParams: firestoreCredentials, metadata: firestoreWith(map[string]string{"filterField": "status", "filterOperator": "in", "filterValue": "pending"}), isError: true},
	{desc: "filter operator that is not an operator", authParams: firestoreCredentials, metadata: firestoreWith(map[string]string{"filterField": "status", "filterOperator": "like", "filterValue": "pending"}), isError: true},
	{desc: "filterField without filterValue", authParams: firestoreCredentials, metadata: firestoreWith(map[string]string{"filterField": "status"}), isError: true},
	{desc: "filterValue without filterField", authParams: firestoreCredentials, metadata: firestoreWith(map[string]string{"filterValue": "pending"}), isError: true},
	{desc: "filterOperator without filterField", authParams: firestoreCredentials, metadata: firestoreWith(map[string]string{"filterOperator": "=="}), isError: true},
	{desc: "filterValueType that is not a type", authParams: firestoreCredentials, metadata: firestoreWith(map[string]string{"filterField": "status", "filterValue": "pending", "filterValueType": "date"}), isError: true},
	{desc: "filterValueType that is a list of types", authParams: firestoreCredentials, metadata: firestoreWith(map[string]string{"filterField": "n", "filterValue": "3", "filterValueType": "int,float"}), isError: true},
	{desc: "filterValueType that is a list of types without a filter", authParams: firestoreCredentials, metadata: firestoreWith(map[string]string{"filterValueType": "int,float"}), isError: true},
	{desc: "filterValueType in another case", authParams: firestoreCredentials, metadata: firestoreWith(map[string]string{"filterField": "n", "filterValue": "3", "filterValueType": "Int"}), isError: true},
	{desc: "filterValue that is not an integer", authParams: firestoreCredentials, metadata: firestoreWith(map[string]string{"filterField": "priority", "filterValue": "high", "filterValueType": "int"}), isError: true},
	{desc: "filterField with a slash", authParams: firestoreCredentials, metadata: firestoreWith(map[string]string{"filterField": "a/b", "filterValue": "x"}), isError: true},
	{desc: "filterField with a bracket", authParams: firestoreCredentials, metadata: firestoreWith(map[string]string{"filterField": "a[0]", "filterValue": "x"}), isError: true},
	{desc: "filterField with a star", authParams: firestoreCredentials, metadata: firestoreWith(map[string]string{"filterField": "a*", "filterValue": "x"}), isError: true},
	{desc: "filterField with a tilde", authParams: firestoreCredentials, metadata: firestoreWith(map[string]string{"filterField": "a~b", "filterValue": "x"}), isError: true},
	{desc: "filterField with an empty name", authParams: firestoreCredentials, metadata: firestoreWith(map[string]string{"filterField": "a..b", "filterValue": "x"}), isError: true},
	{desc: "filterField with a trailing dot", authParams: firestoreCredentials, metadata: firestoreWith(map[string]string{"filterField": "a.", "filterValue": "x"}), isError: true},
	{desc: "filterField reserved by Firestore", authParams: firestoreCredentials, metadata: firestoreWith(map[string]string{"filterField": "__name__", "filterValue": "x"}), isError: true},
	{desc: "NaN compared with equals", authParams: firestoreCredentials, metadata: firestoreWith(map[string]string{"filterField": "score", "filterValue": "NaN", "filterValueType": "float"})},
	{desc: "NaN compared with not equals", authParams: firestoreCredentials, metadata: firestoreWith(map[string]string{"filterField": "score", "filterOperator": "!=", "filterValue": "NaN", "filterValueType": "float"})},
	{desc: "NaN compared with less than", authParams: firestoreCredentials, metadata: firestoreWith(map[string]string{"filterField": "score", "filterOperator": "<", "filterValue": "NaN", "filterValueType": "float"}), isError: true},
}

func TestFirestoreParseMetadata(t *testing.T) {
	for _, tc := range testFirestoreMetadataCases {
		t.Run(tc.desc, func(t *testing.T) {
			// A developer who runs the integration tests has the emulator host exported; it would hide the credentials.
			t.Setenv("FIRESTORE_EMULATOR_HOST", "")
			_, err := parseGcpFirestoreMetadata(&scalersconfig.ScalerConfig{
				AuthParams:      tc.authParams,
				TriggerMetadata: tc.metadata,
				ResolvedEnv:     testFirestoreResolvedEnv,
			})
			if tc.isError && err == nil {
				t.Error("expected error, got nil")
			}
			if !tc.isError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestFirestoreParseMetadata_Defaults verifies the values that are used when the optional fields are omitted.
func TestFirestoreParseMetadata_Defaults(t *testing.T) {
	t.Setenv("FIRESTORE_EMULATOR_HOST", "")
	meta, err := parseGcpFirestoreMetadata(&scalersconfig.ScalerConfig{
		AuthParams:      firestoreCredentials,
		TriggerMetadata: firestoreRequiredFields,
		ResolvedEnv:     testFirestoreResolvedEnv,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.DatabaseID != "(default)" {
		t.Errorf("DatabaseID default: got %q, want %q", meta.DatabaseID, "(default)")
	}
	if meta.TargetValue != 5 {
		t.Errorf("TargetValue default: got %v, want 5", meta.TargetValue)
	}
	if meta.ActivationValue != 0 {
		t.Errorf("ActivationValue default: got %v, want 0", meta.ActivationValue)
	}
	if meta.FilterField != "" || meta.filterValue != nil {
		t.Errorf("no filter expected, got field %q and value %v", meta.FilterField, meta.filterValue)
	}
}

// TestFirestoreParseMetadata_Filter verifies the operator default and the type of the filter value.
func TestFirestoreParseMetadata_Filter(t *testing.T) {
	cases := []struct {
		desc         string
		metadata     map[string]string
		wantOperator string
		wantValue    interface{}
	}{
		{
			desc:         "string by default, operator defaults to ==",
			metadata:     map[string]string{"filterField": "status", "filterValue": "pending"},
			wantOperator: "==",
			wantValue:    "pending",
		},
		{
			desc:         "a string that looks like a number stays a string",
			metadata:     map[string]string{"filterField": "code", "filterValue": "42"},
			wantOperator: "==",
			wantValue:    "42",
		},
		{
			desc:         "integer",
			metadata:     map[string]string{"filterField": "priority", "filterOperator": ">", "filterValue": "3", "filterValueType": "int"},
			wantOperator: ">",
			wantValue:    int64(3),
		},
		{
			desc:         "float",
			metadata:     map[string]string{"filterField": "score", "filterOperator": "<=", "filterValue": "0.5", "filterValueType": "float"},
			wantOperator: "<=",
			wantValue:    0.5,
		},
		{
			desc:         "boolean",
			metadata:     map[string]string{"filterField": "done", "filterOperator": "!=", "filterValue": "true", "filterValueType": "bool"},
			wantOperator: "!=",
			wantValue:    true,
		},
		{
			desc:         "array-contains",
			metadata:     map[string]string{"filterField": "tags", "filterOperator": "array-contains", "filterValue": "urgent"},
			wantOperator: "array-contains",
			wantValue:    "urgent",
		},
		{
			desc:         "the value is trimmed like every other metadata value",
			metadata:     map[string]string{"filterField": "status", "filterValue": " pending "},
			wantOperator: "==",
			wantValue:    "pending",
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			t.Setenv("FIRESTORE_EMULATOR_HOST", "")
			meta, err := parseGcpFirestoreMetadata(&scalersconfig.ScalerConfig{
				AuthParams:      firestoreCredentials,
				TriggerMetadata: firestoreWith(tc.metadata),
				ResolvedEnv:     testFirestoreResolvedEnv,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if meta.FilterOperator != tc.wantOperator {
				t.Errorf("operator: got %q, want %q", meta.FilterOperator, tc.wantOperator)
			}
			if meta.filterValue != tc.wantValue {
				t.Errorf("value: got %v (%T), want %v (%T)", meta.filterValue, meta.filterValue, tc.wantValue, tc.wantValue)
			}
		})
	}
}

func TestFirestoreParseMetadata_NaN(t *testing.T) {
	t.Setenv("FIRESTORE_EMULATOR_HOST", "")
	meta, err := parseGcpFirestoreMetadata(&scalersconfig.ScalerConfig{
		AuthParams:      firestoreCredentials,
		TriggerMetadata: firestoreWith(map[string]string{"filterField": "score", "filterValue": "NaN", "filterValueType": "float"}),
		ResolvedEnv:     testFirestoreResolvedEnv,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if number, ok := meta.filterValue.(float64); !ok || !math.IsNaN(number) {
		t.Errorf("value: got %v (%T), want NaN", meta.filterValue, meta.filterValue)
	}
}

// TestFirestoreParseMetadata_CredentialsFromEnvFile verifies that the file-path credential method is stored.
func TestFirestoreParseMetadata_CredentialsFromEnvFile(t *testing.T) {
	t.Setenv("FIRESTORE_EMULATOR_HOST", "")
	meta, err := parseGcpFirestoreMetadata(&scalersconfig.ScalerConfig{
		AuthParams:      map[string]string{},
		TriggerMetadata: firestoreWith(map[string]string{"credentialsFromEnvFile": "CREDS_FILE_ENV"}),
		ResolvedEnv:     testFirestoreResolvedEnv,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "/var/secrets/sa.json"
	if meta.gcpAuthorization.GoogleApplicationCredentialsFile != want {
		t.Errorf("CredentialsFile: got %q, want %q", meta.gcpAuthorization.GoogleApplicationCredentialsFile, want)
	}
}

// TestFirestoreParseMetadata_CredentialsFromEnv verifies that the inline credential is taken from the environment.
func TestFirestoreParseMetadata_CredentialsFromEnv(t *testing.T) {
	t.Setenv("FIRESTORE_EMULATOR_HOST", "")
	meta, err := parseGcpFirestoreMetadata(&scalersconfig.ScalerConfig{
		AuthParams:      map[string]string{},
		TriggerMetadata: firestoreWith(map[string]string{"credentialsFromEnv": "CREDS_ENV"}),
		ResolvedEnv:     testFirestoreResolvedEnv,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.gcpAuthorization.GoogleApplicationCredentials != testFirestoreResolvedEnv["CREDS_ENV"] {
		t.Errorf("Credentials: got %q, want %q", meta.gcpAuthorization.GoogleApplicationCredentials, testFirestoreResolvedEnv["CREDS_ENV"])
	}
}

// TestFirestoreParseMetadata_Emulator verifies that no credentials are needed when the emulator host is set.
func TestFirestoreParseMetadata_Emulator(t *testing.T) {
	t.Setenv("FIRESTORE_EMULATOR_HOST", "localhost:8080")
	meta, err := parseGcpFirestoreMetadata(&scalersconfig.ScalerConfig{
		AuthParams:      map[string]string{},
		TriggerMetadata: firestoreRequiredFields,
		ResolvedEnv:     testFirestoreResolvedEnv,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.gcpAuthorization == nil {
		t.Error("expected an empty authorization, got nil")
	}
}

func TestFirestoreParseFilterValue(t *testing.T) {
	cases := []struct {
		valueType string
		raw       string
		want      interface{}
		isError   bool
	}{
		{valueType: "string", raw: "a b", want: "a b"},
		{valueType: "int", raw: "-7", want: int64(-7)},
		{valueType: "int", raw: "7.5", isError: true},
		{valueType: "float", raw: "7", want: float64(7)},
		{valueType: "float", raw: "seven", isError: true},
		{valueType: "bool", raw: "false", want: false},
		{valueType: "bool", raw: "yes", isError: true},
		{valueType: "date", raw: "2026-10-30", isError: true},
	}
	for _, tc := range cases {
		got, err := parseFirestoreFilterValue(tc.valueType, tc.raw)
		if tc.isError {
			if err == nil {
				t.Errorf("%s %q: expected error, got %v", tc.valueType, tc.raw, got)
			}
			continue
		}
		if err != nil || got != tc.want {
			t.Errorf("%s %q: got %v (%T) and error %v; want %v (%T)", tc.valueType, tc.raw, got, got, err, tc.want, tc.want)
		}
	}
}

// ── credentials reach the client ──────────────────────────────────────────────

func TestFirestoreClientCredentials(t *testing.T) {
	// Without the emulator host the SDK uses the credentials that it is given, so a wrong one fails at once.
	t.Setenv("FIRESTORE_EMULATOR_HOST", "")
	cases := []struct {
		desc      string
		auth      *gcp.AuthorizationMetadata
		wantInErr string
	}{
		{desc: "inline credentials that are not JSON", auth: &gcp.AuthorizationMetadata{GoogleApplicationCredentials: "not json"}, wantInErr: "invalid character"},
		{desc: "a credentials file that does not exist", auth: &gcp.AuthorizationMetadata{GoogleApplicationCredentialsFile: "/nonexistent/sa.json"}, wantInErr: "no such file"},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			client, err := newGcpFirestoreClient(context.Background(), &gcpFirestoreMetadata{ProjectID: "my-project", DatabaseID: "(default)", gcpAuthorization: tc.auth})
			if err == nil {
				_ = client.Close()
				t.Fatal("expected an error, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantInErr) {
				t.Errorf("error %q does not contain %q", err, tc.wantInErr)
			}
		})
	}
}

// ── the query that reaches Firestore ──────────────────────────────────────────

func TestFirestoreQuery(t *testing.T) {
	operators := map[string]firestorepb.StructuredQuery_FieldFilter_Operator{
		"==":             firestorepb.StructuredQuery_FieldFilter_EQUAL,
		"!=":             firestorepb.StructuredQuery_FieldFilter_NOT_EQUAL,
		"<":              firestorepb.StructuredQuery_FieldFilter_LESS_THAN,
		"<=":             firestorepb.StructuredQuery_FieldFilter_LESS_THAN_OR_EQUAL,
		">":              firestorepb.StructuredQuery_FieldFilter_GREATER_THAN,
		">=":             firestorepb.StructuredQuery_FieldFilter_GREATER_THAN_OR_EQUAL,
		"array-contains": firestorepb.StructuredQuery_FieldFilter_ARRAY_CONTAINS,
	}
	// Every operator that the scaler accepts is tested, so that a new one is not forgotten.
	if len(operators) != len(firestoreFilterOperators) {
		t.Fatalf("the test knows %d operators, the scaler accepts %d", len(operators), len(firestoreFilterOperators))
	}

	fake := startFakeFirestore(t)

	t.Run("the whole collection of the default database", func(t *testing.T) {
		countDocuments(t, nil)
		req := fake.lastRequest(t)
		if req.GetParent() != "projects/my-project/databases/(default)/documents" {
			t.Errorf("parent: got %q", req.GetParent())
		}
		query := req.GetStructuredAggregationQuery().GetStructuredQuery()
		if len(query.GetFrom()) != 1 || query.GetFrom()[0].GetCollectionId() != "jobs" {
			t.Errorf("from: got %v, want the collection jobs", query.GetFrom())
		}
		if query.GetWhere() != nil {
			t.Errorf("where: got %v, want no filter", query.GetWhere())
		}
		aggregations := req.GetStructuredAggregationQuery().GetAggregations()
		if len(aggregations) != 1 || aggregations[0].GetCount() == nil {
			t.Errorf("aggregations: got %v, want one count", aggregations)
		}
	})

	t.Run("a named database", func(t *testing.T) {
		countDocuments(t, map[string]string{"databaseId": "orders-db"})
		if got := fake.lastRequest(t).GetParent(); got != "projects/my-project/databases/orders-db/documents" {
			t.Errorf("parent: got %q", got)
		}
	})

	t.Run("a subcollection", func(t *testing.T) {
		countDocuments(t, map[string]string{"collection": "users/alice/jobs"})
		req := fake.lastRequest(t)
		if req.GetParent() != "projects/my-project/databases/(default)/documents/users/alice" {
			t.Errorf("parent: got %q", req.GetParent())
		}
		if got := req.GetStructuredAggregationQuery().GetStructuredQuery().GetFrom()[0].GetCollectionId(); got != "jobs" {
			t.Errorf("from: got %q, want jobs", got)
		}
	})

	for operator, want := range operators {
		t.Run("operator "+operator, func(t *testing.T) {
			countDocuments(t, map[string]string{"filterField": "owner.name", "filterOperator": operator, "filterValue": "alice"})
			filter := fake.lastRequest(t).GetStructuredAggregationQuery().GetStructuredQuery().GetWhere().GetFieldFilter()
			if filter == nil {
				t.Fatal("the query has no field filter")
			}
			if filter.GetOp() != want {
				t.Errorf("operator: got %v, want %v", filter.GetOp(), want)
			}
			if filter.GetField().GetFieldPath() != "owner.name" {
				t.Errorf("field: got %q, want owner.name", filter.GetField().GetFieldPath())
			}
			if filter.GetValue().GetStringValue() != "alice" {
				t.Errorf("value: got %v, want the string alice", filter.GetValue())
			}
		})
	}

	t.Run("the default operator is equals", func(t *testing.T) {
		countDocuments(t, map[string]string{"filterField": "status", "filterValue": "pending"})
		if got := fake.lastRequest(t).GetStructuredAggregationQuery().GetStructuredQuery().GetWhere().GetFieldFilter().GetOp(); got != firestorepb.StructuredQuery_FieldFilter_EQUAL {
			t.Errorf("operator: got %v, want equals", got)
		}
	})

	t.Run("the types of the value", func(t *testing.T) {
		cases := []struct {
			valueType, raw string
			check          func(*firestorepb.Value) bool
		}{
			{"string", "3", func(v *firestorepb.Value) bool {
				_, ok := v.GetValueType().(*firestorepb.Value_StringValue)
				return ok && v.GetStringValue() == "3"
			}},
			{"int", "3", func(v *firestorepb.Value) bool {
				_, ok := v.GetValueType().(*firestorepb.Value_IntegerValue)
				return ok && v.GetIntegerValue() == 3
			}},
			{"float", "0.5", func(v *firestorepb.Value) bool {
				_, ok := v.GetValueType().(*firestorepb.Value_DoubleValue)
				return ok && v.GetDoubleValue() == 0.5
			}},
			{"bool", "true", func(v *firestorepb.Value) bool {
				_, ok := v.GetValueType().(*firestorepb.Value_BooleanValue)
				return ok && v.GetBooleanValue()
			}},
		}
		for _, tc := range cases {
			countDocuments(t, map[string]string{"filterField": "f", "filterValue": tc.raw, "filterValueType": tc.valueType})
			if value := fake.lastRequest(t).GetStructuredAggregationQuery().GetStructuredQuery().GetWhere().GetFieldFilter().GetValue(); !tc.check(value) {
				t.Errorf("%s %q: the query carries %v", tc.valueType, tc.raw, value)
			}
		}
	})
}

// ── the count, the activity and the errors ────────────────────────────────────

func TestFirestoreGetMetricsAndActivity(t *testing.T) {
	cases := []struct {
		desc       string
		metadata   map[string]string
		count      int64
		wantMilli  int64
		wantActive bool
	}{
		{desc: "no document is not active", count: 0, wantMilli: 0, wantActive: false},
		{desc: "one document is active with the default activation value", count: 1, wantMilli: 1000, wantActive: true},
		{desc: "a count equal to the activation value is not active", metadata: map[string]string{"activationValue": "3"}, count: 3, wantMilli: 3000, wantActive: false},
		{desc: "a count above the activation value is active", metadata: map[string]string{"activationValue": "3"}, count: 4, wantMilli: 4000, wantActive: true},
		{desc: "a large count", count: 1234567, wantMilli: 1234567000, wantActive: true},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			startFakeFirestore(t).setReply(countReply(tc.count))
			s := newTestFirestoreScaler(t, tc.metadata)

			metrics, active, err := s.GetMetricsAndActivity(context.Background(), "s0-test")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(metrics) != 1 {
				t.Fatalf("expected 1 metric, got %d", len(metrics))
			}
			if metrics[0].MetricName != "s0-test" {
				t.Errorf("metric name: got %q, want %q", metrics[0].MetricName, "s0-test")
			}
			if got := metrics[0].Value.MilliValue(); got != tc.wantMilli {
				t.Errorf("metric value: got %d milli, want %d", got, tc.wantMilli)
			}
			if active != tc.wantActive {
				t.Errorf("active: got %v, want %v", active, tc.wantActive)
			}
		})
	}
}

func TestFirestoreGetMetricsAndActivity_Errors(t *testing.T) {
	notAnInteger := func(_ context.Context, req *firestorepb.RunAggregationQueryRequest) (*firestorepb.RunAggregationQueryResponse, error) {
		return aggregationResponse(req, &firestorepb.Value{ValueType: &firestorepb.Value_DoubleValue{DoubleValue: 2.5}}), nil
	}
	nullValue := func(_ context.Context, req *firestorepb.RunAggregationQueryRequest) (*firestorepb.RunAggregationQueryResponse, error) {
		return aggregationResponse(req, &firestorepb.Value{ValueType: &firestorepb.Value_NullValue{}}), nil
	}
	noResult := func(context.Context, *firestorepb.RunAggregationQueryRequest) (*firestorepb.RunAggregationQueryResponse, error) {
		return nil, nil
	}
	denied := func(context.Context, *firestorepb.RunAggregationQueryRequest) (*firestorepb.RunAggregationQueryResponse, error) {
		return nil, status.Error(codes.PermissionDenied, "missing datastore.entities.list")
	}

	cases := []struct {
		desc     string
		reply    firestoreReply
		wantCode codes.Code
		wantText string
	}{
		{desc: "a permission error of Firestore keeps its code", reply: denied, wantCode: codes.PermissionDenied},
		{desc: "a count that is not an integer is not 0", reply: notAnInteger, wantText: "instead of an integer"},
		{desc: "a null count is not 0", reply: nullValue, wantText: "instead of an integer"},
		{desc: "an answer without a result is not 0", reply: noResult, wantText: "instead of an integer"},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			startFakeFirestore(t).setReply(tc.reply)
			s := newTestFirestoreScaler(t, nil)

			metrics, active, err := s.GetMetricsAndActivity(context.Background(), "s0-test")
			if err == nil {
				t.Fatal("expected an error, got nil")
			}
			if metrics != nil || active {
				t.Errorf("expected no metric and no activity on error, got %v and %v", metrics, active)
			}
			if tc.wantCode != codes.OK && status.Code(err) != tc.wantCode {
				t.Errorf("code: got %v, want %v (error %v)", status.Code(err), tc.wantCode, err)
			}
			if tc.wantText != "" && !strings.Contains(err.Error(), tc.wantText) {
				t.Errorf("error %q does not contain %q", err, tc.wantText)
			}
		})
	}
}

func TestFirestoreGetMetricsAndActivity_Deadlines(t *testing.T) {
	startFakeFirestore(t).setReply(func(ctx context.Context, _ *firestorepb.RunAggregationQueryRequest) (*firestorepb.RunAggregationQueryResponse, error) {
		select {
		case <-ctx.Done():
		case <-time.After(10 * time.Second):
		}
		return nil, ctx.Err()
	})
	s := newTestFirestoreScaler(t, nil)

	t.Run("the scaler is created with the timeout of the package", func(t *testing.T) {
		if s.timeout != firestoreQueryTimeout {
			t.Errorf("timeout: got %v, want %v", s.timeout, firestoreQueryTimeout)
		}
	})

	t.Run("a slow backend is cut by the timeout of the scaler", func(t *testing.T) {
		s.timeout = 100 * time.Millisecond
		started := time.Now()
		_, _, err := s.GetMetricsAndActivity(context.Background(), "s0-test")
		if err == nil {
			t.Fatal("expected an error, got nil")
		}
		if code := status.Code(err); code != codes.DeadlineExceeded {
			t.Errorf("code: got %v, want DeadlineExceeded (error %v)", code, err)
		}
		if elapsed := time.Since(started); elapsed > 5*time.Second {
			t.Errorf("the call took %v", elapsed)
		}
	})

	t.Run("a cancelled context stops the call", func(t *testing.T) {
		s.timeout = time.Minute
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, _, err := s.GetMetricsAndActivity(ctx, "s0-test"); err == nil {
			t.Error("expected an error, got nil")
		}
	})
}

func TestFirestoreClose(t *testing.T) {
	startFakeFirestore(t)
	s := newTestFirestoreScaler(t, nil)

	if _, _, err := s.GetMetricsAndActivity(context.Background(), "s0-test"); err != nil {
		t.Fatalf("before Close: %v", err)
	}
	if err := s.Close(context.Background()); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if _, _, err := s.GetMetricsAndActivity(context.Background(), "s0-test"); err == nil {
		t.Error("expected an error from a closed scaler, got nil")
	}
	if err := (&gcpFirestoreScaler{}).Close(context.Background()); err != nil {
		t.Errorf("closing a scaler without a client: %v", err)
	}
}

// ── the metric spec ───────────────────────────────────────────────────────────

var firestoreMetricNameCases = []struct {
	desc         string
	metadata     map[string]string
	triggerIndex int
	wantName     string
}{
	{desc: "trigger index 0", metadata: firestoreRequiredFields, triggerIndex: 0, wantName: "s0-gcp-firestore-my-project-default-jobs"},
	{desc: "trigger index 3", metadata: firestoreRequiredFields, triggerIndex: 3, wantName: "s3-gcp-firestore-my-project-default-jobs"},
	{
		desc:         "named database and a subcollection",
		metadata:     firestoreWith(map[string]string{"databaseId": "orders-db", "collection": "users/alice/jobs"}),
		triggerIndex: 0,
		wantName:     "s0-gcp-firestore-my-project-orders-db-users-alice-jobs",
	},
}

func TestFirestoreGetMetricSpecForScaling(t *testing.T) {
	for _, tc := range firestoreMetricNameCases {
		t.Run(tc.desc, func(t *testing.T) {
			t.Setenv("FIRESTORE_EMULATOR_HOST", "")
			meta, err := parseGcpFirestoreMetadata(&scalersconfig.ScalerConfig{
				AuthParams:      firestoreCredentials,
				TriggerMetadata: tc.metadata,
				ResolvedEnv:     testFirestoreResolvedEnv,
				TriggerIndex:    tc.triggerIndex,
			})
			if err != nil {
				t.Fatalf("parseGcpFirestoreMetadata: %v", err)
			}

			s := &gcpFirestoreScaler{metricType: v2.AverageValueMetricType, metadata: meta, logger: logr.Discard()}

			specs := s.GetMetricSpecForScaling(context.Background())
			if len(specs) != 1 {
				t.Fatalf("expected 1 metric spec, got %d", len(specs))
			}
			if got := specs[0].External.Metric.Name; got != tc.wantName {
				t.Errorf("metric name: got %q, want %q", got, tc.wantName)
			}
		})
	}
}

func TestFirestoreGetMetricSpecForScaling_Target(t *testing.T) {
	t.Setenv("FIRESTORE_EMULATOR_HOST", "")
	meta, err := parseGcpFirestoreMetadata(&scalersconfig.ScalerConfig{
		AuthParams:      firestoreCredentials,
		TriggerMetadata: firestoreWith(map[string]string{"targetValue": "20"}),
		ResolvedEnv:     testFirestoreResolvedEnv,
	})
	if err != nil {
		t.Fatalf("parseGcpFirestoreMetadata: %v", err)
	}

	cases := []struct {
		metricType v2.MetricTargetType
		milli      func(v2.MetricTarget) int64
	}{
		{v2.AverageValueMetricType, func(target v2.MetricTarget) int64 { return target.AverageValue.MilliValue() }},
		{v2.ValueMetricType, func(target v2.MetricTarget) int64 { return target.Value.MilliValue() }},
	}
	for _, tc := range cases {
		s := &gcpFirestoreScaler{metricType: tc.metricType, metadata: meta, logger: logr.Discard()}
		spec := s.GetMetricSpecForScaling(context.Background())[0]
		if spec.Type != v2.ExternalMetricSourceType {
			t.Errorf("%s: spec type: got %v, want External", tc.metricType, spec.Type)
		}
		if spec.External.Target.Type != tc.metricType {
			t.Errorf("%s: target type: got %v", tc.metricType, spec.External.Target.Type)
		}
		if got := tc.milli(spec.External.Target); got != 20000 {
			t.Errorf("%s: target: got %d milli, want 20000", tc.metricType, got)
		}
	}
}
