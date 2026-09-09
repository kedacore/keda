package fallback

import (
	"context"
	"errors"
	"sync"
	"testing"

	"go.uber.org/mock/gomock"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kedacore/keda/v2/pkg/mock/mock_client"
	"github.com/kedacore/keda/v2/pkg/mock/mock_scale"
)

// primeReadyReplicas wires the Scale read and the Pod list that
// fetchReadyReplicasCount performs, and asserts each happens exactly `times`
// times. The Pod list is left empty: this exercises how often the API is hit,
// not what the count comes out to.
func primeReadyReplicas(t *testing.T, times int) (ScaledObjectHandler, *gomock.Controller) {
	t.Helper()
	ctrl := gomock.NewController(t)

	client := mock_client.NewMockClient(ctrl)
	scaleClient := mock_scale.NewMockScalesGetter(ctrl)
	scaleInterface := mock_scale.NewMockScaleInterface(ctrl)

	scaleClient.EXPECT().Scales(gomock.Eq("default")).Return(scaleInterface).Times(times)
	// The resource argument arrives as a schema.GroupResource, so it is matched
	// loosely here - this test is about how many times the call happens.
	scaleInterface.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.Eq("myapp"), gomock.Any()).
		Return(&autoscalingv1.Scale{
			ObjectMeta: metav1.ObjectMeta{Name: "myapp", Namespace: "default"},
		}, nil).
		Times(times)
	client.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(times)

	return ScaledObjectHandler{
		Ctx:          context.Background(),
		KubeClient:   client,
		ScaleClient:  scaleClient,
		UpdateLock:   &sync.RWMutex{},
		ScaledObject: buildScaledObject(nil, nil),
	}, ctrl
}

// The point of the cache: a ScaledObject with several Value-target triggers used
// to repeat a Scale read plus a full Pod list once per metric, for a number that
// is identical across all of them because they share one scale target.
func TestReadyReplicasCacheFetchesOnce(t *testing.T) {
	soh, ctrl := primeReadyReplicas(t, 1)
	defer ctrl.Finish()
	soh.ReadyReplicas = &ReadyReplicasCache{}

	for i := 0; i < 5; i++ {
		if _, err := getReadyReplicasCount(soh); err != nil {
			t.Fatalf("call %d: unexpected error: %v", i, err)
		}
	}
}

// Without a cache the behaviour is unchanged, so call sites that do not carry
// one - the other fallback tests among them - still read through every time.
func TestReadyReplicasNoCacheReadsThrough(t *testing.T) {
	soh, ctrl := primeReadyReplicas(t, 5)
	defer ctrl.Finish()

	for i := 0; i < 5; i++ {
		if _, err := getReadyReplicasCount(soh); err != nil {
			t.Fatalf("call %d: unexpected error: %v", i, err)
		}
	}
}

// Both handlers that share a cache fan out over goroutines, so concurrent
// callers must collapse onto one fetch rather than racing to issue their own.
func TestReadyReplicasCacheIsConcurrencySafe(t *testing.T) {
	soh, ctrl := primeReadyReplicas(t, 1)
	defer ctrl.Finish()
	soh.ReadyReplicas = &ReadyReplicasCache{}

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := getReadyReplicasCount(soh); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	wg.Wait()
}

// A failed lookup is memoised too. Retrying inside a single reconcile would hit
// the API once per metric precisely when the API is already unhappy, and every
// caller in that pass would fail anyway.
func TestReadyReplicasCacheMemoisesFailure(t *testing.T) {
	cache := &ReadyReplicasCache{}
	wantErr := errors.New("scale target is gone")

	var calls int
	fetch := func() (int32, error) {
		calls++
		return -1, wantErr
	}

	for i := 0; i < 3; i++ {
		count, err := cache.get(fetch)
		if !errors.Is(err, wantErr) {
			t.Fatalf("call %d: got error %v, want %v", i, err, wantErr)
		}
		if count != -1 {
			t.Fatalf("call %d: got count %d, want -1", i, count)
		}
	}
	if calls != 1 {
		t.Errorf("fetched %d times, want 1", calls)
	}
}

// A nil cache is a valid zero state - ScaledObjectHandler carries the field as a
// pointer and not every call site sets it.
func TestReadyReplicasNilCacheReadsThrough(t *testing.T) {
	var cache *ReadyReplicasCache

	var calls int
	for i := 0; i < 3; i++ {
		if _, err := cache.get(func() (int32, error) { calls++; return 7, nil }); err != nil {
			t.Fatalf("call %d: unexpected error: %v", i, err)
		}
	}
	if calls != 3 {
		t.Errorf("fetched %d times, want 3 - a nil cache must not memoise", calls)
	}
}
