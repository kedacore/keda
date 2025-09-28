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
// Author Ewout Prangsma
//

package driver

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
)

const (
	// general errors
	ErrNoError               = 0
	ErrFailed                = 1
	ErrSysError              = 2
	ErrOutOfMemory           = 3
	ErrInternal              = 4
	ErrIllegalNumber         = 5
	ErrNumericOverflow       = 6
	ErrIllegalOption         = 7
	ErrDeadPid               = 8
	ErrNotImplemented        = 9
	ErrBadParameter          = 10
	ErrForbidden             = 11
	ErrCorruptedCsv          = 13
	ErrFileNotFound          = 14
	ErrCannotWriteFile       = 15
	ErrCannotOverwriteFile   = 16
	ErrTypeError             = 17
	ErrLockTimeout           = 18
	ErrCannotCreateDirectory = 19
	ErrCannotCreateTempFile  = 20
	ErrRequestCanceled       = 21
	ErrDebug                 = 22
	ErrIpAddressInvalid      = 25
	ErrFileExists            = 27
	ErrLocked                = 28
	ErrDeadlock              = 29
	ErrShuttingDown          = 30
	ErrOnlyEnterprise        = 31
	ErrResourceLimit         = 32
	ErrArangoIcuError        = 33
	ErrCannotReadFile        = 34
	ErrIncompatibleVersion   = 35
	ErrDisabled              = 36
	ErrMalformedJson         = 37
	ErrStartingUp            = 38
	ErrDeserialize           = 39
	ErrEndOfFile             = 40

	// HTTP error status codes
	ErrHttpBadParameter        = 400
	ErrHttpUnauthorized        = 401
	ErrHttpForbidden           = 403
	ErrHttpNotFound            = 404
	ErrHttpMethodNotAllowed    = 405
	ErrHttpNotAcceptable       = 406
	ErrHttpRequestTimeout      = 408
	ErrHttpConflict            = 409
	ErrHttpGone                = 410
	ErrHttpPreconditionFailed  = 412
	ErrHttpEnhanceYourCalm     = 420
	ErrHttpServerError         = 500
	ErrHttpNotImplemented      = 501
	ErrHttpServiceUnavailable  = 503
	ErrHttpGatewayTimeout      = 504
	ErrHttpCorruptedJson       = 600
	ErrHttpSuperfluousSuffices = 601

	// Internal ArangoDB storage errors
	ErrArangoIllegalState        = 1000
	ErrArangoReadOnly            = 1004
	ErrArangoDuplicateIdentifier = 1005

	// External ArangoDB storage errors
	ErrArangoCorruptedDatafile    = 1100
	ErrArangoIllegalParameterFile = 1101
	ErrArangoCorruptedCollection  = 1102
	ErrArangoFilesystemFull       = 1104
	ErrArangoDatadirLocked        = 1107

	// General ArangoDB storage errors
	ErrArangoConflict                   = 1200
	ErrArangoDocumentNotFound           = 1202
	ErrArangoDataSourceNotFound         = 1203
	ErrArangoCollectionParameterMissing = 1204
	ErrArangoDocumentHandleBad          = 1205
	ErrArangoDuplicateName              = 1207
	ErrArangoIllegalName                = 1208
	ErrArangoNoIndex                    = 1209
	ErrArangoUniqueConstraintViolated   = 1210
	ErrArangoIndexNotFound              = 1212
	ErrArangoCrossCollectionRequest     = 1213
	ErrArangoIndexHandleBad             = 1214
	ErrArangoDocumentTooLarge           = 1216
	ErrArangoCollectionTypeInvalid      = 1218
	ErrArangoAttributeParserFailed      = 1220
	ErrArangoDocumentKeyBad             = 1221
	ErrArangoDocumentKeyUnexpected      = 1222
	ErrArangoDatadirNotWritable         = 1224
	ErrArangoOutOfKeys                  = 1225
	ErrArangoDocumentKeyMissing         = 1226
	ErrArangoDocumentTypeInvalid        = 1227
	ErrArangoDatabaseNotFound           = 1228
	ErrArangoDatabaseNameInvalid        = 1229
	ErrArangoUseSystemDatabase          = 1230
	ErrArangoInvalidKeyGenerator        = 1232
	ErrArangoInvalidEdgeAttribute       = 1233
	ErrArangoIndexCreationFailed        = 1235
	ErrArangoCollectionTypeMismatch     = 1237
	ErrArangoCollectionNotLoaded        = 1238
	ErrArangoDocumentRevBad             = 1239
	ErrArangoIncompleteRead             = 1240
	ErrArangoOldRocksdbFormat           = 1241
	ErrArangoIndexHasLegacySortedKeys   = 1242

	ErrArangoEmptyDatadir    = 1301
	ErrArangoTryAgain        = 1302
	ErrArangoBusy            = 1303
	ErrArangoMergeInProgress = 1304
	ErrArangoIoError         = 1305

	// ArangoDB cluster errors
	ErrReplicationNoResponse                            = 1400
	ErrReplicationInvalidResponse                       = 1401
	ErrReplicationLeaderError                           = 1402
	ErrReplicationLeaderIncompatible                    = 1403
	ErrReplicationLeaderChange                          = 1404
	ErrReplicationLoop                                  = 1405
	ErrReplicationUnexpectedMarker                      = 1406
	ErrReplicationInvalidApplierState                   = 1407
	ErrReplicationUnexpectedTransaction                 = 1408
	ErrReplicationShardSyncAttemptTimeoutExceeded       = 1409
	ErrReplicationInvalidApplierConfiguration           = 1410
	ErrReplicationRunning                               = 1411
	ErrReplicationApplierStopped                        = 1412
	ErrReplicationNoStartTick                           = 1413
	ErrReplicationStartTickNotPresent                   = 1414
	ErrReplicationWrongChecksum                         = 1416
	ErrReplicationShardNonempty                         = 1417
	ErrReplicationReplicatedLogNotFound                 = 1418
	ErrReplicationReplicatedLogNotTheLeader             = 1419
	ErrReplicationReplicatedLogNotAFollower             = 1420
	ErrReplicationReplicatedLogAppendEntriesRejected    = 1421
	ErrReplicationReplicatedLogLeaderResigned           = 1422
	ErrReplicationReplicatedLogFollowerResigned         = 1423
	ErrReplicationReplicatedLogParticipantGone          = 1424
	ErrReplicationReplicatedLogInvalidTerm              = 1425
	ErrReplicationReplicatedLogUnconfigured             = 1426
	ErrReplicationReplicatedStateNotFound               = 1427
	ErrReplicationReplicatedStateNotAvailable           = 1428
	ErrReplicationWriteConcernNotFulfilled              = 1429
	ErrReplicationReplicatedLogSubsequentFault          = 1430
	ErrReplicationReplicatedStateImplementationNotFound = 1431
	ErrReplicationReplicatedWalError                    = 1432
	ErrReplicationReplicatedWalInvalidFile              = 1433
	ErrReplicationReplicatedWalCorrupt                  = 1434
	ErrClusterNotFollower                               = 1446
	ErrClusterFollowerTransactionCommitPerformed        = 1447
	ErrClusterCreateCollectionPreconditionFailed        = 1448
	ErrClusterServerUnknown                             = 1449
	ErrClusterTooManyShards                             = 1450
	ErrClusterCouldNotCreateCollectionInPlan            = 1454
	ErrClusterCouldNotCreateCollection                  = 1456
	ErrClusterTimeout                                   = 1457
	ErrClusterCouldNotRemoveCollectionInPlan            = 1458
	ErrClusterCouldNotCreateDatabaseInPlan              = 1460
	ErrClusterCouldNotCreateDatabase                    = 1461
	ErrClusterCouldNotRemoveDatabaseInPlan              = 1462
	ErrClusterCouldNotRemoveDatabaseInCurrent           = 1463
	ErrClusterShardGone                                 = 1464
	ErrClusterConnectionLost                            = 1465
	ErrClusterMustNotSpecifyKey                         = 1466
	ErrClusterGotContradictingAnswers                   = 1467
	ErrClusterNotAllShardingAttributesGiven             = 1468
	ErrClusterMustNotChangeShardingAttributes           = 1469
	ErrClusterUnsupported                               = 1470
	ErrClusterOnlyOnCoordinator                         = 1471
	ErrClusterReadingPlanAgency                         = 1472
	ErrClusterAqlCommunication                          = 1474
	ErrClusterOnlyOnDbserver                            = 1477
	ErrClusterBackendUnavailable                        = 1478
	ErrClusterAqlCollectionOutOfSync                    = 1481
	ErrClusterCouldNotCreateIndexInPlan                 = 1482
	ErrClusterCouldNotDropIndexInPlan                   = 1483
	ErrClusterChainOfDistributeshardslike               = 1484
	ErrClusterMustNotDropCollOtherDistributeshardslike  = 1485
	ErrClusterUnknownDistributeshardslike               = 1486
	ErrClusterInsufficientDbservers                     = 1487
	ErrClusterCouldNotDropFollower                      = 1488
	ErrClusterShardLeaderRefusesReplication             = 1489
	ErrClusterShardFollowerRefusesOperation             = 1490
	ErrClusterShardLeaderResigned                       = 1491
	ErrClusterAgencyCommunicationFailed                 = 1492
	ErrClusterLeadershipChallengeOngoing                = 1495
	ErrClusterNotLeader                                 = 1496
	ErrClusterCouldNotCreateViewInPlan                  = 1497
	ErrClusterViewIdExists                              = 1498
	ErrClusterCouldNotDropCollection                    = 1499
	ErrQueryKilled                                      = 1500
	ErrQueryParse                                       = 1501
	ErrQueryEmpty                                       = 1502
	ErrQueryScript                                      = 1503
	ErrQueryNumberOutOfRange                            = 1504
	ErrQueryInvalidGeoValue                             = 1505
	ErrQueryVariableNameInvalid                         = 1510
	ErrQueryVariableRedeclared                          = 1511
	ErrQueryVariableNameUnknown                         = 1512
	ErrQueryCollectionLockFailed                        = 1521
	ErrQueryTooManyCollections                          = 1522
	ErrQueryTooMuchNesting                              = 1524
	ErrQueryInvalidOptionsAttribute                     = 1539
	ErrQueryFunctionNameUnknown                         = 1540
	ErrQueryFunctionArgumentNumberMismatch              = 1541
	ErrQueryFunctionArgumentTypeMismatch                = 1542
	ErrQueryInvalidRegex                                = 1543
	ErrQueryBindParametersInvalid                       = 1550
	ErrQueryBindParameterMissing                        = 1551
	ErrQueryBindParameterUndeclared                     = 1552
	ErrQueryBindParameterType                           = 1553
	ErrQueryVectorSearchNotApplied                      = 1554
	ErrQueryInvalidArithmeticValue                      = 1561
	ErrQueryDivisionByZero                              = 1562
	ErrQueryArrayExpected                               = 1563
	ErrQueryCollectionUsedInExpression                  = 1568
	ErrQueryFailCalled                                  = 1569
	ErrQueryGeoIndexMissing                             = 1570
	ErrQueryFulltextIndexMissing                        = 1571
	ErrQueryInvalidDateValue                            = 1572
	ErrQueryMultiModify                                 = 1573
	ErrQueryInvalidAggregateExpression                  = 1574
	ErrQueryCompileTimeOptions                          = 1575
	ErrQueryDnfComplexity                               = 1576
	ErrQueryForcedIndexHintUnusable                     = 1577
	ErrQueryDisallowedDynamicCall                       = 1578
	ErrQueryAccessAfterModification                     = 1579
	ErrQueryFunctionInvalidName                         = 1580
	ErrQueryFunctionInvalidCode                         = 1581
	ErrQueryFunctionNotFound                            = 1582
	ErrQueryFunctionRuntimeError                        = 1583
	ErrQueryNotEligibleForPlanCaching                   = 1584
	ErrQueryBadJsonPlan                                 = 1590
	ErrQueryNotFound                                    = 1591
	ErrQueryUserAssert                                  = 1593
	ErrQueryUserWarn                                    = 1594
	ErrQueryWindowAfterModification                     = 1595
	ErrCursorNotFound                                   = 1600
	ErrCursorBusy                                       = 1601
	ErrValidationFailed                                 = 1620
	ErrValidationBadParameter                           = 1621
	ErrTransactionInternal                              = 1650
	ErrTransactionNested                                = 1651
	ErrTransactionUnregisteredCollection                = 1652
	ErrTransactionDisallowedOperation                   = 1653
	ErrTransactionAborted                               = 1654
	ErrTransactionNotFound                              = 1655

	// User management errors
	ErrUserInvalidName                                          = 1700
	ErrUserDuplicate                                            = 1702
	ErrUserNotFound                                             = 1703
	ErrUserExternal                                             = 1705
	ErrServiceDownloadFailed                                    = 1752
	ErrServiceUploadFailed                                      = 1753
	ErrTaskInvalidId                                            = 1850
	ErrTaskDuplicateId                                          = 1851
	ErrTaskNotFound                                             = 1852
	ErrGraphInvalidGraph                                        = 1901
	ErrGraphInvalidEdge                                         = 1906
	ErrGraphInvalidFilterResult                                 = 1910
	ErrGraphCollectionMultiUse                                  = 1920
	ErrGraphCollectionUseInMultiGraphs                          = 1921
	ErrGraphCreateMissingName                                   = 1922
	ErrGraphCreateMalformedEdgeDefinition                       = 1923
	ErrGraphNotFound                                            = 1924
	ErrGraphDuplicate                                           = 1925
	ErrGraphVertexColDoesNotExist                               = 1926
	ErrGraphWrongCollectionTypeVertex                           = 1927
	ErrGraphNotInOrphanCollection                               = 1928
	ErrGraphCollectionUsedInEdgeDef                             = 1929
	ErrGraphEdgeCollectionNotUsed                               = 1930
	ErrGraphNoGraphCollection                                   = 1932
	ErrGraphInvalidNumberOfArguments                            = 1935
	ErrGraphInvalidParameter                                    = 1936
	ErrGraphCollectionUsedInOrphans                             = 1938
	ErrGraphEdgeColDoesNotExist                                 = 1939
	ErrGraphEmpty                                               = 1940
	ErrGraphInternalDataCorrupt                                 = 1941
	ErrGraphMustNotDropCollection                               = 1942
	ErrGraphCreateMalformedOrphanList                           = 1943
	ErrGraphEdgeDefinitionIsDocument                            = 1944
	ErrGraphCollectionIsInitial                                 = 1945
	ErrGraphNoInitialCollection                                 = 1946
	ErrGraphReferencedVertexCollectionNotPartOfTheGraph         = 1947
	ErrGraphNegativeEdgeWeight                                  = 1948
	ErrGraphCollectionNotPartOfTheGraph                         = 1949
	ErrSessionUnknown                                           = 1950
	ErrSessionExpired                                           = 1951
	ErrSimpleClientUnknownError                                 = 2000
	ErrSimpleClientCouldNotConnect                              = 2001
	ErrSimpleClientCouldNotWrite                                = 2002
	ErrSimpleClientCouldNotRead                                 = 2003
	ErrWasErlaube                                               = 2019
	ErrInternalAql                                              = 2200
	ErrMalformedManifestFile                                    = 3000
	ErrInvalidServiceManifest                                   = 3001
	ErrServiceFilesMissing                                      = 3002
	ErrServiceFilesOutdated                                     = 3003
	ErrInvalidFoxxOptions                                       = 3004
	ErrInvalidMountpoint                                        = 3007
	ErrServiceNotFound                                          = 3009
	ErrServiceNeedsConfiguration                                = 3010
	ErrServiceMountpointConflict                                = 3011
	ErrServiceManifestNotFound                                  = 3012
	ErrServiceOptionsMalformed                                  = 3013
	ErrServiceSourceNotFound                                    = 3014
	ErrServiceSourceError                                       = 3015
	ErrServiceUnknownScript                                     = 3016
	ErrServiceApiDisabled                                       = 3099
	ErrModuleNotFound                                           = 3100
	ErrModuleSyntaxError                                        = 3101
	ErrModuleFailure                                            = 3103
	ErrNoSmartCollection                                        = 4000
	ErrNoSmartGraphAttribute                                    = 4001
	ErrCannotDropSmartCollection                                = 4002
	ErrKeyMustBePrefixedWithSmartGraphAttribute                 = 4003
	ErrIllegalSmartGraphAttribute                               = 4004
	ErrSmartGraphAttributeMismatch                              = 4005
	ErrInvalidSmartJoinAttribute                                = 4006
	ErrKeyMustBePrefixedWithSmartJoinAttribute                  = 4007
	ErrNoSmartJoinAttribute                                     = 4008
	ErrClusterMustNotChangeSmartJoinAttribute                   = 4009
	ErrInvalidDisjointSmartEdge                                 = 4010
	ErrUnsupportedChangeInSmartToSatelliteDisjointEdgeDirection = 4011
	ErrAgencyMalformedGossipMessage                             = 20001
	ErrAgencyMalformedInquireRequest                            = 20002
	ErrAgencyInformMustBeObject                                 = 20011
	ErrAgencyInformMustContainTerm                              = 20012
	ErrAgencyInformMustContainId                                = 20013
	ErrAgencyInformMustContainActive                            = 20014
	ErrAgencyInformMustContainPool                              = 20015
	ErrAgencyInformMustContainMinPing                           = 20016
	ErrAgencyInformMustContainMaxPing                           = 20017
	ErrAgencyInformMustContainTimeoutMult                       = 20018
	ErrAgencyCannotRebuildDbs                                   = 20021
	ErrAgencyMalformedTransaction                               = 20030
	ErrSupervisionGeneralFailure                                = 20501
	ErrQueueFull                                                = 21003
	ErrQueueTimeRequirementViolated                             = 21004
	ErrTooManyDetachedThreads                                   = 21005
	ErrActionUnfinished                                         = 6003
	ErrHotBackupInternal                                        = 7001
	ErrHotRestoreInternal                                       = 7002
	ErrBackupTopology                                           = 7003
	ErrNoSpaceLeftOnDevice                                      = 7004
	ErrFailedToUploadBackup                                     = 7005
	ErrFailedToDownloadBackup                                   = 7006
	ErrNoSuchHotBackup                                          = 7007
	ErrRemoteRepositoryConfigBad                                = 7008
	ErrLocalLockFailed                                          = 7009
	ErrLocalLockRetry                                           = 7010
	ErrHotBackupConflict                                        = 7011
	ErrHotBackupDbserversAwol                                   = 7012
	ErrClusterCouldNotModifyAnalyzersInPlan                     = 7021
	ErrLicenseExpiredOrInvalid                                  = 9001
	ErrLicenseSignatureVerification                             = 9002
	ErrLicenseNonMatchingId                                     = 9003
	ErrLicenseFeatureNotEnabled                                 = 9004
	ErrLicenseResourceExhausted                                 = 9005
	ErrLicenseInvalid                                           = 9006
	ErrLicenseConflict                                          = 9007
	ErrLicenseValidationFailed                                  = 9008
)

// ArangoError is a Go error with arangodb specific error information.
type ArangoError struct {
	HasError     bool   `json:"error"`
	Code         int    `json:"code"`
	ErrorNum     int    `json:"errorNum"`
	ErrorMessage string `json:"errorMessage"`
}

// Error returns the error message of an ArangoError.
func (ae ArangoError) Error() string {
	if ae.ErrorMessage != "" {
		return ae.ErrorMessage
	}
	return fmt.Sprintf("ArangoError: Code %d, ErrorNum %d", ae.Code, ae.ErrorNum)
}

// Timeout returns true when the given error is a timeout error.
func (ae ArangoError) Timeout() bool {
	return ae.HasError && (ae.Code == http.StatusRequestTimeout || ae.Code == http.StatusGatewayTimeout)
}

// Temporary returns true when the given error is a temporary error.
func (ae ArangoError) Temporary() bool {
	return ae.HasError && ae.Code == http.StatusServiceUnavailable
}

// newArangoError creates a new ArangoError with given values.
func newArangoError(code, errorNum int, errorMessage string) error {
	return ArangoError{
		HasError:     true,
		Code:         code,
		ErrorNum:     errorNum,
		ErrorMessage: errorMessage,
	}
}

// IsArangoError returns true when the given error is an ArangoError.
func IsArangoError(err error) bool {
	ae, ok := Cause(err).(ArangoError)
	return ok && ae.HasError
}

// AsArangoError returns true when the given error is an ArangoError together with an object.
func AsArangoError(err error) (ArangoError, bool) {
	ae, ok := Cause(err).(ArangoError)
	if ok {
		return ae, true
	} else {
		return ArangoError{}, false
	}
}

// IsArangoErrorWithCode returns true when the given error is an ArangoError and its Code field is equal to the given code.
func IsArangoErrorWithCode(err error, code int) bool {
	ae, ok := Cause(err).(ArangoError)
	return ok && ae.Code == code
}

// IsArangoErrorWithErrorNum returns true when the given error is an ArangoError and its ErrorNum field is equal to one of the given numbers.
func IsArangoErrorWithErrorNum(err error, errorNum ...int) bool {
	ae, ok := Cause(err).(ArangoError)
	if !ok {
		return false
	}
	for _, x := range errorNum {
		if ae.ErrorNum == x {
			return true
		}
	}
	return false
}

// IsInvalidRequest returns true if the given error is an ArangoError with code 400, indicating an invalid request.
func IsInvalidRequest(err error) bool {
	return IsArangoErrorWithCode(err, http.StatusBadRequest)

}

// IsUnauthorized returns true if the given error is an ArangoError with code 401, indicating an unauthorized request.
func IsUnauthorized(err error) bool {
	return IsArangoErrorWithCode(err, http.StatusUnauthorized)
}

// IsForbidden returns true if the given error is an ArangoError with code 403, indicating a forbidden request.
func IsForbidden(err error) bool {
	return IsArangoErrorWithCode(err, http.StatusForbidden)
}

// Deprecated: Use IsNotFoundGeneral instead.
//
// For ErrArangoDocumentNotFound error there is a chance that we get a different HTTP code if the API requires an existing document as input, which is not found.
//
// IsNotFound returns true if the given error is an ArangoError with code 404, indicating an object not found.
func IsNotFound(err error) bool {
	return IsArangoErrorWithCode(err, http.StatusNotFound) ||
		IsArangoErrorWithErrorNum(err, ErrArangoDocumentNotFound, ErrArangoDataSourceNotFound)
}

// IsNotFoundGeneral returns true if the given error is an ArangoError with code 404, indicating an object is not found.
func IsNotFoundGeneral(err error) bool {
	return IsArangoErrorWithCode(err, http.StatusNotFound)
}

// IsDataSourceOrDocumentNotFound returns true if the given error is an Arango storage error, indicating an object is not found.
func IsDataSourceOrDocumentNotFound(err error) bool {
	return IsArangoErrorWithCode(err, http.StatusNotFound) &&
		IsArangoErrorWithErrorNum(err, ErrArangoDocumentNotFound, ErrArangoDataSourceNotFound)
}

// IsExternalStorageError returns true if ArangoDB is having an error with accessing or writing to storage.
func IsExternalStorageError(err error) bool {
	return IsArangoErrorWithErrorNum(
		err,
		ErrArangoCorruptedDatafile,
		ErrArangoIllegalParameterFile,
		ErrArangoCorruptedCollection,
		ErrArangoFilesystemFull,
		ErrArangoDatadirLocked,
	)
}

// IsConflict returns true if the given error is an ArangoError with code 409, indicating a conflict.
func IsConflict(err error) bool {
	return IsArangoErrorWithCode(err, http.StatusConflict) || IsArangoErrorWithErrorNum(err, ErrUserDuplicate)
}

// IsPreconditionFailed returns true if the given error is an ArangoError with code 412, indicating a failed precondition.
func IsPreconditionFailed(err error) bool {
	return IsArangoErrorWithCode(err, http.StatusPreconditionFailed) ||
		IsArangoErrorWithErrorNum(err, ErrArangoConflict, ErrArangoUniqueConstraintViolated)
}

// IsNoLeader returns true if the given error is an ArangoError with code 503 error number 1496.
func IsNoLeader(err error) bool {
	return IsArangoErrorWithCode(err, http.StatusServiceUnavailable) && IsArangoErrorWithErrorNum(err, ErrClusterNotLeader)
}

// IsNoLeaderOrOngoing return true if the given error is an ArangoError with code 503 and error number 1496 or 1495
func IsNoLeaderOrOngoing(err error) bool {
	return IsArangoErrorWithCode(err, http.StatusServiceUnavailable) &&
		IsArangoErrorWithErrorNum(err, ErrClusterLeadershipChallengeOngoing, ErrClusterNotLeader)
}

// InvalidArgumentError is returned when a go function argument is invalid.
type InvalidArgumentError struct {
	Message string
}

// Error implements the error interface for InvalidArgumentError.
func (e InvalidArgumentError) Error() string {
	return e.Message
}

// IsInvalidArgument returns true if the given error is an InvalidArgumentError.
func IsInvalidArgument(err error) bool {
	_, ok := Cause(err).(InvalidArgumentError)
	return ok
}

// NoMoreDocumentsError is returned by Cursor's, when an attempt is made to read documents when there are no more.
type NoMoreDocumentsError struct{}

// Error implements the error interface for NoMoreDocumentsError.
func (e NoMoreDocumentsError) Error() string {
	return "no more documents"
}

// IsNoMoreDocuments returns true if the given error is an NoMoreDocumentsError.
func IsNoMoreDocuments(err error) bool {
	_, ok := Cause(err).(NoMoreDocumentsError)
	return ok
}

// A ResponseError is returned when a request was completely written to a server, but
// the server did not respond, or some kind of network error occurred during the response.
type ResponseError struct {
	Err error
}

// Error returns the Error() result of the underlying error.
func (e *ResponseError) Error() string {
	return e.Err.Error()
}

// IsResponse returns true if the given error is (or is caused by) a ResponseError.
func IsResponse(err error) bool {
	return isCausedBy(err, func(e error) bool { _, ok := e.(*ResponseError); return ok })
}

// IsCanceled returns true if the given error is the result on a cancelled context.
func IsCanceled(err error) bool {
	return isCausedBy(err, func(e error) bool { return e == context.Canceled })
}

// IsTimeout returns true if the given error is the result on a deadline that has been exceeded.
func IsTimeout(err error) bool {
	return isCausedBy(err, func(e error) bool { return e == context.DeadlineExceeded })
}

// isCausedBy returns true if the given error returns true on the given predicate,
// unwrapping various standard library error wrappers.
func isCausedBy(err error, p func(error) bool) bool {
	if p(err) {
		return true
	}
	err = Cause(err)
	for {
		if p(err) {
			return true
		} else if err == nil {
			return false
		}
		if xerr, ok := err.(*ResponseError); ok {
			err = xerr.Err
		} else if xerr, ok := err.(*url.Error); ok {
			err = xerr.Err
		} else if xerr, ok := err.(*net.OpError); ok {
			err = xerr.Err
		} else if xerr, ok := err.(*os.SyscallError); ok {
			err = xerr.Err
		} else {
			return false
		}
	}
}

var (
	// WithStack is called on every return of an error to add stacktrace information to the error.
	// When setting this function, also set the Cause function.
	// The interface of this function is compatible with functions in github.com/pkg/errors.
	WithStack = func(err error) error { return err }
	// Cause is used to get the root cause of the given error.
	// The interface of this function is compatible with functions in github.com/pkg/errors.
	Cause = func(err error) error { return err }
)

// ErrorSlice is a slice of errors
type ErrorSlice []error

// FirstNonNil returns the first error in the slice that is not nil.
// If all errors in the slice are nil, nil is returned.
func (l ErrorSlice) FirstNonNil() error {
	for _, e := range l {
		if e != nil {
			return e
		}
	}
	return nil
}
