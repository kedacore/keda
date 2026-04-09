package converter

import (
	"context"

	commonpb "go.temporal.io/api/common/v1"
)

// StorageDriverTargetInfo identifies the workflow or activity on whose behalf
// a payload is being stored. Use a type switch on [StorageDriverWorkflowInfo]
// and [StorageDriverActivityInfo] to access the concrete values.
//
// NOTE: Experimental
type StorageDriverTargetInfo interface {
	isStorageDriverTargetInfo()
}

// StorageDriverWorkflowInfo carries workflow identity for a storage operation.
//
// NOTE: Experimental
type StorageDriverWorkflowInfo struct {
	// Namespace is the Temporal namespace of the workflow execution.
	Namespace string
	// WorkflowType is the type name of the workflow.
	WorkflowType string
	// WorkflowID is the ID of the workflow execution.
	WorkflowID string
	// RunID is the run ID of the workflow execution.
	RunID string
}

func (StorageDriverWorkflowInfo) isStorageDriverTargetInfo() {}

var _ StorageDriverTargetInfo = StorageDriverWorkflowInfo{}

// StorageDriverActivityInfo carries activity identity for a storage operation.
// This is only used for standalone (non-workflow-bound) activities; activities
// started by a workflow use [StorageDriverWorkflowInfo] as the target.
//
// NOTE: Experimental
type StorageDriverActivityInfo struct {
	// Namespace is the Temporal namespace of the activity execution.
	Namespace string
	// ActivityType is the type name of the activity.
	ActivityType string
	// ActivityID is the ID of the activity execution.
	ActivityID string
	// RunID is the run ID of the activity execution.
	RunID string
}

func (StorageDriverActivityInfo) isStorageDriverTargetInfo() {}

var _ StorageDriverTargetInfo = StorageDriverActivityInfo{}

// StorageDriverStoreContext carries context passed to StorageDriver.Store and
// StorageDriverSelector.SelectDriver operations.
//
// NOTE: Experimental
type StorageDriverStoreContext struct {
	// Context is the context of the operation that triggered the driver call.
	// Drivers should use it to respect cancellation and to propagate deadlines
	// and trace information to downstream calls (e.g. cloud storage SDKs).
	Context context.Context
	// Target identifies the workflow or activity on whose behalf payloads are
	// being stored. Use a type switch on [StorageDriverWorkflowInfo] and
	// [StorageDriverActivityInfo] to access the concrete values.
	Target StorageDriverTargetInfo
}

// StorageDriverRetrieveContext carries context passed to StorageDriver.Retrieve
// operations.
//
// NOTE: Experimental
type StorageDriverRetrieveContext struct {
	// Context is the context of the operation that triggered the driver call.
	// Drivers should use it to respect cancellation and to propagate deadlines
	// and trace information to downstream calls (e.g. cloud storage SDKs).
	Context context.Context
}

// StorageDriverClaim is an opaque token returned by StorageDriver.Store that
// identifies where a payload was stored. The SDK serializes it alongside the
// payload metadata so that StorageDriver.Retrieve can locate the data later.
// Drivers encode their own addressing information (e.g. a bucket name and
// object key) into the Data map.
//
// NOTE: Experimental
type StorageDriverClaim struct {
	ClaimData map[string]string `json:"claim_data"`
}

// StorageDriver is the interface that must be implemented to back external
// payload storage. When a payload exceeds the configured size threshold the SDK
// calls Store instead of embedding it in the Temporal history event. On the
// read path the SDK calls Retrieve using the claim that was persisted with the
// event, transparently restoring the original payload before it is decoded by
// the data converter.
//
// NOTE: Experimental
type StorageDriver interface {
	// Name returns a stable, unique identifier for this driver instance. The
	// name is stored in the Temporal history alongside each claim so that the
	// correct driver is selected on the retrieval path. Multiple instances of
	// the same driver type can be registered simultaneously under different
	// names, allowing a DriverSelector to route payloads to different backends
	// of the same kind. Changing the name of a deployed driver will cause
	// retrieval to fail for any payloads that were stored under the old name.
	Name() string

	// Type identifies the driver implementation. Unlike Name, Type must be
	// identical across all instances of the same driver type regardless of how
	// they are configured or named.
	Type() string

	// Store persists the given payloads and returns one StorageDriverClaim per
	// payload in the same order. The returned claims are serialized into the
	// Temporal event and must contain enough information for Retrieve to locate
	// the data. Store must not modify the input payloads.
	Store(ctx StorageDriverStoreContext, payloads []*commonpb.Payload) ([]StorageDriverClaim, error)

	// Retrieve fetches the payloads identified by the given claims, returning
	// one payload per claim in the same order. It must not modify the input
	// claims.
	Retrieve(ctx StorageDriverRetrieveContext, claims []StorageDriverClaim) ([]*commonpb.Payload, error)
}

// StorageDriverSelector chooses which StorageDriver should store a given
// payload, or returns nil to leave the payload inline (not stored externally).
// Use this when different payloads should be routed to different backends. For
// example, routing large binary blobs to object storage while keeping small
// JSON payloads inline. If no selector is set, the first driver in
// ExternalStorage.Drivers is used for every payload that exceeds the size
// threshold.
//
// NOTE: Experimental
type StorageDriverSelector interface {
	SelectDriver(ctx StorageDriverStoreContext, payloads *commonpb.Payload) (StorageDriver, error)
}

// ExternalStorage configures external payload storage for a Temporal client or
// worker. When set, the SDK intercepts payloads on the way to and from the
// Temporal server: payloads that exceed PayloadSizeThreshold are offloaded to
// an external store by the configured driver(s), and storage references are
// substituted in the history event. The references are resolved back to the
// original payloads before they reach the data converter or application code.
//
// NOTE: Experimental
type ExternalStorage struct {
	// Drivers is the list of available storage drivers. At least one driver
	// must be provided. If no DriverSelector is set, the first driver is used
	// for all payloads that exceed the size threshold. All drivers listed here
	// must have unique names; duplicates are rejected at client/worker
	// construction time.
	Drivers []StorageDriver

	// DriverSelector routes each payload to a specific driver, or returns nil
	// to leave the payload inline. When DriverSelector itself is nil, Drivers[0]
	// is used for every oversized payload. The selector is invoked after the
	// payload has been serialized by the data converter, so it can inspect
	// encoding metadata and raw size to make routing decisions.
	DriverSelector StorageDriverSelector

	// PayloadSizeThreshold is the minimum serialized payload size in bytes that
	// triggers external storage. Payloads smaller than this value are left
	// inline in the Temporal history event. A value of zero uses the default
	// threshold of 256 KiB. Negative values are rejected at client/worker
	// construction time.
	PayloadSizeThreshold int
}
