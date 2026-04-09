package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/proxy"
	"go.temporal.io/sdk/converter"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"
)

const defaultPayloadSizeThreshold = 256 * 1024

type storageParameters struct {
	driverMap            map[string]converter.StorageDriver
	driverSelector       converter.StorageDriverSelector
	payloadSizeThreshold int
}

func ExternalStorageToParams(options converter.ExternalStorage) (storageParameters, error) {
	if options.PayloadSizeThreshold < 0 {
		return storageParameters{}, fmt.Errorf("PayloadSizeThreshold must not be negative")
	}

	driverMap := make(map[string]converter.StorageDriver, len(options.Drivers))
	for _, d := range options.Drivers {
		if _, exists := driverMap[d.Name()]; exists {
			return storageParameters{}, fmt.Errorf("duplicate storage driver name: %q", d.Name())
		}
		driverMap[d.Name()] = d
	}

	if len(options.Drivers) > 1 && options.DriverSelector == nil {
		return storageParameters{}, fmt.Errorf("DriverSelector must be set when more than one driver is provided")
	}

	selector := options.DriverSelector
	if selector == nil && len(options.Drivers) > 0 {
		selector = singleDriverSelector{driver: options.Drivers[0]}
	}

	sizeThreshold := options.PayloadSizeThreshold
	if sizeThreshold == 0 {
		sizeThreshold = defaultPayloadSizeThreshold
	}

	return storageParameters{
		driverMap:            driverMap,
		driverSelector:       selector,
		payloadSizeThreshold: sizeThreshold,
	}, nil
}

// singleDriverSelector is a StorageDriverSelector that always returns the same driver.
type singleDriverSelector struct {
	driver converter.StorageDriver
}

func (s singleDriverSelector) SelectDriver(_ converter.StorageDriverStoreContext, _ *commonpb.Payload) (converter.StorageDriver, error) {
	return s.driver, nil
}

// driversEqual compares two StorageDriver interface values. It uses == when
// the dynamic type is comparable (pointer types, simple value types) and
// falls back to name equality for non-comparable value types (e.g. structs
// with map fields).
func driversEqual(a, b converter.StorageDriver) (equal bool) {
	defer func() {
		if recover() != nil {
			equal = a.Name() == b.Name()
		}
	}()
	return a == b
}

type storageOperationCallback interface {
	PayloadBatchCompleted(count int, size int64, duration time.Duration)
	UnconfiguredStorageReference()
}

const storageOperationCallbackContextKey contextKey = "storageOperationCallback"
const storageTargetContextKey contextKey = "storageTarget"

// metadataEncodingStorageRef is the metadata encoding value used to identify
// payloads that are storage references rather than actual data.
const metadataEncodingStorageRef = "json/external-storage-reference"

type storageReference struct {
	DriverName  string                       `json:"driver_name"`
	DriverClaim converter.StorageDriverClaim `json:"driver_claim"`
}

func storageReferenceToPayload(ref storageReference, storedSizeBytes int64) (*commonpb.Payload, error) {
	data, err := json.Marshal(ref)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal storage reference: %w", err)
	}
	return &commonpb.Payload{
		Metadata: map[string][]byte{
			converter.MetadataEncoding: []byte(metadataEncodingStorageRef),
		},
		Data: data,
		ExternalPayloads: []*commonpb.Payload_ExternalPayloadDetails{
			{SizeBytes: storedSizeBytes},
		},
	}, nil
}

// payloadToStorageReference decodes a storage reference from a payload.
func payloadToStorageReference(p *commonpb.Payload) (storageReference, error) {
	if string(p.GetMetadata()[converter.MetadataEncoding]) != metadataEncodingStorageRef {
		return storageReference{}, fmt.Errorf("payload is not a storage reference: unexpected encoding %q", string(p.GetMetadata()[converter.MetadataEncoding]))
	}
	var ref storageReference
	if err := json.Unmarshal(p.Data, &ref); err != nil {
		return storageReference{}, fmt.Errorf("failed to unmarshal storage reference: %w", err)
	}
	return ref, nil
}

type externalRetrievalVisitor struct {
	params storageParameters
}

func (v *externalRetrievalVisitor) Visit(ctx *proxy.VisitPayloadsContext, payloads []*commonpb.Payload) ([]*commonpb.Payload, error) {
	startTime := time.Now()

	// Identify which payloads are storage references and group them by driver.
	type driverBatch struct {
		driver  converter.StorageDriver
		indices []int
		claims  []converter.StorageDriverClaim
	}
	var driverOrder []string
	driverBatches := map[string]*driverBatch{}

	result := make([]*commonpb.Payload, len(payloads))

	for i, p := range payloads {
		if string(p.GetMetadata()[converter.MetadataEncoding]) != metadataEncodingStorageRef {
			result[i] = p
			continue
		}

		// No storage drivers configured at all. Notify the caller and leave the
		// payload unresolved so downstream code can surface a clear error.
		if len(v.params.driverMap) == 0 {
			if cb, ok := ctx.Value(storageOperationCallbackContextKey).(storageOperationCallback); ok {
				cb.UnconfiguredStorageReference()
			}
			result[i] = p
			continue
		}

		ref, err := payloadToStorageReference(p)
		if err != nil {
			return nil, err
		}

		driver, ok := v.params.driverMap[ref.DriverName]
		if !ok {
			return nil, fmt.Errorf("no storage driver registered with name %q", ref.DriverName)
		}

		batch, exists := driverBatches[ref.DriverName]
		if !exists {
			batch = &driverBatch{driver: driver}
			driverBatches[ref.DriverName] = batch
			driverOrder = append(driverOrder, ref.DriverName)
		}
		batch.indices = append(batch.indices, i)
		batch.claims = append(batch.claims, ref.DriverClaim)
	}

	// Fan out to each driver concurrently. The errgroup context is used as the
	// StorageDriverRetrieveContext so a failing driver cancels in-flight siblings.
	// Intentionally creating an empty context so the retrieval path cannot use ambient
	// information for determing how to retrieve payloads. Drivers should only use information
	// from the StorageDriverClaim to retrieve payloads.
	eg, egCtx := errgroup.WithContext(context.Background())
	driverCtx := converter.StorageDriverRetrieveContext{Context: egCtx}
	sizes := make([]int64, len(driverOrder))

	externalCount := 0
	for i, name := range driverOrder {
		batch := driverBatches[name]
		externalCount += len(batch.claims)
		eg.Go(func() error {
			retrieved, err := callDriverRetrieve(batch.driver, driverCtx, batch.claims)
			if err != nil {
				return fmt.Errorf("storage driver %q retrieve failed: %w", name, err)
			}
			if len(retrieved) != len(batch.claims) {
				return fmt.Errorf("storage driver %q returned %d payloads for %d claims", name, len(retrieved), len(batch.claims))
			}
			var batchSize int64
			for j, p := range retrieved {
				batchSize += int64(len(p.GetData()))
				result[batch.indices[j]] = p
			}
			sizes[i] = batchSize
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return nil, err
	}

	externalTotalSize := int64(0)
	for _, s := range sizes {
		externalTotalSize += s
	}

	if callbackValue := ctx.Value(storageOperationCallbackContextKey); callbackValue != nil {
		if callback, isCallback := callbackValue.(storageOperationCallback); isCallback {
			callback.PayloadBatchCompleted(externalCount, externalTotalSize, time.Since(startTime))
		}
	}
	return result, nil
}

func NewExternalRetrievalVisitor(params storageParameters) PayloadVisitor {
	return &externalRetrievalVisitor{params: params}
}

type externalStorageVisitor struct {
	params storageParameters
}

func (v *externalStorageVisitor) Visit(ctx *proxy.VisitPayloadsContext, payloads []*commonpb.Payload) ([]*commonpb.Payload, error) {
	startTime := time.Now()

	if v.params.driverSelector == nil {
		return payloads, nil
	}

	// Determine which driver (if any) should store each payload.
	type driverBatch struct {
		driver   converter.StorageDriver
		indices  []int
		payloads []*commonpb.Payload
	}
	var driverOrder []string
	driverBatches := map[string]*driverBatch{}

	result := make([]*commonpb.Payload, len(payloads))
	var target converter.StorageDriverTargetInfo
	if t, ok := ctx.Context.Value(storageTargetContextKey).(converter.StorageDriverTargetInfo); ok {
		target = t
	}
	driverCtx := converter.StorageDriverStoreContext{Context: ctx.Context, Target: target}

	for i, p := range payloads {
		if proto.Size(p) < v.params.payloadSizeThreshold {
			result[i] = p
			continue
		}

		selected, err := callDriverSelector(v.params.driverSelector, driverCtx, p)
		if err != nil {
			return nil, fmt.Errorf("storage driver selector failed: %w", err)
		}
		var driver converter.StorageDriver
		if selected != nil {
			registered, ok := v.params.driverMap[selected.Name()]
			if !ok || !driversEqual(registered, selected) {
				return nil, fmt.Errorf("storage driver selector returned unregistered driver %q", selected.Name())
			}
			driver = selected
		}

		if driver == nil {
			result[i] = p
			continue
		}

		name := driver.Name()
		batch, exists := driverBatches[name]
		if !exists {
			batch = &driverBatch{driver: driver}
			driverBatches[name] = batch
			driverOrder = append(driverOrder, name)
		}
		batch.indices = append(batch.indices, i)
		batch.payloads = append(batch.payloads, p)
	}

	// Fan out to each driver concurrently. The errgroup context is used as the
	// StorageDriverStoreContext so a failing driver cancels in-flight siblings.
	eg, egCtx := errgroup.WithContext(ctx.Context)
	storeDrCtx := converter.StorageDriverStoreContext{Context: egCtx, Target: target}
	sizes := make([]int64, len(driverOrder))

	externalCount := 0
	for i, name := range driverOrder {
		batch := driverBatches[name]
		externalCount += len(batch.payloads)
		eg.Go(func() error {
			claims, err := callDriverStore(batch.driver, storeDrCtx, batch.payloads)
			if err != nil {
				return fmt.Errorf("storage driver %q store failed: %w", name, err)
			}
			if len(claims) != len(batch.payloads) {
				return fmt.Errorf("storage driver %q returned %d claims for %d payloads", name, len(claims), len(batch.payloads))
			}
			var batchSize int64
			for j, claim := range claims {
				ref := storageReference{
					DriverName:  name,
					DriverClaim: claim,
				}
				storedSize := int64(batch.payloads[j].Size())
				batchSize += storedSize
				refPayload, err := storageReferenceToPayload(ref, storedSize)
				if err != nil {
					return err
				}
				result[batch.indices[j]] = refPayload
			}
			sizes[i] = batchSize
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return nil, err
	}

	externalTotalSize := int64(0)
	for _, s := range sizes {
		externalTotalSize += s
	}

	if callbackValue := ctx.Value(storageOperationCallbackContextKey); callbackValue != nil {
		if callback, isCallback := callbackValue.(storageOperationCallback); isCallback {
			callback.PayloadBatchCompleted(externalCount, externalTotalSize, time.Since(startTime))
		}
	}
	return result, nil
}

func NewExternalStorageVisitor(params storageParameters) PayloadVisitor {
	return &externalStorageVisitor{params: params}
}

func callDriverSelector(s converter.StorageDriverSelector, ctx converter.StorageDriverStoreContext, p *commonpb.Payload) (driver converter.StorageDriver, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panicked: %v", r)
		}
	}()
	return s.SelectDriver(ctx, p)
}

func callDriverStore(d converter.StorageDriver, ctx converter.StorageDriverStoreContext, payloads []*commonpb.Payload) (claims []converter.StorageDriverClaim, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panicked: %v", r)
		}
	}()
	return d.Store(ctx, payloads)
}

func callDriverRetrieve(d converter.StorageDriver, ctx converter.StorageDriverRetrieveContext, claims []converter.StorageDriverClaim) (payloads []*commonpb.Payload, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panicked: %v", r)
		}
	}()
	return d.Retrieve(ctx, claims)
}
