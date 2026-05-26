package converter

import "go.temporal.io/sdk/internal/extstore"

// StorageDriverTargetInfo identifies the workflow or activity on whose behalf
// a payload is being stored. Use a type switch on [StorageDriverWorkflowInfo]
// and [StorageDriverActivityInfo] to access the concrete values.
//
// NOTE: Experimental
type StorageDriverTargetInfo = extstore.StorageDriverTargetInfo

// StorageDriverWorkflowInfo carries workflow identity for a storage operation.
//
// NOTE: Experimental
type StorageDriverWorkflowInfo = extstore.StorageDriverWorkflowInfo

// StorageDriverActivityInfo carries activity identity for a storage operation.
// This is only used for standalone (non-workflow-bound) activities; activities
// started by a workflow use [StorageDriverWorkflowInfo] as the target.
//
// NOTE: Experimental
type StorageDriverActivityInfo = extstore.StorageDriverActivityInfo

// StorageDriverStoreContext carries context passed to StorageDriver.Store and
// StorageDriverSelector.SelectDriver operations.
//
// NOTE: Experimental
type StorageDriverStoreContext = extstore.StorageDriverStoreContext

// StorageDriverRetrieveContext carries context passed to StorageDriver.Retrieve
// operations.
//
// NOTE: Experimental
type StorageDriverRetrieveContext = extstore.StorageDriverRetrieveContext

// StorageDriverClaim is an opaque token returned by StorageDriver.Store that
// identifies where a payload was stored. The SDK serializes it alongside the
// payload metadata so that StorageDriver.Retrieve can locate the data later.
// Drivers encode their own addressing information (e.g. a bucket name and
// object key) into the ClaimData map.
//
// NOTE: Experimental
type StorageDriverClaim = extstore.StorageDriverClaim

// StorageDriver is the interface that must be implemented to back external
// payload storage. When a payload exceeds the configured size threshold the SDK
// calls Store instead of embedding it in the Temporal history event. On the
// read path the SDK calls Retrieve using the claim that was persisted with the
// event, transparently restoring the original payload before it is decoded by
// the data converter.
//
// NOTE: Experimental
type StorageDriver = extstore.StorageDriver

// StorageDriverSelector chooses which StorageDriver should store a given
// payload, or returns nil to leave the payload inline (not stored externally).
// Use this when different payloads should be routed to different backends. For
// example, routing large binary blobs to object storage while keeping small
// JSON payloads inline. If no selector is set, the first driver in
// ExternalStorage.Drivers is used for every payload that exceeds the size
// threshold.
//
// NOTE: Experimental
type StorageDriverSelector = extstore.StorageDriverSelector

// ExternalStorage configures external payload storage for a Temporal client or
// worker. When set, the SDK intercepts payloads on the way to and from the
// Temporal server: payloads that exceed PayloadSizeThreshold are offloaded to
// an external store by the configured driver(s), and storage references are
// substituted in the history event. The references are resolved back to the
// original payloads before they reach the data converter or application code.
//
// NOTE: Experimental
type ExternalStorage = extstore.ExternalStorage
