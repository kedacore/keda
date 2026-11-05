//go:build integration

package scalers

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"cloud.google.com/go/firestore"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

const intTestFirestoreProject = "test-project"

// requireFirestoreEmulator skips the test if FIRESTORE_EMULATOR_HOST is not set. Start the emulator with
//
//	gcloud emulators firestore start --host-port=localhost:8080
//
// or in a container (the image holds the emulator and Java):
//
//	podman run --rm -p 127.0.0.1:8080:8080 gcr.io/google.com/cloudsdktool/google-cloud-cli:emulators \
//	  gcloud emulators firestore start --host-port=0.0.0.0:8080
//
// then run: FIRESTORE_EMULATOR_HOST=localhost:8080 go test -tags integration ./pkg/scalers/ -run FirestoreIntegration
func requireFirestoreEmulator(t *testing.T) {
	t.Helper()
	if os.Getenv("FIRESTORE_EMULATOR_HOST") == "" {
		t.Skip("FIRESTORE_EMULATOR_HOST not set, skipping integration test")
	}
}

// newFirestoreTestClient returns a client of the emulator (the SDK reads FIRESTORE_EMULATOR_HOST).
func newFirestoreTestClient(t *testing.T) *firestore.Client {
	t.Helper()
	client, err := firestore.NewClient(context.Background(), intTestFirestoreProject)
	if err != nil {
		t.Fatalf("firestore.NewClient: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return client
}

// uniqueFirestoreCollection names a collection that no other run or test uses.
func uniqueFirestoreCollection(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

// seedFirestore writes the documents and registers their deletion right away.
func seedFirestore(t *testing.T, client *firestore.Client, collection string, docs map[string]map[string]interface{}) {
	t.Helper()
	ctx := context.Background()
	for id, data := range docs {
		ref := client.Collection(collection).Doc(id)
		if _, err := ref.Set(ctx, data); err != nil {
			t.Fatalf("writing %s/%s: %v", collection, id, err)
		}
		t.Cleanup(func() { _, _ = ref.Delete(context.Background()) })
	}
}

func newFirestoreIntegrationScaler(t *testing.T, metadata map[string]string) Scaler {
	t.Helper()
	scaler, err := NewGcpFirestoreScaler(context.Background(), &scalersconfig.ScalerConfig{
		TriggerMetadata: withOverrides(map[string]string{"projectId": intTestFirestoreProject}, metadata),
		AuthParams:      map[string]string{},
		ResolvedEnv:     map[string]string{},
	})
	if err != nil {
		t.Fatalf("NewGcpFirestoreScaler: %v", err)
	}
	t.Cleanup(func() { _ = scaler.Close(context.Background()) })
	return scaler
}

func firestoreMetric(t *testing.T, scaler Scaler) (count int64, active bool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	metrics, active, err := scaler.GetMetricsAndActivity(ctx, "s0-test")
	if err != nil {
		t.Fatalf("GetMetricsAndActivity: %v", err)
	}
	if len(metrics) != 1 {
		t.Fatalf("expected 1 metric, got %d", len(metrics))
	}
	return metrics[0].Value.MilliValue() / 1000, active
}

// firestoreTestJobs are the documents of the filter tests.
var firestoreTestJobs = map[string]map[string]interface{}{
	"j1": {"status": "pending", "priority": 1, "score": 0.2, "done": false, "tags": []string{"urgent"}, "owner": map[string]interface{}{"name": "alice"}},
	"j2": {"status": "pending", "priority": 3, "score": 0.5, "done": false, "tags": []string{"urgent", "billing"}, "owner": map[string]interface{}{"name": "alice"}},
	"j3": {"status": "pending", "priority": 5, "score": 0.9, "done": true, "tags": []string{}, "owner": map[string]interface{}{"name": "bob"}},
	"j4": {"status": "done", "priority": 3, "score": 0.7, "done": true, "tags": []string{"billing"}},
	"j5": {"status": "failed", "priority": 1, "score": 0.1, "done": false, "tags": []string{}},
}

func TestFirestoreIntegration_Filters(t *testing.T) {
	requireFirestoreEmulator(t)
	client := newFirestoreTestClient(t)
	collection := uniqueFirestoreCollection("jobs")
	seedFirestore(t, client, collection, firestoreTestJobs)

	cases := []struct {
		desc      string
		filter    map[string]string
		wantCount int64
	}{
		{desc: "the whole collection", filter: nil, wantCount: 5},
		{desc: "string equals (default operator)", filter: map[string]string{"filterField": "status", "filterValue": "pending"}, wantCount: 3},
		{desc: "string not equals", filter: map[string]string{"filterField": "status", "filterOperator": "!=", "filterValue": "pending"}, wantCount: 2},
		{desc: "integer greater or equal", filter: map[string]string{"filterField": "priority", "filterOperator": ">=", "filterValue": "3", "filterValueType": "int"}, wantCount: 3},
		{desc: "integer less than", filter: map[string]string{"filterField": "priority", "filterOperator": "<", "filterValue": "3", "filterValueType": "int"}, wantCount: 2},
		{desc: "float greater than", filter: map[string]string{"filterField": "score", "filterOperator": ">", "filterValue": "0.5", "filterValueType": "float"}, wantCount: 2},
		{desc: "boolean equals", filter: map[string]string{"filterField": "done", "filterValue": "true", "filterValueType": "bool"}, wantCount: 2},
		{desc: "array contains", filter: map[string]string{"filterField": "tags", "filterOperator": "array-contains", "filterValue": "urgent"}, wantCount: 2},
		{desc: "a nested field written with dots", filter: map[string]string{"filterField": "owner.name", "filterValue": "alice"}, wantCount: 2},
		{desc: "a number is not equal to the same text", filter: map[string]string{"filterField": "priority", "filterValue": "3"}, wantCount: 0},
		{desc: "the same number read as an integer", filter: map[string]string{"filterField": "priority", "filterValue": "3", "filterValueType": "int"}, wantCount: 2},
		{desc: "a value that no document has", filter: map[string]string{"filterField": "status", "filterValue": "archived"}, wantCount: 0},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			scaler := newFirestoreIntegrationScaler(t, withOverrides(map[string]string{"collection": collection}, tc.filter))

			count, active := firestoreMetric(t, scaler)
			if count != tc.wantCount {
				t.Errorf("count: got %d, want %d", count, tc.wantCount)
			}
			if active != (tc.wantCount > 0) {
				t.Errorf("active: got %v, want %v", active, tc.wantCount > 0)
			}
		})
	}
}

func TestFirestoreIntegration_EmptyAndMissingCollection(t *testing.T) {
	requireFirestoreEmulator(t)

	scaler := newFirestoreIntegrationScaler(t, map[string]string{"collection": uniqueFirestoreCollection("never-written")})

	count, active := firestoreMetric(t, scaler)
	if count != 0 || active {
		t.Errorf("a collection without documents: got count %d and active %v, want 0 and false", count, active)
	}
}

func TestFirestoreIntegration_Subcollection(t *testing.T) {
	requireFirestoreEmulator(t)
	client := newFirestoreTestClient(t)
	parent := uniqueFirestoreCollection("tenants")
	seedFirestore(t, client, parent+"/acme/jobs", map[string]map[string]interface{}{
		"a": {"status": "pending"},
		"b": {"status": "pending"},
	})
	seedFirestore(t, client, parent+"/other/jobs", map[string]map[string]interface{}{
		"c": {"status": "pending"},
	})

	scaler := newFirestoreIntegrationScaler(t, map[string]string{"collection": parent + "/acme/jobs"})

	if count, _ := firestoreMetric(t, scaler); count != 2 {
		t.Errorf("subcollection of one document: got %d, want 2", count)
	}
}

// TestFirestoreIntegration_FollowsTheCollection changes the collection while the scaler is alive,
// as a workload that adds and finishes jobs does, and checks that the metric and the activity follow.
func TestFirestoreIntegration_FollowsTheCollection(t *testing.T) {
	requireFirestoreEmulator(t)
	client := newFirestoreTestClient(t)
	collection := uniqueFirestoreCollection("queue")
	ctx := context.Background()

	scaler := newFirestoreIntegrationScaler(t, map[string]string{
		"collection":      collection,
		"filterField":     "status",
		"filterValue":     "pending",
		"activationValue": "2",
	})

	steps := []struct {
		desc       string
		change     func()
		wantCount  int64
		wantActive bool
	}{
		{desc: "empty", change: func() {}, wantCount: 0, wantActive: false},
		{
			desc: "two pending jobs are not above the activation value",
			change: func() {
				seedFirestore(t, client, collection, map[string]map[string]interface{}{"a": {"status": "pending"}, "b": {"status": "pending"}})
			},
			wantCount: 2, wantActive: false,
		},
		{
			desc: "a third pending job activates the scaler",
			change: func() {
				seedFirestore(t, client, collection, map[string]map[string]interface{}{"c": {"status": "pending"}})
			},
			wantCount: 3, wantActive: true,
		},
		{
			desc: "a job that is not pending is not counted",
			change: func() {
				seedFirestore(t, client, collection, map[string]map[string]interface{}{"d": {"status": "done"}})
			},
			wantCount: 3, wantActive: true,
		},
		{
			desc: "finished jobs lower the count",
			change: func() {
				for _, id := range []string{"a", "b"} {
					if _, err := client.Collection(collection).Doc(id).Update(ctx, []firestore.Update{{Path: "status", Value: "done"}}); err != nil {
						t.Fatalf("updating %s: %v", id, err)
					}
				}
			},
			wantCount: 1, wantActive: false,
		},
	}
	for _, step := range steps {
		step.change()
		count, active := firestoreMetric(t, scaler)
		if count != step.wantCount || active != step.wantActive {
			t.Errorf("%s: got count %d and active %v, want %d and %v", step.desc, count, active, step.wantCount, step.wantActive)
		}
	}
}

func TestFirestoreIntegration_CancelledContext(t *testing.T) {
	requireFirestoreEmulator(t)

	scaler := newFirestoreIntegrationScaler(t, map[string]string{"collection": uniqueFirestoreCollection("jobs")})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := scaler.GetMetricsAndActivity(ctx, "s0-test"); err == nil {
		t.Error("expected an error from a cancelled context, got nil")
	}
}
