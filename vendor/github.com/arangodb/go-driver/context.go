//
// DISCLAIMER
//
// Copyright 2017-2025 ArangoDB GmbH, Cologne, Germany
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Copyright holder is ArangoDB GmbH, Cologne, Germany
//

package driver

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/arangodb/go-driver/util"
)

// ContextKey is an internal type used for holding values in a `context.Context`
// do not use!.
type ContextKey string

const (
	keyRevision                 ContextKey = "arangodb-revision"
	keyRevisions                ContextKey = "arangodb-revisions"
	keyReturnNew                ContextKey = "arangodb-returnNew"
	keyReturnOld                ContextKey = "arangodb-returnOld"
	keySilent                   ContextKey = "arangodb-silent"
	keyWaitForSync              ContextKey = "arangodb-waitForSync"
	keyDetails                  ContextKey = "arangodb-details"
	keyKeepNull                 ContextKey = "arangodb-keepNull"
	keyMergeObjects             ContextKey = "arangodb-mergeObjects"
	keyRawResponse              ContextKey = "arangodb-rawResponse"
	keyImportDetails            ContextKey = "arangodb-importDetails"
	keyResponse                 ContextKey = "arangodb-response"
	keyEndpoint                 ContextKey = "arangodb-endpoint"
	keyIsRestore                ContextKey = "arangodb-isRestore"
	keyIsSystem                 ContextKey = "arangodb-isSystem"
	keyIgnoreRevs               ContextKey = "arangodb-ignoreRevs"
	keyEnforceReplicationFactor ContextKey = "arangodb-enforceReplicationFactor"
	keyConfigured               ContextKey = "arangodb-configured"
	keyFollowLeaderRedirect     ContextKey = "arangodb-followLeaderRedirect"
	keyDBServerID               ContextKey = "arangodb-dbserverID"
	keyBatchID                  ContextKey = "arangodb-batchID"
	keyJobIDResponse            ContextKey = "arangodb-jobIDResponse"
	keyAllowDirtyReads          ContextKey = "arangodb-allowDirtyReads"
	keyTransactionID            ContextKey = "arangodb-transactionID"
	keyOverwriteMode            ContextKey = "arangodb-overwriteMode"
	keyOverwrite                ContextKey = "arangodb-overwrite"
	keyUseQueueTimeout          ContextKey = "arangodb-use-queue-timeout"
	keyMaxQueueTime             ContextKey = "arangodb-max-queue-time-seconds"
	keyDropCollections          ContextKey = "arangodb-drop-collections"
	keyDriverFlags              ContextKey = "arangodb-driver-flags"
	keyRefillIndexCaches        ContextKey = "arangodb-driver-refill-index-caches"
	keyAsyncRequest             ContextKey = "arangodb-async-request"
	keyAsyncID                  ContextKey = "arangodb-async-id"
	keySkipExistCheck           ContextKey = "arangodb-skip-exist-check"
)

type OverwriteMode string

const (
	OverwriteModeIgnore   OverwriteMode = "ignore"
	OverwriteModeReplace  OverwriteMode = "replace"
	OverwriteModeUpdate   OverwriteMode = "update"
	OverwriteModeConflict OverwriteMode = "conflict"
)

// WithRevision is used to configure a context to make document
// functions specify an explicit revision of the document using an `If-Match` condition.
func WithRevision(parent context.Context, revision string) context.Context {
	return context.WithValue(contextOrBackground(parent), keyRevision, revision)
}

// WithRevisions is used to configure a context to make multi-document
// functions specify explicit revisions of the documents.
func WithRevisions(parent context.Context, revisions []string) context.Context {
	return context.WithValue(contextOrBackground(parent), keyRevisions, revisions)
}

// WithReturnNew is used to configure a context to make create, update & replace document
// functions return the new document into the given result.
func WithReturnNew(parent context.Context, result interface{}) context.Context {
	return context.WithValue(contextOrBackground(parent), keyReturnNew, result)
}

// WithReturnOld is used to configure a context to make update & replace document
// functions return the old document into the given result.
func WithReturnOld(parent context.Context, result interface{}) context.Context {
	return context.WithValue(contextOrBackground(parent), keyReturnOld, result)
}

// WithDetails is used to configure a context to make Client.Version return additional details.
// You can pass a single (optional) boolean. If that is set to false, you explicitly ask to not provide details.
func WithDetails(parent context.Context, value ...bool) context.Context {
	v := true
	if len(value) == 1 {
		v = value[0]
	}
	return context.WithValue(contextOrBackground(parent), keyDetails, v)
}

// WithEndpoint is used to configure a context that forces a request to be executed on a specific endpoint.
// If you specify an endpoint like this, failover is disabled.
// If you specify an unknown endpoint, an InvalidArgumentError is returned from requests.
func WithEndpoint(parent context.Context, endpoint string) context.Context {
	endpoint = util.FixupEndpointURLScheme(endpoint)
	return context.WithValue(contextOrBackground(parent), keyEndpoint, endpoint)
}

// WithKeepNull is used to configure a context to make update functions keep null fields (value==true)
// or remove fields with null values (value==false).
func WithKeepNull(parent context.Context, value bool) context.Context {
	return context.WithValue(contextOrBackground(parent), keyKeepNull, value)
}

// WithMergeObjects is used to configure a context to make update functions merge objects present in both
// the existing document and the patch document (value==true) or overwrite objects in the existing document
// with objects found in the patch document (value==false)
func WithMergeObjects(parent context.Context, value bool) context.Context {
	return context.WithValue(contextOrBackground(parent), keyMergeObjects, value)
}

// WithSilent is used to configure a context to make functions return an empty result (silent==true),
// instead of a metadata result (silent==false, default).
// You can pass a single (optional) boolean. If that is set to false, you explicitly ask to return metadata result.
func WithSilent(parent context.Context, value ...bool) context.Context {
	v := true
	if len(value) == 1 {
		v = value[0]
	}
	return context.WithValue(contextOrBackground(parent), keySilent, v)
}

// WithWaitForSync is used to configure a context to make modification
// functions wait until the data has been synced to disk (or not).
// You can pass a single (optional) boolean. If that is set to false, you explicitly do not wait for
// data to be synced to disk.
func WithWaitForSync(parent context.Context, value ...bool) context.Context {
	v := true
	if len(value) == 1 {
		v = value[0]
	}
	return context.WithValue(contextOrBackground(parent), keyWaitForSync, v)
}

// WithAllowDirtyReads is used in an active failover deployment to allow reads from the follower.
// You can pass a reference to a boolean that will set according to whether a potentially dirty read
// happened or not. nil is allowed.
// This is valid for document reads, aql queries, gharial vertex and edge reads.
// Since 3.10 This feature is available in the Enterprise Edition for cluster deployments as well
func WithAllowDirtyReads(parent context.Context, wasDirtyRead *bool) context.Context {
	return context.WithValue(contextOrBackground(parent), keyAllowDirtyReads, wasDirtyRead)
}

// WithArangoQueueTimeout is used to enable Queue timeout on the server side.
// If WithArangoQueueTime is used then its value takes precedence in other case value of ctx.Deadline will be taken
func WithArangoQueueTimeout(parent context.Context, useQueueTimeout bool) context.Context {
	return context.WithValue(contextOrBackground(parent), keyUseQueueTimeout, useQueueTimeout)
}

// WithArangoQueueTime defines max queue timeout on the server side.
func WithArangoQueueTime(parent context.Context, duration time.Duration) context.Context {
	return context.WithValue(contextOrBackground(parent), keyMaxQueueTime, duration)
}

// WithRawResponse is used to configure a context that will make all functions store the raw response into a
// buffer.
func WithRawResponse(parent context.Context, value *[]byte) context.Context {
	return context.WithValue(contextOrBackground(parent), keyRawResponse, value)
}

// WithResponse is used to configure a context that will make all functions store the response into the given value.
func WithResponse(parent context.Context, value *Response) context.Context {
	return context.WithValue(contextOrBackground(parent), keyResponse, value)
}

// WithImportDetails is used to configure a context that will make import document requests return
// details about documents that could not be imported.
func WithImportDetails(parent context.Context, value *[]string) context.Context {
	return context.WithValue(contextOrBackground(parent), keyImportDetails, value)
}

// WithIsRestore is used to configure a context to make insert functions use the "isRestore=<value>"
// setting.
// Note: This function is intended for internal (replication) use. It is NOT intended to
// be used by normal client. This CAN screw up your database.
func WithIsRestore(parent context.Context, value bool) context.Context {
	return context.WithValue(contextOrBackground(parent), keyIsRestore, value)
}

// WithIsSystem is used to configure a context to make insert functions use the "isSystem=<value>"
// setting.
func WithIsSystem(parent context.Context, value bool) context.Context {
	return context.WithValue(contextOrBackground(parent), keyIsSystem, value)
}

// WithIgnoreRevisions is used to configure a context to make modification
// functions ignore revisions in the update.
// Do not use in combination with WithRevision or WithRevisions.
func WithIgnoreRevisions(parent context.Context, value ...bool) context.Context {
	v := true
	if len(value) == 1 {
		v = value[0]
	}
	return context.WithValue(contextOrBackground(parent), keyIgnoreRevs, v)
}

// WithEnforceReplicationFactor is used to configure a context to make adding collections
// fail if the replication factor is too high (default or true) or
// silently accept (false).
func WithEnforceReplicationFactor(parent context.Context, value bool) context.Context {
	return context.WithValue(contextOrBackground(parent), keyEnforceReplicationFactor, value)
}

// WithConfigured is used to configure a context to return the configured value of
// a user grant instead of the effective grant.
func WithConfigured(parent context.Context, value ...bool) context.Context {
	v := true
	if len(value) == 1 {
		v = value[0]
	}
	return context.WithValue(contextOrBackground(parent), keyConfigured, v)
}

// WithFollowLeaderRedirect is used to configure a context to return turn on/off
// following redirection responses from the server when the request is answered by a follower.
// Default behavior is "on".
func WithFollowLeaderRedirect(parent context.Context, value bool) context.Context {
	return context.WithValue(contextOrBackground(parent), keyFollowLeaderRedirect, value)
}

// WithDBServerID is used to configure a context that includes an ID of a specific DBServer.
func WithDBServerID(parent context.Context, id string) context.Context {
	return context.WithValue(contextOrBackground(parent), keyDBServerID, id)
}

// WithBatchID is used to configure a context that includes an ID of a Batch.
// This is used in replication functions.
func WithBatchID(parent context.Context, id string) context.Context {
	return context.WithValue(contextOrBackground(parent), keyBatchID, id)
}

// WithJobIDResponse is used to configure a context that includes a reference to a JobID
// that is filled on a error-free response.
// This is used in cluster functions.
func WithJobIDResponse(parent context.Context, jobID *string) context.Context {
	return context.WithValue(contextOrBackground(parent), keyJobIDResponse, jobID)
}

// WithTransactionID is used to bind a request to a specific transaction
func WithTransactionID(parent context.Context, tid TransactionID) context.Context {
	return context.WithValue(contextOrBackground(parent), keyTransactionID, tid)
}

// WithOverwriteMode is used to configure a context to instruct how a document should be overwritten.
func WithOverwriteMode(parent context.Context, mode OverwriteMode) context.Context {
	return context.WithValue(contextOrBackground(parent), keyOverwriteMode, mode)
}

// WithOverwrite is used to configure a context to instruct if a document should be overwritten.
func WithOverwrite(parent context.Context) context.Context {
	return context.WithValue(contextOrBackground(parent), keyOverwrite, true)
}

// WithDropCollections is used to configure a context to make graph removal functions to also drop the collections of the graph instead only the graph definition.
// You can pass a single (optional) boolean. If that is set to true, you explicitly ask to also drop the collections of the graph.
func WithDropCollections(parent context.Context, value ...bool) context.Context {
	v := true
	if len(value) == 1 {
		v = value[0]
	}
	return context.WithValue(contextOrBackground(parent), keyDropCollections, v)
}

// WithDriverFlags is used to configure additional flags for the `x-arango-driver` header.
func WithDriverFlags(parent context.Context, value []string) context.Context {
	return context.WithValue(contextOrBackground(parent), keyDriverFlags, value)
}

// WithRefillIndexCaches is used to refill index caches during AQL operations.
func WithRefillIndexCaches(parent context.Context, value bool) context.Context {
	return context.WithValue(contextOrBackground(parent), keyRefillIndexCaches, value)
}

// WithAsync is used to configure a context to make an async operation - requires Connection with Async wrapper!
func WithAsync(parent context.Context) context.Context {
	return context.WithValue(contextOrBackground(parent), keyAsyncRequest, true)
}

// WithAsyncID is used to check an async operation result - requires Connection with Async wrapper!
func WithAsyncID(parent context.Context, asyncID string) context.Context {
	return context.WithValue(contextOrBackground(parent), keyAsyncID, asyncID)
}

// WithSkipExistCheck is used to disable validation for resource existence
// e.g.: ClientDatabases.Database will do not call the additional check to ArangoDB for ensuring that DB exist
func WithSkipExistCheck(parent context.Context, value bool) context.Context {
	return context.WithValue(contextOrBackground(parent), keySkipExistCheck, value)
}

type contextSettings struct {
	Silent                   bool
	WaitForSync              bool
	ReturnOld                interface{}
	ReturnNew                interface{}
	Revision                 string
	Revisions                []string
	ImportDetails            *[]string
	IsRestore                bool
	IsSystem                 bool
	AllowDirtyReads          bool
	DirtyReadFlag            *bool
	IgnoreRevs               *bool
	EnforceReplicationFactor *bool
	Configured               *bool
	FollowLeaderRedirect     *bool
	DBServerID               string
	BatchID                  string
	JobIDResponse            *string
	OverwriteMode            OverwriteMode
	Overwrite                bool
	QueueTimeout             bool
	MaxQueueTime             time.Duration
	DropCollections          *bool
	RefillIndexCaches        *bool
}

// loadContextResponseValue loads generic values from the response and puts it into variables specified
// via context values.
func loadContextResponseValues(cs contextSettings, resp Response) {
	// Parse potential dirty read
	if cs.DirtyReadFlag != nil {
		if dirtyRead := resp.Header("X-Arango-Potential-Dirty-Read"); dirtyRead != "" {
			*cs.DirtyReadFlag = true // The documentation does not say anything about the actual value (dirtyRead == "true")
		} else {
			*cs.DirtyReadFlag = false
		}
	}
}

// setDirtyReadFlagIfRequired is a helper function that sets the bool reference for allowDirtyReads to the
// specified value, if required and reference is not nil.
func setDirtyReadFlagIfRequired(ctx context.Context, wasDirty bool) {
	if v := ctx.Value(keyAllowDirtyReads); v != nil {
		if ref, ok := v.(*bool); ok && ref != nil {
			*ref = wasDirty
		}
	}
}

// ApplyVersionHeader adds the driver version to the request.
func ApplyVersionHeader(ctx context.Context, req Request) {
	val := "go-driver-v1/" + DriverVersion()
	if ctx != nil {
		if v := ctx.Value(keyDriverFlags); v != nil {
			if flags, ok := v.([]string); ok {
				val = fmt.Sprintf("%s (%s)", val, strings.Join(flags, ","))
			}
		}
	}
	req.SetHeader("x-arango-driver", val)
}

// applyContextSettings returns the settings configured in the context in the given request.
// It then returns information about the applied settings that may be needed later in API implementation functions.
func applyContextSettings(ctx context.Context, req Request) contextSettings {
	result := contextSettings{}
	if ctx == nil {
		return result
	}

	// Details
	if v := ctx.Value(keyDetails); v != nil {
		if details, ok := v.(bool); ok {
			req.SetQuery("details", strconv.FormatBool(details))
		}
	}
	// KeepNull
	if v := ctx.Value(keyKeepNull); v != nil {
		if keepNull, ok := v.(bool); ok {
			req.SetQuery("keepNull", strconv.FormatBool(keepNull))
		}
	}
	// MergeObjects
	if v := ctx.Value(keyMergeObjects); v != nil {
		if mergeObjects, ok := v.(bool); ok {
			req.SetQuery("mergeObjects", strconv.FormatBool(mergeObjects))
		}
	}
	// Silent
	if v := ctx.Value(keySilent); v != nil {
		if silent, ok := v.(bool); ok {
			req.SetQuery("silent", strconv.FormatBool(silent))
			result.Silent = silent
		}
	}
	// WaitForSync
	if v := ctx.Value(keyWaitForSync); v != nil {
		if waitForSync, ok := v.(bool); ok {
			req.SetQuery("waitForSync", strconv.FormatBool(waitForSync))
			result.WaitForSync = waitForSync
		}
	}
	// AllowDirtyReads
	if v := ctx.Value(keyAllowDirtyReads); v != nil {
		req.SetHeader("x-arango-allow-dirty-read", "true")
		result.AllowDirtyReads = true
		if dirtyReadFlag, ok := v.(*bool); ok {
			result.DirtyReadFlag = dirtyReadFlag
		}
	}

	// Enable Queue timeout
	if v := ctx.Value(keyUseQueueTimeout); v != nil {
		if useQueueTimeout, ok := v.(bool); ok && useQueueTimeout {
			result.QueueTimeout = useQueueTimeout
			if v := ctx.Value(keyMaxQueueTime); v != nil {
				if timeout, ok := v.(time.Duration); ok {
					result.MaxQueueTime = timeout
					req.SetHeader("x-arango-queue-time-seconds", fmt.Sprint(timeout.Seconds()))
				}
			} else if deadline, ok := ctx.Deadline(); ok {
				timeout := deadline.Sub(time.Now())
				req.SetHeader("x-arango-queue-time-seconds", fmt.Sprint(timeout.Seconds()))
			}
		}
	}

	// TransactionID
	if v := ctx.Value(keyTransactionID); v != nil {
		req.SetHeader("x-arango-trx-id", string(v.(TransactionID)))
	}
	// ReturnOld
	if v := ctx.Value(keyReturnOld); v != nil {
		req.SetQuery("returnOld", "true")
		result.ReturnOld = v
	}
	// ReturnNew
	if v := ctx.Value(keyReturnNew); v != nil {
		req.SetQuery("returnNew", "true")
		result.ReturnNew = v
	}
	// If-Match
	if v := ctx.Value(keyRevision); v != nil {
		if rev, ok := v.(string); ok {
			req.SetHeader("If-Match", rev)
			result.Revision = rev
		}
	}
	// Revisions
	if v := ctx.Value(keyRevisions); v != nil {
		if revs, ok := v.([]string); ok {
			req.SetQuery("ignoreRevs", "false")
			result.Revisions = revs
		}
	}
	// ImportDetails
	if v := ctx.Value(keyImportDetails); v != nil {
		if details, ok := v.(*[]string); ok {
			req.SetQuery("details", "true")
			result.ImportDetails = details
		}
	}
	// IsRestore
	if v := ctx.Value(keyIsRestore); v != nil {
		if isRestore, ok := v.(bool); ok {
			req.SetQuery("isRestore", strconv.FormatBool(isRestore))
			result.IsRestore = isRestore
		}
	}
	// IsSystem
	if v := ctx.Value(keyIsSystem); v != nil {
		if isSystem, ok := v.(bool); ok {
			req.SetQuery("isSystem", strconv.FormatBool(isSystem))
			result.IsSystem = isSystem
		}
	}
	// IgnoreRevs
	if v := ctx.Value(keyIgnoreRevs); v != nil {
		if ignoreRevs, ok := v.(bool); ok {
			req.SetQuery("ignoreRevs", strconv.FormatBool(ignoreRevs))
			result.IgnoreRevs = &ignoreRevs
		}
	}
	// EnforeReplicationFactor
	if v := ctx.Value(keyEnforceReplicationFactor); v != nil {
		if enforceReplicationFactor, ok := v.(bool); ok {
			req.SetQuery("enforceReplicationFactor", strconv.FormatBool(enforceReplicationFactor))
			result.EnforceReplicationFactor = &enforceReplicationFactor
		}
	}
	// Configured
	if v := ctx.Value(keyConfigured); v != nil {
		if configured, ok := v.(bool); ok {
			req.SetQuery("configured", strconv.FormatBool(configured))
			result.Configured = &configured
		}
	}
	// FollowLeaderRedirect
	if v := ctx.Value(keyFollowLeaderRedirect); v != nil {
		if followLeaderRedirect, ok := v.(bool); ok {
			result.FollowLeaderRedirect = &followLeaderRedirect
		}
	}
	// DBServerID
	if v := ctx.Value(keyDBServerID); v != nil {
		if id, ok := v.(string); ok {
			req.SetQuery("DBserver", id)
			result.DBServerID = id
		}
	}
	// BatchID
	if v := ctx.Value(keyBatchID); v != nil {
		if id, ok := v.(string); ok {
			req.SetQuery("batchId", id)
			result.BatchID = id
		}
	}
	// JobIDResponse
	if v := ctx.Value(keyJobIDResponse); v != nil {
		if idRef, ok := v.(*string); ok {
			result.JobIDResponse = idRef
		}
	}
	// OverwriteMode
	if v := ctx.Value(keyOverwriteMode); v != nil {
		if mode, ok := v.(OverwriteMode); ok {
			req.SetQuery("overwriteMode", string(mode))
			result.OverwriteMode = mode
		}
	}
	// DropCollections
	if v := ctx.Value(keyDropCollections); v != nil {
		if dropCollections, ok := v.(bool); ok {
			req.SetQuery("dropCollections", strconv.FormatBool(dropCollections))
			result.DropCollections = &dropCollections
		}
	}
	// IndexCacheRefilling
	if v := ctx.Value(keyRefillIndexCaches); v != nil {
		if local, ok := v.(bool); ok {
			req.SetQuery("refillIndexCaches", strconv.FormatBool(local))
			result.RefillIndexCaches = &local
		}
	}
	// Overwrite
	if v := ctx.Value(keyOverwrite); v != nil {
		if overwrite, ok := v.(bool); ok && overwrite {
			req.SetQuery("overwrite", "true")
			result.Overwrite = true
		}
	}

	// AsyncID
	if v := ctx.Value(keyAsyncID); v != nil {
		if asyncID, ok := v.(string); ok {
			req.SetHeader("x-arango-async-id", asyncID)
		}
	}

	return result
}

// contextOrBackground returns the given context if it is not nil.
// Returns context.Background() otherwise.
func contextOrBackground(ctx context.Context) context.Context {
	if ctx != nil {
		return ctx
	}
	return context.Background()
}

// withDocumentAt returns a context derived from the given parent context to be used in multi-document options
// that needs a client side "loop" implementation.
// It handle:
// - WithRevisions
// - WithReturnNew
// - WithReturnOld
func withDocumentAt(ctx context.Context, index int) (context.Context, error) {
	if ctx == nil {
		return nil, nil
	}
	// Revisions
	if v := ctx.Value(keyRevisions); v != nil {
		if revs, ok := v.([]string); ok {
			if index >= len(revs) {
				return nil, WithStack(InvalidArgumentError{Message: "Index out of range: revisions"})
			}
			ctx = WithRevision(ctx, revs[index])
		}
	}
	// ReturnOld
	if v := ctx.Value(keyReturnOld); v != nil {
		val := reflect.ValueOf(v)
		ctx = WithReturnOld(ctx, val.Index(index).Addr().Interface())
	}
	// ReturnNew
	if v := ctx.Value(keyReturnNew); v != nil {
		val := reflect.ValueOf(v)
		ctx = WithReturnNew(ctx, val.Index(index).Addr().Interface())
	}

	return ctx, nil
}
