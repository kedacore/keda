package scalers

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

var testNeo4jResolvedEnv = map[string]string{
	"Neo4j_CONN_STR": "neo4j://localhost:7687/",
	"Neo4j_USERNAME": "neo4j",
	"Neo4j_PASSWORD": "password",
}

type parseNeo4jMetadataTestData struct {
	metadata    map[string]string
	authParams  map[string]string
	resolvedEnv map[string]string
	raisesError bool
}

type neo4jMetricIdentifier struct {
	metadataTestData *parseNeo4jMetadataTestData
	scalerIndex      int
	name             string
}

var testNEO4JMetadata = []parseNeo4jMetadataTestData{
	// No metadata
	{
		metadata:    map[string]string{},
		authParams:  map[string]string{},
		resolvedEnv: testNeo4jResolvedEnv,
		raisesError: true,
	},
	// connectionStringFromEnv
	{
		metadata: map[string]string{"query": `MATCH (n:Person)<-[r:FOLLOWS]-() WHERE n.popularfor IS NOT NULL RETURN n,COUNT(r) order by COUNT(r) desc LIMIT 1`,
			"queryValue": "9", "connectionStringFromEnv": "Neo4j_CONN_STR", "username": "Neo4j_USERNAME", "password": "Neo4j_PASSWORD"},
		authParams:  map[string]string{},
		resolvedEnv: testNeo4jResolvedEnv,
		raisesError: false,
	},
	// with metric name
	{
		metadata: map[string]string{"query": `MATCH (n:Person)<-[r:FOLLOWS]-() WHERE n.popularfor IS NOT NULL RETURN n,COUNT(r) order by COUNT(r) desc LIMIT 1`,
			"queryValue": "9", "connectionStringFromEnv": "Neo4j_CONN_STR", "username": "Neo4j_USERNAME", "password": "Neo4j_PASSWORD"},
		authParams:  map[string]string{},
		resolvedEnv: testNeo4jResolvedEnv,
		raisesError: false,
	},
	// from trigger auth
	{
		metadata: map[string]string{"query": `MATCH (n:Person)<-[r:FOLLOWS]-() WHERE n.popularfor IS NOT NULL RETURN n,COUNT(r) order by COUNT(r) desc LIMIT 1`,
			"queryValue": "9"},
		authParams:  map[string]string{"host": "localhost", "port": "7687", "username": "neo4j", "password": "password"},
		resolvedEnv: testNeo4jResolvedEnv,
		raisesError: false,
	},
	// wrong activationQueryValue
	{
		metadata: map[string]string{"query": `MATCH (n:Person)<-[r:FOLLOWS]-() WHERE n.popularfor IS NOT NULL RETURN n,COUNT(r) order by COUNT(r) desc LIMIT 1`,
			"queryValue": "9", "activationQueryValue": "9", "connectionStringFromEnv": "Neo4j_CONN_STR",
			"username": "Neo4j_USERNAME", "password": "Neo4j_PASSWORD"},
		authParams:  map[string]string{},
		resolvedEnv: testNeo4jResolvedEnv,
		raisesError: true,
	},
}

var neo4jMetricIdentifiers = []neo4jMetricIdentifier{
	{metadataTestData: &testNEO4JMetadata[2], scalerIndex: 0, name: "s0-s0-neo4j"},
	{metadataTestData: &testNEO4JMetadata[2], scalerIndex: 1, name: "s1-s1-neo4j"},
}

func TestParseNeo4jMetadata(t *testing.T) {
	for _, testData := range testNEO4JMetadata {
		_, _, err := parseNeo4jMetadata(&ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: testData.authParams})
		if err != nil && !testData.raisesError {
			t.Error("Expected success but got error:", err)
		}
	}
}

func TestNeo4jGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range neo4jMetricIdentifiers {
		meta, _, err := parseNeo4jMetadata(&ScalerConfig{ResolvedEnv: testData.metadataTestData.resolvedEnv, AuthParams: testData.metadataTestData.authParams, TriggerMetadata: testData.metadataTestData.metadata, ScalerIndex: testData.scalerIndex})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}

		driverWithContext, err := neo4j.NewDriverWithContext("neo4j://host:7687", neo4j.BasicAuth("username", "password", ""))
		if err != nil {
			t.Fatal("couldn't create driver: ", err)
		}
		mockNeo4jScaler := neo4jScaler{
			metricType: "",
			metadata:   meta,
			driver:     driverWithContext,
			logger:     logr.Discard(),
		}

		metricSpec := mockNeo4jScaler.GetMetricSpecForScaling(context.Background())
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}
