package scalers

import (
	"context"
	"fmt"
	"math"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/go-logr/logr"
	"google.golang.org/api/option"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/pkg/metricscollector"
	"github.com/kedacore/keda/v2/pkg/scalers/gcp"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	firestoreMetricNamePrefix = "gcp-firestore"
	firestoreCountAlias       = "count"
	// firestoreQueryTimeout bounds one count. Firestore itself gives up on an aggregation after
	// 60 seconds, and the polling interval of a ScaledObject is 30 seconds by default.
	firestoreQueryTimeout = 30 * time.Second
)

// firestoreFilterOperators are the comparison operators accepted by filterOperator. The default is
// the first one. The operators that take a list of values (in, not-in, array-contains-any) are not
// supported yet.
var firestoreFilterOperators = []string{"==", "!=", "<", "<=", ">", ">=", "array-contains"}

// firestoreFilterValueTypes are the values accepted by filterValueType. The default is the first one.
var firestoreFilterValueTypes = []string{"string", "int", "float", "bool"}

// firestoreReservedName matches the ids that Firestore reserves for itself (for example __name__).
var firestoreReservedName = regexp.MustCompile(`^__.*__$`)

type gcpFirestoreScaler struct {
	client       *firestore.Client
	query        firestore.Query
	timeout      time.Duration
	metricType   v2.MetricTargetType
	metadata     *gcpFirestoreMetadata
	scalerConfig scalersconfig.ScalerConfig
	logger       logr.Logger
}

type gcpFirestoreMetadata struct {
	ProjectID       string  `keda:"name=projectId,       order=triggerMetadata"`
	DatabaseID      string  `keda:"name=databaseId,      order=triggerMetadata, default=(default)"`
	Collection      string  `keda:"name=collection,      order=triggerMetadata"`
	FilterField     string  `keda:"name=filterField,     order=triggerMetadata, optional"`
	FilterOperator  string  `keda:"name=filterOperator,  order=triggerMetadata, optional"`
	FilterValue     string  `keda:"name=filterValue,     order=triggerMetadata, optional"`
	FilterValueType string  `keda:"name=filterValueType, order=triggerMetadata, default=string, enum=string;int;float;bool"`
	TargetValue     float64 `keda:"name=targetValue,     order=triggerMetadata, default=5"`
	ActivationValue float64 `keda:"name=activationValue, order=triggerMetadata, default=0"`

	gcp.AuthMetadata `keda:"optional"`

	gcpAuthorization *gcp.AuthorizationMetadata
	filterValue      interface{}
	metricName       string
	triggerIndex     int
}

// Validate checks the metadata that the typed config cannot check. The default of filterOperator
// is set here and not in the tag, because a tag cannot hold a value that contains "=".
func (m *gcpFirestoreMetadata) Validate() error {
	if m.TargetValue <= 0 {
		return fmt.Errorf("targetValue must be greater than 0, got %v", m.TargetValue)
	}

	if err := validateFirestoreCollectionPath(m.Collection); err != nil {
		return err
	}

	// The enum of the tag accepts a comma separated list such as "int,float".
	if !slices.Contains(firestoreFilterValueTypes, m.FilterValueType) {
		return fmt.Errorf("filterValueType must be one of %s, got %q", strings.Join(firestoreFilterValueTypes, ", "), m.FilterValueType)
	}

	if m.FilterField == "" {
		if m.FilterOperator != "" || m.FilterValue != "" {
			return fmt.Errorf("filterOperator and filterValue require filterField")
		}
		return nil
	}

	if err := validateFirestoreFieldPath(m.FilterField); err != nil {
		return err
	}
	if m.FilterOperator == "" {
		m.FilterOperator = firestoreFilterOperators[0]
	}
	if !slices.Contains(firestoreFilterOperators, m.FilterOperator) {
		return fmt.Errorf("filterOperator must be one of %s, got %q", strings.Join(firestoreFilterOperators, ", "), m.FilterOperator)
	}
	if m.FilterValue == "" {
		return fmt.Errorf("filterValue is required when filterField is set")
	}

	value, err := parseFirestoreFilterValue(m.FilterValueType, m.FilterValue)
	if err != nil {
		return err
	}
	// Firestore refuses to compare with NaN other than for equality.
	if number, ok := value.(float64); ok && math.IsNaN(number) && m.FilterOperator != "==" && m.FilterOperator != "!=" {
		return fmt.Errorf("filterValue NaN can be used only with the operators == and !=, got %q", m.FilterOperator)
	}
	m.filterValue = value

	return nil
}

// validateFirestoreCollectionPath accepts a collection id or the path of a subcollection
// (collection/document/collection), which has an odd number of segments. No segment may be empty
// or an id that Firestore reserves.
func validateFirestoreCollectionPath(path string) error {
	segments := strings.Split(path, "/")
	if len(segments)%2 == 0 {
		return fmt.Errorf("collection must be a collection id or the path of a subcollection (collection/document/collection), got %q", path)
	}
	for _, segment := range segments {
		if segment == "" || segment == "." || segment == ".." || firestoreReservedName.MatchString(segment) {
			return fmt.Errorf("collection %q has a segment that is empty or reserved by Firestore: %q", path, segment)
		}
	}
	return nil
}

// validateFirestoreFieldPath accepts a field name or dot-separated names of nested fields. A name
// may not be empty, may not be reserved by Firestore and may not contain any of ~*/[ and ].
func validateFirestoreFieldPath(path string) error {
	for _, name := range strings.Split(path, ".") {
		if name == "" || firestoreReservedName.MatchString(name) || strings.ContainsAny(name, "~*/[]") {
			return fmt.Errorf("filterField %q is not a valid field path: a name in it is empty, reserved by Firestore or contains one of ~*/[]", path)
		}
	}
	return nil
}

// parseFirestoreFilterValue converts the filterValue metadata, which is always a string, to the Go type
// that Firestore compares the field with.
func parseFirestoreFilterValue(valueType, raw string) (interface{}, error) {
	switch valueType {
	case "string":
		return raw, nil
	case "int":
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("filterValue %q is not an integer: %w", raw, err)
		}
		return value, nil
	case "float":
		value, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return nil, fmt.Errorf("filterValue %q is not a number: %w", raw, err)
		}
		return value, nil
	case "bool":
		value, err := strconv.ParseBool(raw)
		if err != nil {
			return nil, fmt.Errorf("filterValue %q is not a boolean: %w", raw, err)
		}
		return value, nil
	default:
		return nil, fmt.Errorf("filterValueType must be one of %s, got %q", strings.Join(firestoreFilterValueTypes, ", "), valueType)
	}
}

// NewGcpFirestoreScaler creates a new gcpFirestoreScaler.
func NewGcpFirestoreScaler(ctx context.Context, config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, "gcp_firestore_scaler")

	meta, err := parseGcpFirestoreMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing Firestore metadata: %w", err)
	}

	client, err := newGcpFirestoreClient(ctx, meta)
	if err != nil {
		return nil, fmt.Errorf("error creating Firestore client: %w", err)
	}

	return &gcpFirestoreScaler{
		client:       client,
		query:        newGcpFirestoreQuery(client, meta),
		timeout:      firestoreQueryTimeout,
		metricType:   metricType,
		metadata:     meta,
		scalerConfig: *config,
		logger:       logger,
	}, nil
}

func parseGcpFirestoreMetadata(config *scalersconfig.ScalerConfig) (*gcpFirestoreMetadata, error) {
	meta := &gcpFirestoreMetadata{triggerIndex: config.TriggerIndex}
	if err := config.TypedConfig(meta); err != nil {
		return nil, err
	}

	// The default database is called "(default)"; the parentheses are dropped so that its name does
	// not become "--default-" in the metric name.
	meta.metricName = GenerateMetricNameWithIndex(
		config.TriggerIndex,
		kedautil.NormalizeString(fmt.Sprintf("%s-%s-%s-%s",
			firestoreMetricNamePrefix,
			meta.ProjectID,
			strings.Trim(meta.DatabaseID, "()"),
			meta.Collection,
		)),
	)

	// When FIRESTORE_EMULATOR_HOST is set the SDK talks to the emulator without
	// credentials; they are not required in that case.
	if os.Getenv("FIRESTORE_EMULATOR_HOST") != "" {
		meta.gcpAuthorization = &gcp.AuthorizationMetadata{}
		return meta, nil
	}

	auth, err := gcp.GetGCPAuthorization(config)
	if err != nil {
		return nil, err
	}
	meta.gcpAuthorization = auth

	return meta, nil
}

func newGcpFirestoreClient(ctx context.Context, meta *gcpFirestoreMetadata) (*firestore.Client, error) {
	return firestore.NewClientWithDatabase(
		ctx,
		meta.ProjectID,
		meta.DatabaseID,
		gcpFirestoreClientOptions(meta)...,
	)
}

// gcpFirestoreClientOptions returns the options of the Firestore client: the credentials of the trigger and
// the gRPC dial options of the metrics collectors that instrument the outbound calls. The SDK ignores dial
// options when it talks to the emulator (FIRESTORE_EMULATOR_HOST), because it then uses a connection of its own.
func gcpFirestoreClientOptions(meta *gcpFirestoreMetadata) []option.ClientOption {
	var opts []option.ClientOption

	switch {
	case meta.gcpAuthorization.PodIdentityProviderEnabled:
		// Workload Identity / ADC: the SDK picks up credentials from the
		// metadata server automatically; no explicit option is needed.
	case meta.gcpAuthorization.GoogleApplicationCredentials != "":
		opts = append(opts, option.WithAuthCredentialsJSON(option.ServiceAccount, []byte(meta.gcpAuthorization.GoogleApplicationCredentials)))
	case meta.gcpAuthorization.GoogleApplicationCredentialsFile != "":
		opts = append(opts, option.WithAuthCredentialsFile(option.ServiceAccount, meta.gcpAuthorization.GoogleApplicationCredentialsFile))
	default:
		// No explicit credentials and no pod identity: the SDK uses Application
		// Default Credentials (GOOGLE_APPLICATION_CREDENTIALS, gcloud default, the
		// metadata server). This is reached when the emulator host is set, and also
		// when credentialsFromEnv or credentialsFromEnvFile names a variable that is
		// empty, which GetGCPAuthorization does not check (the same as for the other
		// GCP scalers).
	}

	for _, dialOption := range metricscollector.GRPCClientDialOptions() {
		opts = append(opts, option.WithGRPCDialOption(dialOption))
	}
	return opts
}

// newGcpFirestoreQuery builds the query whose documents are counted: the whole collection,
// or the documents that match the one filter of the trigger.
func newGcpFirestoreQuery(client *firestore.Client, meta *gcpFirestoreMetadata) firestore.Query {
	query := client.Collection(meta.Collection).Query
	if meta.FilterField != "" {
		query = query.WhereEntity(firestore.PropertyFilter{
			Path:     meta.FilterField,
			Operator: meta.FilterOperator,
			Value:    meta.filterValue,
		})
	}
	return query
}

func (s *gcpFirestoreScaler) Close(_ context.Context) error {
	if s.client != nil {
		return s.client.Close()
	}
	return nil
}

// count returns the number of documents that the query selects, with a Firestore count aggregation,
// so that no document is read. A collection without documents gives 0. A response that holds no
// integer count is an error and not 0, because 0 would scale the workload to zero.
func (s *gcpFirestoreScaler) count(ctx context.Context) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	result, err := s.query.NewAggregationQuery().WithCount(firestoreCountAlias).Get(ctx)
	if err != nil {
		return 0, fmt.Errorf("running the count aggregation: %w", err)
	}

	var fields map[string]interface{}
	if err := result.DataTo(&fields); err != nil {
		return 0, fmt.Errorf("reading the count aggregation: %w", err)
	}
	count, ok := fields[firestoreCountAlias].(int64)
	if !ok {
		return 0, fmt.Errorf("the count aggregation returned %v (%T) instead of an integer", fields[firestoreCountAlias], fields[firestoreCountAlias])
	}
	return count, nil
}

func (s *gcpFirestoreScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	ctx = metricscollector.BuildScalerRequestCtx(ctx, s.scalerConfig, metricName)
	count, err := s.count(ctx)
	if err != nil {
		s.logger.Error(err, "error counting Firestore documents")
		return nil, false, err
	}

	metric := GenerateMetricInMili(metricName, float64(count))

	return []external_metrics.ExternalMetricValue{metric}, float64(count) > s.metadata.ActivationValue, nil
}

func (s *gcpFirestoreScaler) GetMetricSpecForScaling(_ context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{Name: s.metadata.metricName},
		Target: GetMetricTargetMili(s.metricType, s.metadata.TargetValue),
	}
	return []v2.MetricSpec{{External: externalMetric, Type: externalMetricType}}
}
