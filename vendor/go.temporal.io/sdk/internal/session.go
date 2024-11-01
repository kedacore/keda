// The MIT License
//
// Copyright (c) 2020 Temporal Technologies Inc.  All rights reserved.
//
// Copyright (c) 2020 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package internal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/pborman/uuid"

	"go.temporal.io/sdk/internal/common/backoff"
)

type (
	// SessionInfo contains information of a created session. For now, exported
	// fields are SessionID and HostName.
	// SessionID is a uuid generated when CreateSession() or RecreateSession()
	// is called and can be used to uniquely identify a session.
	// HostName specifies which host is executing the session
	SessionInfo struct {
		SessionID         string
		HostName          string
		SessionState      SessionState
		resourceID        string     // hide from user for now
		taskqueue         string     // resource specific taskqueue
		sessionCancelFunc CancelFunc // cancel func for the session context, used by both creation activity and user activities
		completionCtx     Context    // context for executing the completion activity
	}

	// SessionOptions specifies metadata for a session.
	// ExecutionTimeout: required, no default
	//     Specifies the maximum amount of time the session can run
	// CreationTimeout: required, no default
	//     Specifies how long session creation can take before returning an error
	// HeartbeatTimeout: optional, default 20s
	//     Specifies the heartbeat timeout. If heartbeat is not received by server
	//     within the timeout, the session will be declared as failed
	SessionOptions struct {
		ExecutionTimeout time.Duration
		CreationTimeout  time.Duration
		HeartbeatTimeout time.Duration
	}

	recreateSessionParams struct {
		Taskqueue string
	}

	SessionState int

	sessionTokenBucket struct {
		*sync.Cond
		availableToken int
	}

	sessionEnvironment interface {
		CreateSession(ctx context.Context, sessionID string) (<-chan struct{}, error)
		CompleteSession(sessionID string)
		AddSessionToken()
		SignalCreationResponse(ctx context.Context, sessionID string) error
		GetResourceSpecificTaskqueue() string
		GetTokenBucket() *sessionTokenBucket
	}

	sessionEnvironmentImpl struct {
		*sync.Mutex
		doneChanMap               map[string]chan struct{}
		resourceID                string
		resourceSpecificTaskqueue string
		sessionTokenBucket        *sessionTokenBucket
	}

	sessionCreationResponse struct {
		Taskqueue  string
		HostName   string
		ResourceID string
	}
)

// Session State enum
const (
	SessionStateOpen SessionState = iota
	SessionStateFailed
	SessionStateClosed
)

const (
	sessionInfoContextKey        contextKey = "sessionInfo"
	sessionEnvironmentContextKey contextKey = "sessionEnvironment"

	sessionCreationActivityName   string = "internalSessionCreationActivity"
	sessionCompletionActivityName string = "internalSessionCompletionActivity"

	errTooManySessionsMsg string = "too many outstanding sessions"

	defaultSessionHeartbeatTimeout = time.Second * 20
	maxSessionHeartbeatInterval    = time.Second * 10
)

var (
	// ErrSessionFailed is the error returned when user tries to execute an activity but the
	// session it belongs to has already failed
	ErrSessionFailed            = errors.New("session has failed")
	errFoundExistingOpenSession = errors.New("found exisiting open session in the context")
)

// Note: Worker should be configured to process session. To do this, set the following
// fields in WorkerOptions:
//     EnableSessionWorker: true
//     SessionResourceID: The identifier of the resource consumed by sessions.
//         It's the user's responsibility to ensure there's only one worker using this resourceID.
//         This option is not available for now as automatic session reestablishing is not implemented.
//     MaxConcurrentSessionExecutionSize: the maximum number of concurrently sessions the resource
//         support. By default, 1000 is used.

// CreateSession creates a session and returns a new context which contains information
// of the created session. The session will be created on the taskqueue user specified in
// ActivityOptions. If none is specified, the default one will be used.
//
// CreationSession will fail in the following situations:
//  1. The context passed in already contains a session which is still open
//     (not closed and failed).
//  2. All the workers are busy (number of sessions currently running on all the workers have reached
//     MaxConcurrentSessionExecutionSize, which is specified when starting the workers) and session
//     cannot be created within a specified timeout.
//
// If an activity is executed using the returned context, it's regarded as part of the
// session. All activities within the same session will be executed by the same worker.
// User still needs to handle the error returned when executing an activity. Session will
// not be marked as failed if an activity within it returns an error. Only when the worker
// executing the session is down, that session will be marked as failed. Executing an activity
// within a failed session will return ErrSessionFailed immediately without scheduling that activity.
//
// The returned session Context will be canceled if the session fails (worker died) or CompleteSession()
// is called. This means that in these two cases, all user activities scheduled using the returned session
// Context will also be canceled.
//
// If user wants to end a session since activity returns some error, use CompleteSession API below.
// New session can be created if necessary to retry the whole session.
//
// Example:
//
//	   so := &SessionOptions{
//		      ExecutionTimeout: time.Minute,
//		      CreationTimeout:  time.Minute,
//	   }
//	   sessionCtx, err := CreateSession(ctx, so)
//	   if err != nil {
//			    // Creation failed. Wrong ctx or too many outstanding sessions.
//	   }
//	   defer CompleteSession(sessionCtx)
//	   err = ExecuteActivity(sessionCtx, someActivityFunc, activityInput).Get(sessionCtx, nil)
//	   if err == ErrSessionFailed {
//	       // Session has failed
//	   } else {
//	       // Handle activity error
//	   }
//	   ... // execute more activities using sessionCtx
func CreateSession(ctx Context, sessionOptions *SessionOptions) (Context, error) {
	options := getActivityOptions(ctx)
	baseTaskqueue := options.TaskQueueName
	if baseTaskqueue == "" {
		baseTaskqueue = options.OriginalTaskQueueName
	}
	return createSession(ctx, getCreationTaskqueue(baseTaskqueue), sessionOptions, true)
}

// RecreateSession recreate a session based on the sessionInfo passed in. Activities executed within
// the recreated session will be executed by the same worker as the previous session. RecreateSession()
// returns an error under the same situation as CreateSession() or the token passed in is invalid.
// It also has the same usage as CreateSession().
//
// The main usage of RecreateSession is for long sessions that are split into multiple runs. At the end of
// one run, complete the current session, get recreateToken from sessionInfo by calling SessionInfo.GetRecreateToken()
// and pass the token to the next run. In the new run, session can be recreated using that token.
func RecreateSession(ctx Context, recreateToken []byte, sessionOptions *SessionOptions) (Context, error) {
	recreateParams, err := deserializeRecreateToken(recreateToken)
	if err != nil {
		return nil, fmt.Errorf("failed to deserilalize recreate token: %v", err)
	}
	return createSession(ctx, recreateParams.Taskqueue, sessionOptions, true)
}

// CompleteSession completes a session. It releases worker resources, so other sessions can be created.
// CompleteSession won't do anything if the context passed in doesn't contain any session information or the
// session has already completed or failed.
//
// After a session has completed, user can continue to use the context, but the activities will be scheduled
// on the normal taskQueue (as user specified in ActivityOptions) and may be picked up by another worker since
// it's not in a session.
func CompleteSession(ctx Context) {
	sessionInfo := getSessionInfo(ctx)
	if sessionInfo == nil || sessionInfo.SessionState != SessionStateOpen {
		return
	}

	// first cancel both the creation activity and all user activities
	// this will cancel the ctx passed into this function
	sessionInfo.sessionCancelFunc()

	// then execute then completion activity using the completionCtx, which is not canceled.
	completionCtx := WithActivityOptions(sessionInfo.completionCtx, ActivityOptions{
		ScheduleToStartTimeout: time.Second * 3,
		StartToCloseTimeout:    time.Second * 3,
	})

	// even though the creation activity has been canceled, the session worker doesn't know. The worker will wait until
	// next heartbeat to figure out that the workflow is completed and then release the resource. We need to make sure the
	// completion activity is executed before the workflow exits.
	// the taskqueue will be overridden to use the one stored in sessionInfo.
	err := ExecuteActivity(completionCtx, sessionCompletionActivityName, sessionInfo.SessionID).Get(completionCtx, nil)
	if err != nil {
		GetLogger(completionCtx).Warn("Complete session activity failed", tagError, err)
	}

	sessionInfo.SessionState = SessionStateClosed
	getWorkflowEnvironment(ctx).RemoveSession(sessionInfo.SessionID)
	GetLogger(ctx).Debug("Completed session", "sessionID", sessionInfo.SessionID)
}

// GetSessionInfo returns the sessionInfo stored in the context. If there are multiple sessions in the context,
// (for example, the same context is used to create, complete, create another session. Then user found that the
// session has failed, and created a new one on it), the most recent sessionInfo will be returned.
//
// This API will return nil if there's no sessionInfo in the context.
func GetSessionInfo(ctx Context) *SessionInfo {
	info := getSessionInfo(ctx)
	if info == nil {
		GetLogger(ctx).Warn("Context contains no session information")
	}
	return info
}

// GetRecreateToken returns the token needed to recreate a session. The returned value should be passed to
// RecreateSession() API.
func (s *SessionInfo) GetRecreateToken() []byte {
	params := recreateSessionParams{
		Taskqueue: s.taskqueue,
	}
	return mustSerializeRecreateToken(&params)
}

func getSessionInfo(ctx Context) *SessionInfo {
	info := ctx.Value(sessionInfoContextKey)
	if info == nil {
		return nil
	}
	return info.(*SessionInfo)
}

func setSessionInfo(ctx Context, sessionInfo *SessionInfo) Context {
	return WithValue(ctx, sessionInfoContextKey, sessionInfo)
}

func createSession(ctx Context, creationTaskqueue string, options *SessionOptions, retryable bool) (Context, error) {
	logger := GetLogger(ctx)
	logger.Debug("Start creating session")
	if prevSessionInfo := getSessionInfo(ctx); prevSessionInfo != nil && prevSessionInfo.SessionState == SessionStateOpen {
		return nil, errFoundExistingOpenSession
	}
	sessionID, err := generateSessionID(ctx)
	if err != nil {
		return nil, err
	}

	taskqueueChan := GetSignalChannel(ctx, sessionID) // use sessionID as channel name
	// Retry is only needed when creating new session and the error returned is
	// NewApplicationError(errTooManySessionsMsg). Therefore we make sure to
	// disable retrying for start-to-close and heartbeat timeouts which can occur
	// when attempting to retry a create-session on a different worker.
	retryPolicy := &RetryPolicy{
		InitialInterval:        time.Second,
		BackoffCoefficient:     1.1,
		MaximumInterval:        time.Second * 10,
		MaximumAttempts:        0,
		NonRetryableErrorTypes: []string{"TemporalTimeout:StartToClose", "TemporalTimeout:Heartbeat"},
	}

	heartbeatTimeout := defaultSessionHeartbeatTimeout
	if options.HeartbeatTimeout != 0 {
		heartbeatTimeout = options.HeartbeatTimeout
	}
	ao := ActivityOptions{
		TaskQueue:              creationTaskqueue,
		ScheduleToStartTimeout: options.CreationTimeout,
		StartToCloseTimeout:    options.ExecutionTimeout,
		HeartbeatTimeout:       heartbeatTimeout,
	}
	if retryable {
		ao.RetryPolicy = retryPolicy
	}

	sessionInfo := &SessionInfo{
		SessionID:    sessionID,
		SessionState: SessionStateOpen,
	}
	completionCtx := setSessionInfo(ctx, sessionInfo)
	sessionInfo.completionCtx = completionCtx

	// create sessionCtx as a child ctx as the completionCtx for two reasons:
	//   1. completionCtx still needs the session information
	//   2. When completing session, we need to cancel both creation activity and all user activities, but
	//      we can't cancel the completionCtx.
	sessionCtx, sessionCancelFunc := WithCancel(completionCtx)
	creationCtx := WithActivityOptions(sessionCtx, ao)
	creationFuture := ExecuteActivity(creationCtx, sessionCreationActivityName, sessionID)

	var creationErr error
	var creationResponse sessionCreationResponse
	s := NewSelector(creationCtx)
	s.AddReceive(taskqueueChan, func(c ReceiveChannel, more bool) {
		c.Receive(creationCtx, &creationResponse)
	})
	s.AddFuture(creationFuture, func(f Future) {
		// activity stoped before signal is received, must be creation timeout.
		creationErr = f.Get(creationCtx, nil)
		GetLogger(creationCtx).Debug("Failed to create session", "sessionID", sessionID, tagError, creationErr)
	})
	s.Select(creationCtx)

	if creationErr != nil {
		sessionCancelFunc()
		return nil, creationErr
	}

	sessionInfo.taskqueue = creationResponse.Taskqueue
	sessionInfo.resourceID = creationResponse.ResourceID
	sessionInfo.HostName = creationResponse.HostName
	sessionInfo.sessionCancelFunc = sessionCancelFunc

	Go(creationCtx, func(creationCtx Context) {
		err := creationFuture.Get(creationCtx, nil)
		if err == nil {
			return
		}
		var canceledErr *CanceledError
		if !errors.As(err, &canceledErr) {
			getWorkflowEnvironment(creationCtx).RemoveSession(sessionID)
			GetLogger(creationCtx).Debug("Session failed", "sessionID", sessionID, tagError, err)
			sessionInfo.SessionState = SessionStateFailed
			sessionCancelFunc()
		}
	})

	logger.Debug("Created session", "sessionID", sessionID)
	getWorkflowEnvironment(ctx).AddSession(sessionInfo)
	return sessionCtx, nil
}

func generateSessionID(ctx Context) (string, error) {
	var sessionID string
	err := SideEffect(ctx, func(ctx Context) interface{} {
		return uuid.New()
	}).Get(&sessionID)
	return sessionID, err
}

func getCreationTaskqueue(base string) string {
	return base + "__internal_session_creation"
}

func getResourceSpecificTaskqueue(resourceID string) string {
	return resourceID + "@" + getHostName()
}

func sessionCreationActivity(ctx context.Context, sessionID string) error {
	sessionEnv, ok := ctx.Value(sessionEnvironmentContextKey).(sessionEnvironment)
	if !ok {
		panic("no session environment in context")
	}

	doneCh, err := sessionEnv.CreateSession(ctx, sessionID)
	if err != nil {
		return err
	}

	defer sessionEnv.AddSessionToken()

	if err := sessionEnv.SignalCreationResponse(ctx, sessionID); err != nil {
		return err
	}

	activityEnv := getActivityEnv(ctx)
	heartbeatInterval := activityEnv.heartbeatTimeout / 3
	if heartbeatInterval > maxSessionHeartbeatInterval {
		heartbeatInterval = maxSessionHeartbeatInterval
	}
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	heartbeatRetryPolicy := backoff.NewExponentialRetryPolicy(time.Second)
	heartbeatRetryPolicy.SetMaximumInterval(time.Second * 2)
	heartbeatRetryPolicy.SetExpirationInterval(heartbeatInterval)

	for {
		select {
		case <-ctx.Done():
			sessionEnv.CompleteSession(sessionID)
			return ctx.Err()
		case <-ticker.C:
			heartbeatOp := func() error {
				// here we skip the internal heartbeat batching, as otherwise the activity has only once chance
				// for heartbeating and if that failed, the entire session will get fail due to heartbeat timeout.
				// since the heartbeat interval is controlled by the session framework, we don't need to worry about
				// calling heartbeat too frequently and causing trouble for the sever. (note the min heartbeat timeout
				// is 1 sec.)
				return activityEnv.serviceInvoker.Heartbeat(ctx, nil, true)
			}
			isRetryable := func(_ error) bool {
				// there will be two types of error here:
				// 1. transient errors like timeout, in which case we should not fail the session
				// 2. non-retryable errors like activity canceled, activity not found or domain
				// not active. In those cases, the internal implementation will cancel the context,
				// so in the next iteration, ctx.Done() will be selected. Here we rely on the heartbeat
				// internal implementation to tell which error is non-retryable.
				select {
				case <-ctx.Done():
					return false
				default:
					return true
				}
			}
			// TODO refactor using grpc-retry, add support for custom handling for error codes.
			err := backoff.Retry(ctx, heartbeatOp, heartbeatRetryPolicy, isRetryable)
			if err != nil {
				GetActivityLogger(ctx).Info("session heartbeat failed", tagError, err, "sessionID", sessionID)
			}
		case <-doneCh:
			return nil
		}
	}
}

func sessionCompletionActivity(ctx context.Context, sessionID string) error {
	sessionEnv, ok := ctx.Value(sessionEnvironmentContextKey).(sessionEnvironment)
	if !ok {
		panic("no session environment in context")
	}
	sessionEnv.CompleteSession(sessionID)
	return nil
}

func isSessionCreationActivity(activity interface{}) bool {
	activityName, ok := activity.(string)
	return ok && activityName == sessionCreationActivityName
}

func mustSerializeRecreateToken(params *recreateSessionParams) []byte {
	token, err := json.Marshal(params)
	if err != nil {
		panic(err)
	}
	return token
}

func deserializeRecreateToken(token []byte) (*recreateSessionParams, error) {
	var recreateParams recreateSessionParams
	err := json.Unmarshal(token, &recreateParams)
	return &recreateParams, err
}

func newSessionTokenBucket(concurrentSessionExecutionSize int) *sessionTokenBucket {
	return &sessionTokenBucket{
		Cond:           sync.NewCond(&sync.Mutex{}),
		availableToken: concurrentSessionExecutionSize,
	}
}

func (t *sessionTokenBucket) waitForAvailableToken() {
	t.L.Lock()
	defer t.L.Unlock()
	for t.availableToken == 0 {
		t.Wait()
	}
}

func (t *sessionTokenBucket) addToken() {
	t.L.Lock()
	t.availableToken++
	t.L.Unlock()
	t.Signal()
}

func (t *sessionTokenBucket) getToken() bool {
	t.L.Lock()
	defer t.L.Unlock()
	if t.availableToken == 0 {
		return false
	}
	t.availableToken--
	return true
}

func newSessionEnvironment(resourceID string, concurrentSessionExecutionSize int) sessionEnvironment {
	return &sessionEnvironmentImpl{
		Mutex:                     &sync.Mutex{},
		doneChanMap:               make(map[string]chan struct{}),
		resourceID:                resourceID,
		resourceSpecificTaskqueue: getResourceSpecificTaskqueue(resourceID),
		sessionTokenBucket:        newSessionTokenBucket(concurrentSessionExecutionSize),
	}
}

func (env *sessionEnvironmentImpl) CreateSession(_ context.Context, sessionID string) (<-chan struct{}, error) {
	if !env.sessionTokenBucket.getToken() {
		// This error must be retryable so sessions can keep trying to be created
		return nil, NewApplicationError(errTooManySessionsMsg, "", false, nil)
	}

	env.Lock()
	defer env.Unlock()
	doneCh := make(chan struct{})
	env.doneChanMap[sessionID] = doneCh
	return doneCh, nil
}

func (env *sessionEnvironmentImpl) AddSessionToken() {
	env.sessionTokenBucket.addToken()
}

func (env *sessionEnvironmentImpl) SignalCreationResponse(ctx context.Context, sessionID string) error {
	activityEnv := getActivityEnv(ctx)
	client := activityEnv.serviceInvoker.GetClient(ClientOptions{Namespace: activityEnv.workflowNamespace})
	return client.SignalWorkflow(ctx, activityEnv.workflowExecution.ID, activityEnv.workflowExecution.RunID,
		sessionID, env.getCreationResponse())
}

func (env *sessionEnvironmentImpl) getCreationResponse() *sessionCreationResponse {
	return &sessionCreationResponse{
		Taskqueue:  env.resourceSpecificTaskqueue,
		ResourceID: env.resourceID,
		HostName:   getHostName(),
	}
}

func (env *sessionEnvironmentImpl) CompleteSession(sessionID string) {
	env.Lock()
	defer env.Unlock()

	if doneChan, ok := env.doneChanMap[sessionID]; ok {
		delete(env.doneChanMap, sessionID)
		close(doneChan)
	}
}

func (env *sessionEnvironmentImpl) GetResourceSpecificTaskqueue() string {
	return env.resourceSpecificTaskqueue
}

func (env *sessionEnvironmentImpl) GetTokenBucket() *sessionTokenBucket {
	return env.sessionTokenBucket
}

// The following two implemention is for testsuite only. The only difference is that
// the creation activity is not long running, otherwise it will block timers from auto firing.
func sessionCreationActivityForTest(ctx context.Context, sessionID string) error {
	sessionEnv := ctx.Value(sessionEnvironmentContextKey).(sessionEnvironment)

	if _, err := sessionEnv.CreateSession(ctx, sessionID); err != nil {
		return err
	}

	return sessionEnv.SignalCreationResponse(ctx, sessionID)
}

func sessionCompletionActivityForTest(ctx context.Context, sessionID string) error {
	sessionEnv := ctx.Value(sessionEnvironmentContextKey).(sessionEnvironment)

	sessionEnv.CompleteSession(sessionID)

	// Add session token in the completion activity.
	sessionEnv.AddSessionToken()
	return nil
}
