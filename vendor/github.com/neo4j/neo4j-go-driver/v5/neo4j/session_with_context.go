/*
 * Copyright (c) "Neo4j"
 * Neo4j Sweden AB [https://neo4j.com]
 *
 * This file is part of Neo4j.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      https://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

package neo4j

import (
	"context"
	"fmt"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/internal/collection"
	idb "github.com/neo4j/neo4j-go-driver/v5/neo4j/internal/db"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/internal/errorutil"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/internal/pool"
	"math"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j/internal/retry"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/log"
)

// TransactionWork represents a unit of work that will be executed against the provided
// transaction
// Deprecated: use ManagedTransactionWork instead.
// ManagedTransactionWork is created via the context-aware driver returned
// by NewDriverWithContext.
// TransactionWork will be removed in 6.0.
type TransactionWork func(tx Transaction) (any, error)

// ManagedTransactionWork represents a unit of work that will be executed against the provided
// transaction
type ManagedTransactionWork func(tx ManagedTransaction) (any, error)

// SessionWithContext represents a logical connection (which is not tied to a physical connection)
// to the server
type SessionWithContext interface {
	// LastBookmarks returns the bookmark received following the last successfully completed transaction.
	// If no bookmark was received or if this transaction was rolled back, the initial set of bookmarks will be
	// returned.
	LastBookmarks() Bookmarks
	lastBookmark() string
	// BeginTransaction starts a new explicit transaction on this session
	BeginTransaction(ctx context.Context, configurers ...func(*TransactionConfig)) (ExplicitTransaction, error)
	// ExecuteRead executes the given unit of work in a AccessModeRead transaction with
	// retry logic in place
	ExecuteRead(ctx context.Context, work ManagedTransactionWork, configurers ...func(*TransactionConfig)) (any, error)
	// ExecuteWrite executes the given unit of work in a AccessModeWrite transaction with
	// retry logic in place
	ExecuteWrite(ctx context.Context, work ManagedTransactionWork, configurers ...func(*TransactionConfig)) (any, error)
	// Run executes an auto-commit statement and returns a result
	Run(ctx context.Context, cypher string, params map[string]any, configurers ...func(*TransactionConfig)) (ResultWithContext, error)
	// Close closes any open resources and marks this session as unusable
	Close(ctx context.Context) error

	legacy() Session
	getServerInfo(ctx context.Context) (ServerInfo, error)
}

// SessionConfig is used to configure a new session, its zero value uses safe defaults.
type SessionConfig struct {
	// AccessMode used when using Session.Run and explicit transactions. Used to route query to
	// to read or write servers when running in a cluster. Session.ReadTransaction and Session.WriteTransaction
	// does not rely on this mode.
	AccessMode AccessMode
	// Bookmarks are the initial bookmarks used to ensure that the executing server is at least up
	// to date to the point represented by the latest of the provided bookmarks. After running commands
	// on the session the bookmark can be retrieved with Session.LastBookmark. All commands executing
	// within the same session will automatically use the bookmark from the previous command in the
	// session.
	Bookmarks Bookmarks
	// DatabaseName sets the target database name for the queries executed within the session created with this
	// configuration.
	// Usage of Cypher clauses like USE is not a replacement for this option.
	// Drive​r sends Cypher to the server for processing.
	// This option has no explicit value by default, but it is recommended to set one if the target database is known
	// in advance. This has the benefit of ensuring a consistent target database name throughout the session in a
	// straightforward way and potentially simplifies driver logic as well as reduces network communication resulting
	// in better performance.
	// When no explicit name is set, the driver behavior depends on the connection URI scheme supplied to the driver on
	// instantiation and Bolt protocol version.
	//
	// Specifically, the following applies:
	//
	// - for bolt schemes
	//		queries are dispatched to the server for execution without explicit database name supplied,
	// 		meaning that the target database name for query execution is determined by the server.
	//		It is important to note that the target database may change (even within the same session), for instance if the
	//		user's home database is changed on the server.
	//
	// - for neo4j schemes
	//		providing that Bolt protocol version 4.4, which was introduced with Neo4j server 4.4, or above
	//		is available, the driver fetches the user's home database name from the server on first query execution
	//		within the session and uses the fetched database name explicitly for all queries executed within the session.
	//		This ensures that the database name remains consistent within the given session. For instance, if the user's
	//		home database name is 'movies' and the server supplies it to the driver upon database name fetching for the
	//		session, all queries within that session are executed with the explicit database name 'movies' supplied.
	//		Any change to the user’s home database is reflected only in sessions created after such change takes effect.
	//		This behavior requires additional network communication.
	//		In clustered environments, it is strongly recommended to avoid a single point of failure.
	//		For instance, by ensuring that the connection URI resolves to multiple endpoints.
	//		For older Bolt protocol versions, the behavior is the same as described for the bolt schemes above.
	DatabaseName string
	// FetchSize defines how many records to pull from server in each batch.
	// From Bolt protocol v4 (Neo4j 4+) records can be fetched in batches as compared to fetching
	// all in previous versions.
	//
	// If FetchSize is set to FetchDefault, the driver decides the appropriate size. If set to a positive value
	// that size is used if the underlying protocol supports it otherwise it is ignored.
	//
	// To turn off fetching in batches and always fetch everything, set FetchSize to FetchAll.
	// If a single large result is to be retrieved this is the most performant setting.
	FetchSize int
	// Logging target the session will send its Bolt message traces
	//
	// Possible to use custom logger (implement log.BoltLogger interface) or
	// use neo4j.ConsoleBoltLogger.
	BoltLogger log.BoltLogger
	// ImpersonatedUser sets the Neo4j user that the session will be acting as.
	// If not set, the user configured for the driver will be used.
	//
	// If user impersonation is used, the default database for that impersonated
	// user will be used unless DatabaseName is set.
	//
	// In the former case, when routing is enabled, using impersonation
	// without DatabaseName will cause the driver to query the
	// cluster for the name of the default database of the impersonated user.
	// This is done at the beginning of the session so that queries are routed
	// to the correct cluster member (different databases may have different
	// leaders).
	ImpersonatedUser string
	// BookmarkManager defines a central point to externally supply bookmarks
	// and be notified of bookmark updates per database
	// This is experimental and may be changed or removed without prior notice
	// Since 5.0
	// default: nil (no-op)
	BookmarkManager BookmarkManager
}

// FetchAll turns off fetching records in batches.
const FetchAll = -1

// FetchDefault lets the driver decide fetch size
const FetchDefault = 0

// Connection pool as seen by the session.
type sessionPool interface {
	Borrow(ctx context.Context, serverNames []string, wait bool, boltLogger log.BoltLogger, livenessCheckThreshold time.Duration) (idb.Connection, error)
	Return(ctx context.Context, c idb.Connection) error
	CleanUp(ctx context.Context) error
}

type sessionWithContext struct {
	config           *Config
	defaultMode      idb.AccessMode
	bookmarks        *sessionBookmarks
	databaseName     string
	impersonatedUser string
	resolveHomeDb    bool
	pool             sessionPool
	router           sessionRouter
	explicitTx       *explicitTransaction
	autocommitTx     *autocommitTransaction
	sleep            func(d time.Duration)
	now              func() time.Time
	logId            string
	log              log.Logger
	throttleTime     time.Duration
	fetchSize        int
	boltLogger       log.BoltLogger
}

func newSessionWithContext(config *Config, sessConfig SessionConfig, router sessionRouter, pool sessionPool, logger log.Logger) *sessionWithContext {
	logId := log.NewId()
	logger.Debugf(log.Session, logId, "Created with context")

	fetchSize := config.FetchSize
	if sessConfig.FetchSize != FetchDefault {
		fetchSize = sessConfig.FetchSize
	}

	return &sessionWithContext{
		config:           config,
		router:           router,
		pool:             pool,
		defaultMode:      idb.AccessMode(sessConfig.AccessMode),
		bookmarks:        newSessionBookmarks(sessConfig.BookmarkManager, sessConfig.Bookmarks),
		databaseName:     sessConfig.DatabaseName,
		impersonatedUser: sessConfig.ImpersonatedUser,
		resolveHomeDb:    sessConfig.DatabaseName == "",
		sleep:            time.Sleep,
		now:              time.Now,
		log:              logger,
		logId:            logId,
		throttleTime:     time.Second * 1,
		fetchSize:        fetchSize,
		boltLogger:       sessConfig.BoltLogger,
	}
}

func (s *sessionWithContext) lastBookmark() string {
	// Pick up bookmark from pending auto-commit if there is a bookmark on it
	// Note: the bookmark manager should not be notified here because:
	//  - the results of the autocommit transaction may have not been consumed
	// 	yet, in which case, the underlying connection may have an outdated
	//	cached bookmark value
	//  - moreover, the bookmark manager may already hold newer bookmarks
	// 	because other sessions for the same DB have completed some work in
	//	parallel
	if s.autocommitTx != nil {
		s.retrieveSessionBookmarks(s.autocommitTx.conn)
	}

	// Report bookmark from previously closed connection or from initial set
	return s.bookmarks.lastBookmark()
}

func (s *sessionWithContext) LastBookmarks() Bookmarks {
	// Pick up bookmark from pending auto-commit if there is a bookmark on it
	// Note: the bookmark manager should not be notified here because:
	//  - the results of the autocommit transaction may have not been consumed
	// 	yet, in which case, the underlying connection may have an outdated
	//	cached bookmark value
	//  - moreover, the bookmark manager may already hold newer bookmarks
	// 	because other sessions for the same DB have completed some work in
	//	parallel
	if s.autocommitTx != nil {
		s.retrieveSessionBookmarks(s.autocommitTx.conn)
	}

	// Report bookmarks from previously closed connection or from initial set
	return s.bookmarks.currentBookmarks()
}

func (s *sessionWithContext) BeginTransaction(ctx context.Context, configurers ...func(*TransactionConfig)) (ExplicitTransaction, error) {
	// Guard for more than one transaction per session
	if s.explicitTx != nil {
		err := &UsageError{Message: "Session already has a pending transaction"}
		s.log.Error(log.Session, s.logId, err)
		return nil, err
	}

	if s.autocommitTx != nil {
		s.autocommitTx.done(ctx)
	}

	// Apply configuration functions
	config := defaultTransactionConfig()
	for _, c := range configurers {
		c(&config)
	}
	if err := validateTransactionConfig(config); err != nil {
		return nil, err
	}

	// Get a connection from the pool. This could fail in clustered environment.
	conn, err := s.getConnection(ctx, s.defaultMode, pool.DefaultLivenessCheckThreshold)
	if err != nil {
		return nil, wrapError(err)
	}

	// Begin transaction
	beginBookmarks, err := s.getBookmarks(ctx)
	if err != nil {
		s.pool.Return(ctx, conn)
		return nil, wrapError(err)
	}
	txHandle, err := conn.TxBegin(ctx,
		idb.TxConfig{
			Mode:             s.defaultMode,
			Bookmarks:        beginBookmarks,
			Timeout:          config.Timeout,
			Meta:             config.Metadata,
			ImpersonatedUser: s.impersonatedUser,
		})
	if err != nil {
		s.pool.Return(ctx, conn)
		return nil, wrapError(err)
	}

	// Create transaction wrapper
	s.explicitTx = &explicitTransaction{
		conn:      conn,
		fetchSize: s.fetchSize,
		txHandle:  txHandle,
		onClosed: func(tx *explicitTransaction) {
			// On transaction closed (rolled back or committed)
			bookmarkErr := s.retrieveBookmarks(ctx, conn, beginBookmarks)
			poolErr := s.pool.Return(ctx, conn)
			tx.err = errorutil.CombineAllErrors(tx.err, bookmarkErr, poolErr)
			s.explicitTx = nil
		},
	}

	return s.explicitTx, nil
}

func (s *sessionWithContext) ExecuteRead(ctx context.Context,
	work ManagedTransactionWork, configurers ...func(*TransactionConfig)) (any, error) {

	return s.runRetriable(ctx, idb.ReadMode, work, configurers...)
}

func (s *sessionWithContext) ExecuteWrite(ctx context.Context,
	work ManagedTransactionWork, configurers ...func(*TransactionConfig)) (any, error) {

	return s.runRetriable(ctx, idb.WriteMode, work, configurers...)
}

func (s *sessionWithContext) runRetriable(
	ctx context.Context,
	mode idb.AccessMode,
	work ManagedTransactionWork, configurers ...func(*TransactionConfig)) (any, error) {

	// Guard for more than one transaction per session
	if s.explicitTx != nil {
		err := &UsageError{Message: "Session already has a pending transaction"}
		return nil, err
	}

	if s.autocommitTx != nil {
		s.autocommitTx.done(ctx)
	}

	config := defaultTransactionConfig()
	for _, c := range configurers {
		c(&config)
	}
	if err := validateTransactionConfig(config); err != nil {
		return nil, err
	}

	state := retry.State{
		MaxTransactionRetryTime: s.config.MaxTransactionRetryTime,
		Log:                     s.log,
		LogName:                 log.Session,
		LogId:                   s.logId,
		Now:                     s.now,
		Sleep:                   s.sleep,
		Throttle:                retry.Throttler(s.throttleTime),
		MaxDeadConnections:      s.config.MaxConnectionPoolSize,
		Router:                  s.router,
		DatabaseName:            s.databaseName,
		OnDeadConnection: func(server string) error {
			if mode == idb.WriteMode {
				if err := s.router.InvalidateWriter(ctx, s.databaseName, server); err != nil {
					return err
				}
			} else {
				if err := s.router.InvalidateReader(ctx, s.databaseName, server); err != nil {
					return err
				}
			}
			return nil
		},
	}
	for state.Continue() {
		if tryAgain, result := s.executeTransactionFunction(ctx, mode, config, &state, work); tryAgain {
			continue
		} else {
			return result, nil
		}
	}

	// When retries has occurred wrap the error, the last error is always added but
	// cause is only set when the retry logic could detect something strange.
	if state.LastErrWasRetryable {
		err := newTransactionExecutionLimit(state.Errs, state.Causes)
		s.log.Error(log.Session, s.logId, err)
		return nil, err
	}
	// Wrap and log the error if it belongs to the driver
	err := wrapError(state.LastErr)
	switch err.(type) {
	case *UsageError, *ConnectivityError:
		s.log.Error(log.Session, s.logId, err)
	}
	return nil, err
}

func (s *sessionWithContext) executeTransactionFunction(
	ctx context.Context,
	mode idb.AccessMode,
	config TransactionConfig,
	state *retry.State,
	work ManagedTransactionWork) (bool, any) {

	conn, err := s.getConnection(ctx, mode, pool.DefaultLivenessCheckThreshold)
	if err != nil {
		state.OnFailure(ctx, conn, err, false)
		return true, nil
	}

	// handle transaction function panic as well
	defer s.pool.Return(ctx, conn)

	beginBookmarks, err := s.getBookmarks(ctx)
	if err != nil {
		state.OnFailure(ctx, conn, err, false)
		return true, nil
	}
	txHandle, err := conn.TxBegin(ctx,
		idb.TxConfig{
			Mode:             mode,
			Bookmarks:        beginBookmarks,
			Timeout:          config.Timeout,
			Meta:             config.Metadata,
			ImpersonatedUser: s.impersonatedUser,
		})
	if err != nil {
		state.OnFailure(ctx, conn, err, false)
		return true, nil
	}

	tx := managedTransaction{conn: conn, fetchSize: s.fetchSize, txHandle: txHandle}
	x, err := work(&tx)
	if err != nil {
		// If the client returns a client specific error that means that
		// client wants to rollback. We don't do an explicit rollback here
		// but instead rely on the pool invoking reset on the connection,
		// that will do an implicit rollback.
		state.OnFailure(ctx, conn, err, false)
		return true, nil
	}

	err = conn.TxCommit(ctx, txHandle)
	if err != nil {
		state.OnFailure(ctx, conn, err, true)
		return true, nil
	}

	// transaction has been committed so let's ignore (ie just log) the error
	if err = s.retrieveBookmarks(ctx, conn, beginBookmarks); err != nil {
		s.log.Warnf(log.Session, s.logId, "could not retrieve bookmarks after successful commit: %s\n"+
			"the results of this transaction may not be visible to subsequent operations", err.Error())
	}
	return false, x
}

func (s *sessionWithContext) getServers(ctx context.Context, mode idb.AccessMode) ([]string, error) {
	if mode == idb.ReadMode {
		return s.router.Readers(ctx, s.getBookmarks, s.databaseName, s.boltLogger)
	} else {
		return s.router.Writers(ctx, s.getBookmarks, s.databaseName, s.boltLogger)
	}
}

func (s *sessionWithContext) getConnection(ctx context.Context, mode idb.AccessMode, livenessCheckThreshold time.Duration) (idb.Connection, error) {
	if s.config.ConnectionAcquisitionTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.config.ConnectionAcquisitionTimeout)
		if cancel != nil {
			defer cancel()
		}
		s.log.Debugf(log.Session, s.logId, "connection acquisition timeout is: %s",
			s.config.ConnectionAcquisitionTimeout.String())
		if deadline, ok := ctx.Deadline(); ok {
			s.log.Debugf(log.Session, s.logId, "connection acquisition resolved deadline is: %s",
				deadline.String())
		}
	}

	if err := s.resolveHomeDatabase(ctx); err != nil {
		return nil, wrapError(err)
	}
	servers, err := s.getServers(ctx, mode)
	if err != nil {
		return nil, wrapError(err)
	}

	conn, err := s.pool.Borrow(ctx, servers, s.config.ConnectionAcquisitionTimeout != 0, s.boltLogger, livenessCheckThreshold)
	if err != nil {
		return nil, wrapError(err)
	}

	// Select database on server
	if s.databaseName != idb.DefaultDatabase {
		dbSelector, ok := conn.(idb.DatabaseSelector)
		if !ok {
			s.pool.Return(ctx, conn)
			return nil, &UsageError{Message: "Database does not support multi-database"}
		}
		dbSelector.SelectDatabase(s.databaseName)
	}

	return conn, nil
}

func (s *sessionWithContext) retrieveBookmarks(ctx context.Context, conn idb.Connection, sentBookmarks Bookmarks) error {
	if conn == nil {
		return nil
	}
	return s.bookmarks.replaceBookmarks(ctx, sentBookmarks, conn.Bookmark())
}

func (s *sessionWithContext) retrieveSessionBookmarks(conn idb.Connection) {
	if conn == nil {
		return
	}
	s.bookmarks.replaceSessionBookmarks(conn.Bookmark())
}

func (s *sessionWithContext) Run(ctx context.Context,
	cypher string, params map[string]any, configurers ...func(*TransactionConfig)) (ResultWithContext, error) {

	if s.explicitTx != nil {
		err := &UsageError{Message: "Trying to run auto-commit transaction while in explicit transaction"}
		s.log.Error(log.Session, s.logId, err)
		return nil, err
	}

	if s.autocommitTx != nil {
		s.autocommitTx.done(ctx)
	}

	config := defaultTransactionConfig()
	for _, c := range configurers {
		c(&config)
	}
	if err := validateTransactionConfig(config); err != nil {
		return nil, err
	}

	conn, err := s.getConnection(ctx, s.defaultMode, pool.DefaultLivenessCheckThreshold)
	if err != nil {
		return nil, wrapError(err)
	}

	runBookmarks, err := s.getBookmarks(ctx)
	if err != nil {
		s.pool.Return(ctx, conn)
		return nil, wrapError(err)
	}
	stream, err := conn.Run(
		ctx,
		idb.Command{
			Cypher:    cypher,
			Params:    params,
			FetchSize: s.fetchSize,
		},
		idb.TxConfig{
			Mode:             s.defaultMode,
			Bookmarks:        runBookmarks,
			Timeout:          config.Timeout,
			Meta:             config.Metadata,
			ImpersonatedUser: s.impersonatedUser,
		})
	if err != nil {
		s.pool.Return(ctx, conn)
		return nil, wrapError(err)
	}

	s.autocommitTx = &autocommitTransaction{
		conn: conn,
		res: newResultWithContext(conn, stream, cypher, params, func() {
			if err := s.retrieveBookmarks(ctx, conn, runBookmarks); err != nil {
				s.log.Warnf(log.Session, s.logId, "could not retrieve bookmarks after result consumption: %s\n"+
					"the result of the initiating auto-commit transaction may not be visible to subsequent operations", err.Error())
			}
		}),
		onClosed: func() {
			s.pool.Return(ctx, conn)
			s.autocommitTx = nil
		},
	}

	return s.autocommitTx.res, nil
}

func (s *sessionWithContext) Close(ctx context.Context) error {
	var txErr error
	if s.explicitTx != nil {
		txErr = s.explicitTx.Close(ctx)
	}

	if s.autocommitTx != nil {
		s.autocommitTx.discard(ctx)
	}

	defer s.log.Debugf(log.Session, s.logId, "Closed")
	poolErrChan := make(chan error, 1)
	routerErrChan := make(chan error, 1)
	go func() {
		poolErrChan <- s.pool.CleanUp(ctx)
	}()
	go func() {
		routerErrChan <- s.router.CleanUp(ctx)
	}()
	return errorutil.CombineAllErrors(txErr, <-poolErrChan, <-routerErrChan)
}

func (s *sessionWithContext) legacy() Session {
	return &session{delegate: s}
}

func (s *sessionWithContext) getServerInfo(ctx context.Context) (ServerInfo, error) {
	if err := s.resolveHomeDatabase(ctx); err != nil {
		return nil, wrapError(err)
	}
	servers, err := s.getServers(ctx, idb.ReadMode)
	if err != nil {
		return nil, wrapError(err)
	}
	conn, err := s.pool.Borrow(ctx, servers, s.config.ConnectionAcquisitionTimeout != 0, s.boltLogger, 0)
	if err != nil {
		return nil, wrapError(err)
	}
	defer s.pool.Return(ctx, conn)
	return &simpleServerInfo{
		address:         conn.ServerName(),
		agent:           conn.ServerVersion(),
		protocolVersion: conn.Version(),
	}, nil
}

func (s *sessionWithContext) resolveHomeDatabase(ctx context.Context) error {
	if !s.resolveHomeDb {
		return nil
	}

	bookmarks, err := s.getBookmarks(ctx)
	if err != nil {
		return err
	}
	defaultDb, err := s.router.GetNameOfDefaultDatabase(ctx, bookmarks, s.impersonatedUser, s.boltLogger)
	if err != nil {
		return err
	}
	s.log.Debugf(log.Session, s.logId, "Resolved home database, uses db '%s'", defaultDb)
	s.databaseName = defaultDb
	s.resolveHomeDb = false
	return nil
}

func (s *sessionWithContext) getBookmarks(ctx context.Context) (Bookmarks, error) {
	bookmarks, err := s.bookmarks.getBookmarks(ctx)
	if err != nil {
		return nil, err
	}
	result := collection.NewSet(bookmarks)
	result.AddAll(s.bookmarks.currentBookmarks())
	return result.Values(), nil
}

type erroredSessionWithContext struct {
	err error
}

func (s *erroredSessionWithContext) LastBookmarks() Bookmarks {
	return nil
}

func (s *erroredSessionWithContext) lastBookmark() string {
	return ""
}
func (s *erroredSessionWithContext) BeginTransaction(context.Context, ...func(*TransactionConfig)) (ExplicitTransaction, error) {
	return nil, s.err
}
func (s *erroredSessionWithContext) ExecuteRead(context.Context, ManagedTransactionWork, ...func(*TransactionConfig)) (any, error) {
	return nil, s.err
}
func (s *erroredSessionWithContext) ExecuteWrite(context.Context, ManagedTransactionWork, ...func(*TransactionConfig)) (any, error) {
	return nil, s.err
}
func (s *erroredSessionWithContext) Run(context.Context, string, map[string]any, ...func(*TransactionConfig)) (ResultWithContext, error) {
	return nil, s.err
}
func (s *erroredSessionWithContext) Close(context.Context) error {
	return s.err
}
func (s *erroredSessionWithContext) legacy() Session {
	return &erroredSession{err: s.err}
}
func (s *erroredSessionWithContext) getServerInfo(context.Context) (ServerInfo, error) {
	return nil, s.err
}

func defaultTransactionConfig() TransactionConfig {
	return TransactionConfig{Timeout: math.MinInt, Metadata: nil}
}

func validateTransactionConfig(config TransactionConfig) error {
	if config.Timeout != math.MinInt && config.Timeout < 0 {
		err := fmt.Sprintf("Negative transaction timeouts are not allowed. Given: %d", config.Timeout)
		return &UsageError{Message: err}
	}
	return nil
}
