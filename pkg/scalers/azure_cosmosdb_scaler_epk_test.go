package scalers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const cosmosDBTestEPKBoundary = "1FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"

func newCosmosDBEPKTestClient(server *httptest.Server) *cosmosDBClient {
	return &cosmosDBClient{
		httpClient:       server.Client(),
		dataEndpoint:     server.URL,
		dataKey:          "dGVzdGtleQ==",
		leaseEndpoint:    server.URL,
		leaseKey:         "bGVhc2VrZXk=",
		databaseID:       "testdb",
		containerID:      "data",
		leaseDatabaseID:  "testdb",
		leaseContainerID: "leases",
		processorName:    "testprocessor",
		logger:           logr.Discard(),
	}
}

func testCosmosDBEPKLeaseFixture(t *testing.T, fixture string, etags []string) {
	t.Helper()
	// Fixtures follow DocumentServiceLeaseCoreEpk (.NET) and ServiceItemLeaseV1 (Java).
	// Java tokens contain ChangeFeedStateV1 with a nested FeedRangeCompositeContinuationImpl.
	data, err := os.ReadFile("testdata/azure_cosmosdb/" + fixture)
	require.NoError(t, err)
	for _, subrange := range []bool{false, true} {
		t.Run(fmt.Sprintf("subrange=%t", subrange), func(t *testing.T) {
			rangeRequests, feedRequests := 0, 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/dbs/testdb/colls/leases/docs":
					_, _ = w.Write(data)
				case "/dbs/testdb/colls/data/pkranges":
					rangeRequests++
					assert.Equal(t, http.MethodGet, r.Method)
					assert.Equal(t, cosmosDBRestAPIVersion, r.Header.Get("x-ms-version"))
					auth, authErr := generateCosmosDBAuthToken(http.MethodGet, "pkranges", "dbs/testdb/colls/data", r.Header.Get("x-ms-date"), "dGVzdGtleQ==")
					assert.NoError(t, authErr)
					assert.Equal(t, auth, r.Header.Get("Authorization"))
					switch {
					case subrange:
						_, _ = w.Write([]byte(`{"PartitionKeyRanges":[{"id":"7","minInclusive":"","maxExclusive":"FF"}]}`))
					case rangeRequests == 1:
						assert.Empty(t, r.Header.Get("x-ms-continuation"))
						w.Header().Set("x-ms-continuation", "next-page")
						_, _ = fmt.Fprintf(w, `{"PartitionKeyRanges":[{"id":"6","minInclusive":"","maxExclusive":"%s"}]}`, cosmosDBTestEPKBoundary)
					default:
						assert.Equal(t, "next-page", r.Header.Get("x-ms-continuation"))
						_, _ = fmt.Fprintf(w, `{"PartitionKeyRanges":[{"id":"9","minInclusive":"%s","maxExclusive":"FF"}]}`, cosmosDBTestEPKBoundary)
					}
				case "/dbs/testdb/colls/data/docs":
					index := feedRequests
					feedRequests++
					if !assert.Less(t, index, len(etags)) {
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					assert.Equal(t, etags[index], r.Header.Get("If-None-Match"))
					assert.Equal(t, "Incremental feed", r.Header.Get("A-IM"))
					assert.Equal(t, "1", r.Header.Get("x-ms-max-item-count"))
					if subrange {
						assert.Equal(t, "7", r.Header.Get("x-ms-documentdb-partitionkeyrangeid"))
						assert.Equal(t, "EffectivePartitionKeyRange", r.Header.Get("x-ms-read-key-type"))
						assert.Contains(t, r.Header, http.CanonicalHeaderKey("x-ms-start-epk"))
						assert.Equal(t, []string{"", cosmosDBTestEPKBoundary}[index], r.Header.Get("x-ms-start-epk"))
						assert.Equal(t, []string{cosmosDBTestEPKBoundary, "FF"}[index], r.Header.Get("x-ms-end-epk"))
					} else {
						assert.Equal(t, []string{"6", "9"}[index], r.Header.Get("x-ms-documentdb-partitionkeyrangeid"))
						for _, header := range []string{"x-ms-read-key-type", "x-ms-start-epk", "x-ms-end-epk"} {
							assert.NotContains(t, r.Header, http.CanonicalHeaderKey(header))
						}
					}
					if index == 0 {
						w.Header().Set("x-ms-session-token", "6:0#900")
						_, _ = w.Write([]byte(`{"Documents":[{"id":"doc1","_lsn":751}]}`))
					} else {
						w.WriteHeader(http.StatusNotModified)
					}
				default:
					t.Errorf("unexpected request: %s", r.URL)
					w.WriteHeader(http.StatusBadRequest)
				}
			}))
			defer server.Close()
			lag, count, split, err := estimateCosmosDBLegacyLag(newCosmosDBEPKTestClient(server))
			require.NoError(t, err)
			assert.Equal(t, int64(150), lag)
			assert.Equal(t, int64(1), count)
			assert.False(t, split)
			assert.Equal(t, 2, feedRequests)
			if subrange {
				assert.Equal(t, 1, rangeRequests)
			} else {
				assert.Equal(t, 2, rangeRequests)
			}
		})
	}
}

func TestCosmosDBEPKRangeResolution(t *testing.T) {
	ranges := []cosmosDBPartitionKeyRange{
		{ID: "4", Range: cosmosDBEPKRange{Min: "", Max: "40"}},
		{ID: "5", Range: cosmosDBEPKRange{Min: "40", Max: "80"}},
		{ID: "6", Range: cosmosDBEPKRange{Min: "80", Max: "FF"}},
	}
	tests := []struct {
		name  string
		min   string
		max   string
		id    string
		split bool
	}{
		{name: "exact first", min: "", max: "40", id: "4"},
		{name: "exact adjacent", min: "40", max: "80", id: "5"},
		{name: "exact last", min: "80", max: "FF", id: "6"},
		{name: "interior subrange", min: "50", max: "60", id: "5"},
		{name: "lower subrange", min: "", max: "20", id: "4"},
		{name: "upper subrange", min: "90", max: "FF", id: "6"},
		{name: "full split", min: "", max: "FF", split: true},
		{name: "partial split", min: "30", max: "50", split: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match, split, err := resolveCosmosDBEPKRange(cosmosDBEPKRange{Min: tt.min, Max: tt.max}, ranges)
			require.NoError(t, err)
			assert.Equal(t, tt.split, split)
			if tt.split {
				assert.Nil(t, match)
			} else {
				require.NotNil(t, match)
				assert.Equal(t, tt.id, match.ID)
			}
		})
	}
	for _, epkRange := range []cosmosDBEPKRange{{Min: "", Max: "20"}, {Min: "", Max: "60"}, {Min: "60", Max: "FF"}} {
		_, _, err := resolveCosmosDBEPKRange(epkRange, ranges[1:2])
		assert.ErrorContains(t, err, "no physical partition contains")
	}
}

func TestCosmosDBEPKSplitRecovery(t *testing.T) {
	const parent = `{"id":"parent","version":1,"LeaseToken":"-80","FeedRange":{"Range":{"min":"","max":"80"}},"ContinuationToken":"\"100\""}`
	const children = `{"id":"child1","version":1,"LeaseToken":"-40","FeedRange":{"Range":{"min":"","max":"40"}},"ContinuationToken":"\"100\""},
		{"id":"child2","version":1,"LeaseToken":"40-80","FeedRange":{"Range":{"min":"40","max":"80"}},"ContinuationToken":"\"100\""}`
	tests := []struct {
		name         string
		refreshLease bool
		serverSplit  bool
	}{
		{name: "persistent overlap"},
		{name: "children replace stale parent", refreshLease: true},
		{name: "split after range lookup", refreshLease: true, serverSplit: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			leaseRequests, rangeRequests := 0, 0
			feedRequests := map[string]int{}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/dbs/testdb/colls/leases/docs":
					leaseRequests++
					leases := parent
					if tt.refreshLease && leaseRequests > 1 {
						leases = children
					}
					_, _ = fmt.Fprintf(w, `{"Documents":[%s,{"id":"legacy","LeaseToken":"9","ContinuationToken":"\"200\""}]}`, leases)
				case "/dbs/testdb/colls/data/pkranges":
					rangeRequests++
					if tt.serverSplit && rangeRequests == 1 {
						_, _ = w.Write([]byte(`{"PartitionKeyRanges":[{"id":"0","minInclusive":"","maxExclusive":"80"},{"id":"9","minInclusive":"80","maxExclusive":"FF"}]}`))
					} else {
						_, _ = w.Write([]byte(`{"PartitionKeyRanges":[{"id":"1","minInclusive":"","maxExclusive":"40"},{"id":"2","minInclusive":"40","maxExclusive":"80"},{"id":"9","minInclusive":"80","maxExclusive":"FF"}]}`))
					}
				case "/dbs/testdb/colls/data/docs":
					id := r.Header.Get("x-ms-documentdb-partitionkeyrangeid")
					feedRequests[id]++
					switch id {
					case "0":
						w.Header().Set(cosmosDBSubStatusHeader, cosmosDBPartitionKeyRangeGoneSubStatus)
						w.WriteHeader(http.StatusGone)
					case "1", "2":
						assert.Equal(t, `"100"`, r.Header.Get("If-None-Match"))
						w.WriteHeader(http.StatusNotModified)
					case "9":
						w.Header().Set("x-ms-session-token", "9:0#500")
						_, _ = w.Write([]byte(`{"Documents":[{"_lsn":201}]}`))
					default:
						t.Errorf("unexpected range ID %q", id)
						w.WriteHeader(http.StatusBadRequest)
					}
				default:
					t.Errorf("unexpected request: %s", r.URL)
					w.WriteHeader(http.StatusBadRequest)
				}
			}))
			defer server.Close()
			lag, count, split, err := estimateCosmosDBLegacyLag(newCosmosDBEPKTestClient(server))
			require.NoError(t, err)
			assert.Equal(t, int64(300), lag)
			assert.Equal(t, int64(1), count)
			assert.Equal(t, !tt.refreshLease, split)
			assert.Equal(t, 2, leaseRequests)
			assert.Equal(t, 2, rangeRequests)
			assert.Equal(t, 2, feedRequests["9"])
			assert.Zero(t, feedRequests["-80"])
			if tt.refreshLease {
				assert.Equal(t, 1, feedRequests["1"])
				assert.Equal(t, 1, feedRequests["2"])
			}
			if tt.serverSplit {
				assert.Equal(t, 1, feedRequests["0"])
			}
		})
	}
}

func TestCosmosDBEPKSplitActivatesProcessor(t *testing.T) {
	leaseRequests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/dbs/testdb/colls/leases/docs":
			leaseRequests++
			_, _ = w.Write([]byte(`{"Documents":[{"id":"parent","version":1,"LeaseToken":"-FF","feedRange":{"Range":{"min":"","max":"FF"}},"ContinuationToken":"\"100\""}]}`))
		case "/dbs/testdb/colls/data/pkranges":
			_, _ = w.Write([]byte(`{"PartitionKeyRanges":[{"id":"1","minInclusive":"","maxExclusive":"80"},{"id":"2","minInclusive":"80","maxExclusive":"FF"}]}`))
		default:
			t.Errorf("split parent must not read the feed: %s", r.URL)
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	defer server.Close()
	scaler := &azureCosmosDBScaler{
		cosmosClient: newCosmosDBEPKTestClient(server),
		metadata:     &azureCosmosDBMetadata{Threshold: 100, ActivationThreshold: 10},
		logger:       logr.Discard(),
	}
	metrics, active, err := scaler.GetMetricsAndActivity(context.Background(), "test-metric")
	require.NoError(t, err)
	require.Len(t, metrics, 1)
	assert.Equal(t, int64(11), metrics[0].Value.Value())
	assert.True(t, active)
	assert.Equal(t, 2, leaseRequests)
}

func TestCosmosDBEPKMalformedFeedRange(t *testing.T) {
	for _, field := range []string{"FeedRange", "feedRange"} {
		for _, feedRange := range []string{
			`null`, `{}`, `[]`, `"invalid"`, `{"min":"","max":"FF"}`, `{"PKRangeId":"0"}`,
			`{"Range":null}`, `{"Range":{"max":"FF"}}`, `{"Range":{"min":null,"max":"FF"}}`,
			`{"Range":{"min":"","max":null}}`, `{"Range":{"min":0,"max":"FF"}}`,
			`{"Range":{"min":"80","max":"40"}}`, `{"Range":{"min":"80","max":"80"}}`,
			`{"Range":{"min":"","max":""}}`, `{"Range":{"min":"","max":"FFFF"}}`,
			`{"Range":{"min":"GG","max":"FF"}}`, `{"Range":{"min":"","max":"ff"}}`,
			`{"Range":{"min":"","max":"FF","isMinInclusive":false}}`,
			`{"Range":{"min":"","max":"FF","isMaxInclusive":true}}`,
		} {
			t.Run(field+"/"+feedRange, func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, "/dbs/testdb/colls/leases/docs", r.URL.Path)
					_, _ = fmt.Fprintf(w, `{"Documents":[{"id":"bad","version":1,"LeaseToken":"-FF",%q:%s}]}`, field, feedRange)
				}))
				defer server.Close()
				_, _, _, err := estimateCosmosDBLegacyLag(newCosmosDBEPKTestClient(server))
				assert.ErrorContains(t, err, `invalid FeedRange for lease "bad"`)
			})
		}
	}
	for _, version := range []int{1, 2} {
		lease := leaseDocument{ID: "missing", Version: version, LeaseToken: "-FF"}
		require.Error(t, lease.parseFeedRange())
	}
}

func TestCosmosDBEPKJavaContinuation(t *testing.T) {
	const valid = `{"V":1,"Rid":"rid???","Mode":"INCREMENTAL","StartFrom":{"Type":"BEGINNING"},"Continuation":{"V":1,"Rid":"rid???","Continuation":[{"token":"\"500\"","range":{"min":"","max":"FF"}}],"Range":{"min":"","max":"FF"}}}`
	for _, encoding := range []*base64.Encoding{base64.URLEncoding, base64.StdEncoding, base64.RawURLEncoding, base64.RawStdEncoding} {
		encoded := encoding.EncodeToString([]byte(valid))
		assert.True(t, strings.Contains(encoded, "_") || strings.Contains(encoded, "/"))
		token, err := decodeCosmosDBContinuation(encoded, cosmosDBEPKRange{Min: "20", Max: "40"})
		require.NoError(t, err)
		assert.Equal(t, cosmosDBChangeFeedStart{etag: `"500"`, checkpointed: true}, token)
	}

	for _, token := range []string{"", `"750"`} {
		actual, err := decodeCosmosDBContinuation(token, cosmosDBEPKRange{Min: "", Max: "FF"})
		require.NoError(t, err)
		assert.Equal(t, cosmosDBChangeFeedStart{etag: token, checkpointed: token != ""}, actual)
	}
	tests := []string{
		`{`, `null`, `{}`, `{"V":"1"}`,
		strings.Replace(valid, `"V":1`, `"V":2`, 1),
		strings.ReplaceAll(valid, `"V":1`, `"V":2`),
		strings.Replace(valid, `"INCREMENTAL"`, `"FULL_FIDELITY"`, 1),
		strings.Replace(valid, `"token":"\"500\""`, `"token":""`, 1),
		strings.Replace(valid, `"token":"\"500\""`, `"token":42`, 1),
		strings.Replace(valid, `"max":"FF"`, `"max":"80"`, 1),
		strings.Replace(valid, `"min":""`, `"min":"80"`, 1),
		strings.Replace(valid, `"range":{"min":"","max":"FF"}`, `"range":{}`, 1),
		`{"V":1,"Mode":"INCREMENTAL","Continuation":{"V":1,"Continuation":[]}}`,
		`{"V":1,"Mode":"INCREMENTAL","Continuation":{"V":1,"Continuation":[{},{}]}}`,
		`{"V":1,"Mode":"INCREMENTAL","StartFrom":{"Type":"NOW"}}`,
	}
	for _, startFrom := range []string{
		`null`, `{}`, `{"Type":"UNSUPPORTED"}`, `{"Type":42}`,
		`{"Type":"POINT_IN_TIME"}`, `{"Type":"POINT_IN_TIME","PointInTimeMs":null}`,
		`{"Type":"POINT_IN_TIME","PointInTimeMs":"1767225600000"}`,
		`{"Type":"POINT_IN_TIME","PointInTimeMs":1.5}`,
		`{"Type":"POINT_IN_TIME","PointInTimeMs":9223372036854775807}`,
		`{"Type":"POINT_IN_TIME","PointInTimeMs":-62135596800001}`,
		`{"Type":"LEASE"}`, `{"Type":"LEASE","Etag":""}`, `{"Type":"LEASE","Etag":null}`,
		`{"Type":"LEASE","Etag":"\"100\""}`,
		`{"Type":"LEASE","Etag":"\"100\"","Range":{"min":"","max":"80"}}`,
	} {
		for _, token := range []string{`"\"500\""`, `null`} {
			state := strings.Replace(valid, `{"Type":"BEGINNING"}`, startFrom, 1)
			tests = append(tests, strings.Replace(state, `"token":"\"500\""`, `"token":`+token, 1))
		}
	}
	for _, state := range tests {
		t.Run(state, func(t *testing.T) {
			_, err := decodeCosmosDBContinuation(base64.URLEncoding.EncodeToString([]byte(state)), cosmosDBEPKRange{Min: "", Max: "FF"})
			require.Error(t, err)
		})
	}
	_, err := decodeCosmosDBContinuation("not-base64!", cosmosDBEPKRange{Min: "", Max: "FF"})
	assert.ErrorContains(t, err, "invalid Java change feed continuation encoding")
}

func TestCosmosDBJavaInitialLeaseState(t *testing.T) {
	tests := []struct {
		name              string
		startFrom         string
		token             string
		noContinuation    bool
		wantETag          string
		wantModifiedSince string
		checkpointed      bool
	}{
		{name: "migrated beginning", startFrom: `{"Type":"BEGINNING"}`, token: `null`},
		{name: "migrated now", startFrom: `{"Type":"NOW"}`, token: `null`, wantETag: "*"},
		{name: "migrated point in time", startFrom: `{"Type":"POINT_IN_TIME","PointInTimeMs":1767225600123}`, token: `null`, wantModifiedSince: "Thu, 01 Jan 2026 00:00:00 GMT"},
		{name: "beginning time sentinel", startFrom: `{"Type":"POINT_IN_TIME","PointInTimeMs":-62135596800000}`, token: `null`},
		{name: "migrated configured lease", startFrom: `{"Type":"LEASE","Etag":"\"100\"","Range":{"min":"","max":"FF"}}`, token: `null`, wantETag: `"100"`, checkpointed: true},
		{name: "legacy checkpoint start", startFrom: `{"Type":"LEGACY_CHECKPOINT"}`, token: `null`},
		{name: "initial beginning", startFrom: `{"Type":"BEGINNING"}`, noContinuation: true},
		{name: "initial now", startFrom: `{"Type":"NOW"}`, noContinuation: true, wantETag: "*"},
		{name: "checkpoint replaces now", startFrom: `{"Type":"NOW"}`, token: `"\"500\""`, wantETag: `"500"`, checkpointed: true},
		{name: "checkpoint replaces configured lease", startFrom: `{"Type":"LEASE","Etag":"\"100\"","Range":{"min":"","max":"FF"}}`, token: `"\"500\""`, wantETag: `"500"`, checkpointed: true},
		{name: "checkpoint retains time filter", startFrom: `{"Type":"POINT_IN_TIME","PointInTimeMs":1767225600000}`, token: `"\"500\""`, wantETag: `"500"`, wantModifiedSince: "Thu, 01 Jan 2026 00:00:00 GMT", checkpointed: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			docs := make([]map[string]interface{}, 0, 2)
			for _, bounds := range [][2]string{{"", "80"}, {"80", "FF"}} {
				leaseRange := fmt.Sprintf(`{"min":%q,"max":%q}`, bounds[0], bounds[1])
				stateRange := `"Range":` + leaseRange
				if !tt.noContinuation {
					stateRange = fmt.Sprintf(`"Continuation":{"V":1,"Rid":"q0oIALS6YvQ=","Continuation":[{"token":%s,"range":%s}],"Range":%s}`, tt.token, leaseRange, leaseRange)
				}
				state := fmt.Sprintf(`{"V":1,"Rid":"q0oIALS6YvQ=","Mode":"INCREMENTAL","StartFrom":%s,%s}`, tt.startFrom, stateRange)
				token := bounds[0] + "-" + bounds[1]
				docs = append(docs, map[string]interface{}{
					"id":      "testprocessormyaccount.documents.azure.com_q0oIAA==_q0oIALS6YvQ=.." + token,
					"version": 1, "LeaseToken": token, "Owner": "java-host",
					"feedRange":         json.RawMessage(`{"Range":` + leaseRange + `}`),
					"ContinuationToken": base64.URLEncoding.EncodeToString([]byte(state)),
				})
			}
			feedRequests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/dbs/testdb/colls/leases/docs":
					_ = json.NewEncoder(w).Encode(map[string]interface{}{"Documents": docs})
				case "/dbs/testdb/colls/data/pkranges":
					_, _ = w.Write([]byte(`{"PartitionKeyRanges":[{"id":"6","minInclusive":"","maxExclusive":"80"},{"id":"9","minInclusive":"80","maxExclusive":"FF"}]}`))
				case "/dbs/testdb/colls/data/docs":
					feedRequests++
					assert.Contains(t, []string{"6", "9"}, r.Header.Get("x-ms-documentdb-partitionkeyrangeid"))
					assert.Equal(t, tt.wantETag, r.Header.Get("If-None-Match"))
					assert.Equal(t, tt.wantModifiedSince, r.Header.Get("If-Modified-Since"))
					if tt.wantETag == "" {
						assert.NotContains(t, r.Header, "If-None-Match")
					}
					if tt.wantModifiedSince == "" {
						assert.NotContains(t, r.Header, "If-Modified-Since")
					}
					if tt.wantETag == "*" {
						w.WriteHeader(http.StatusNotModified)
						return
					}
					w.Header().Set("x-ms-session-token", "6:0#200")
					_, _ = w.Write([]byte(`{"Documents":[{"_lsn":1}]}`))
				default:
					t.Errorf("unexpected request: %s", r.URL)
					w.WriteHeader(http.StatusBadRequest)
				}
			}))
			defer server.Close()
			state, err := newCosmosDBEPKTestClient(server).estimateLag(context.Background())
			require.NoError(t, err)
			assert.Equal(t, int64(2), state.totalLeases)
			wantCap := int64(1)
			if tt.checkpointed {
				wantCap = 2
			}
			if tt.wantETag == "*" {
				wantCap = 0
				assert.Zero(t, state.activeLeases)
				assert.Zero(t, state.totalLag)
			} else {
				assert.Equal(t, int64(2), state.activeLeases)
				assert.Equal(t, int64(400), state.totalLag)
			}
			assert.Equal(t, wantCap, state.legacyActivePartitions)

			for _, capacity := range []string{"", "1"} {
				config := cosmosDBCapacityConfig(server.URL)
				if capacity != "" {
					config.TriggerMetadata["maxActiveLeasesPerReplica"] = capacity
				}
				scaler, err := NewAzureCosmosDBScaler(config)
				require.NoError(t, err)
				spec := scaler.GetMetricSpecForScaling(context.Background())[0]
				metrics, active, err := scaler.GetMetricsAndActivity(context.Background(), spec.External.Metric.Name)
				require.NoError(t, err)
				assert.Equal(t, tt.wantETag != "*", active)
				require.Len(t, metrics, 1)
				wantMetric := wantCap * 100
				if capacity != "" {
					wantMetric = state.activeLeases
				}
				assert.Equal(t, wantMetric, metrics[0].Value.Value())
				require.NoError(t, scaler.Close(context.Background()))
			}
			assert.Equal(t, 6, feedRequests)
		})
	}
}

func TestCosmosDBLegacyContinuationIsUnchanged(t *testing.T) {
	for _, version := range []string{"", `"version":0,`} {
		t.Run(version, func(t *testing.T) {
			feedRequests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/dbs/testdb/colls/leases/docs":
					_, _ = fmt.Fprintf(w, `{"Documents":[{"id":"legacy",%s"LeaseToken":"42","ContinuationToken":"opaque-server-token","FeedRange":{"Range":{"min":"","max":"FF"}}}]}`, version)
				case "/dbs/testdb/colls/data/docs":
					feedRequests++
					assert.Equal(t, "42", r.Header.Get("x-ms-documentdb-partitionkeyrangeid"))
					assert.Equal(t, "opaque-server-token", r.Header.Get("If-None-Match"))
					for _, header := range []string{"x-ms-read-key-type", "x-ms-start-epk", "x-ms-end-epk"} {
						assert.NotContains(t, r.Header, http.CanonicalHeaderKey(header))
					}
					w.WriteHeader(http.StatusNotModified)
				default:
					t.Errorf("legacy lease must not query physical ranges: %s", r.URL)
					w.WriteHeader(http.StatusBadRequest)
				}
			}))
			defer server.Close()
			lag, count, split, err := estimateCosmosDBLegacyLag(newCosmosDBEPKTestClient(server))
			require.NoError(t, err)
			assert.Zero(t, lag)
			assert.Zero(t, count)
			assert.False(t, split)
			assert.Equal(t, 1, feedRequests)
		})
	}
}

func TestCosmosDBEPKRangeReadErrors(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
	}{
		{name: "authorization", status: http.StatusForbidden},
		{name: "throttling", status: http.StatusTooManyRequests},
		{name: "invalid JSON", status: http.StatusOK, body: `{`},
		{name: "empty ranges", status: http.StatusOK, body: `{"PartitionKeyRanges":[]}`},
		{name: "missing min", status: http.StatusOK, body: `{"PartitionKeyRanges":[{"id":"0","maxExclusive":"FF"}]}`},
		{name: "missing ID", status: http.StatusOK, body: `{"PartitionKeyRanges":[{"minInclusive":"","maxExclusive":"FF"}]}`},
		{name: "reversed bounds", status: http.StatusOK, body: `{"PartitionKeyRanges":[{"id":"0","minInclusive":"FF","maxExclusive":""}]}`},
		{name: "missing overlap", status: http.StatusOK, body: `{"PartitionKeyRanges":[{"id":"0","minInclusive":"80","maxExclusive":"FF"}]}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/dbs/testdb/colls/leases/docs":
					_, _ = w.Write([]byte(`{"Documents":[{"id":"epk","version":1,"LeaseToken":"-80","FeedRange":{"Range":{"min":"","max":"80"}}}]}`))
				case "/dbs/testdb/colls/data/pkranges":
					w.WriteHeader(tt.status)
					_, _ = w.Write([]byte(tt.body))
				default:
					t.Errorf("unexpected feed request: %s", r.URL)
					w.WriteHeader(http.StatusBadRequest)
				}
			}))
			defer server.Close()
			lag, count, split, err := estimateCosmosDBLegacyLag(newCosmosDBEPKTestClient(server))
			require.Error(t, err)
			assert.Zero(t, lag)
			assert.Zero(t, count)
			assert.False(t, split)
		})
	}
}

func TestCosmosDBEPKAndLegacyNeverCheckpointed(t *testing.T) {
	for _, version := range []int{0, 1} {
		t.Run(fmt.Sprintf("version=%d", version), func(t *testing.T) {
			rangeRequests, feedRequests := 0, 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/dbs/testdb/colls/leases/docs":
					leases := []map[string]interface{}{
						{"id": "first", "version": version, "LeaseToken": "6", "ContinuationToken": nil, "FeedRange": "ignored in v0"},
						{"id": "second", "version": version, "LeaseToken": "9", "ContinuationToken": "", "FeedRange": "ignored in v0"},
					}
					if version == 1 {
						for i, bounds := range [][2]string{{"", "80"}, {"80", "FF"}} {
							leases[i]["LeaseToken"] = bounds[0] + "-" + bounds[1]
							leases[i]["FeedRange"] = map[string]interface{}{"Range": map[string]string{"min": bounds[0], "max": bounds[1]}}
						}
					}
					_ = json.NewEncoder(w).Encode(map[string]interface{}{"Documents": leases})
				case "/dbs/testdb/colls/data/pkranges":
					rangeRequests++
					_, _ = w.Write([]byte(`{"PartitionKeyRanges":[{"id":"6","minInclusive":"","maxExclusive":"80"},{"id":"9","minInclusive":"80","maxExclusive":"FF"}]}`))
				case "/dbs/testdb/colls/data/docs":
					assert.Equal(t, []string{"6", "9"}[feedRequests], r.Header.Get("x-ms-documentdb-partitionkeyrangeid"))
					feedRequests++
					assert.NotContains(t, r.Header, "If-None-Match")
					assert.Empty(t, r.Header.Get("x-ms-read-key-type"))
					w.Header().Set("x-ms-session-token", "6:0#500")
					_, _ = w.Write([]byte(`{"Documents":[{"_lsn":1}]}`))
				default:
					t.Errorf("unexpected request: %s", r.URL)
					w.WriteHeader(http.StatusBadRequest)
				}
			}))
			defer server.Close()
			lag, count, split, err := estimateCosmosDBLegacyLag(newCosmosDBEPKTestClient(server))
			require.NoError(t, err)
			assert.Equal(t, int64(1000), lag)
			assert.Equal(t, int64(1), count)
			assert.False(t, split)
			assert.Equal(t, version, rangeRequests)
			assert.Equal(t, 2, feedRequests)
		})
	}
}
