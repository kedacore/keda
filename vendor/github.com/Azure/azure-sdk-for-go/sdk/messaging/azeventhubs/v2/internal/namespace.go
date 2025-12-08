// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package internal

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"runtime"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/internal/log"
	"github.com/Azure/azure-sdk-for-go/sdk/internal/telemetry"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs/v2/internal/amqpwrap"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs/v2/internal/auth"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs/v2/internal/exported"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs/v2/internal/sbauth"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs/v2/internal/utils"
	"github.com/Azure/go-amqp"
)

var rootUserAgent = telemetry.Format("azeventhubs", Version)

type (
	// Namespace is an abstraction over an amqp.Client, allowing us to hold onto a single
	// instance of a connection per client..
	Namespace struct {
		// NOTE: values need to be 64-bit aligned. Simplest way to make sure this happens
		// is just to make it the first value in the struct
		// See:
		//   Godoc: https://pkg.go.dev/sync/atomic#pkg-note-BUG
		//   PR: https://github.com/Azure/azure-sdk-for-go/pull/16847
		connID uint64

		FQDN          string
		TokenProvider *sbauth.TokenProvider
		tlsConfig     *tls.Config
		userAgent     string

		newWebSocketConn func(ctx context.Context, args exported.WebSocketConnParams) (net.Conn, error)

		// NOTE: exported only so it can be checked in a test
		RetryOptions exported.RetryOptions

		clientMu         sync.RWMutex
		client           amqpwrap.AMQPClient
		negotiateClaimMu sync.Mutex
		// indicates that the client was closed permanently, and not just
		// for recovery.
		closedPermanently bool

		// newClientFn exists so we can stub out newClient for unit tests.
		newClientFn func(ctx context.Context, connID uint64) (amqpwrap.AMQPClient, error)

		customEndpoint string
	}

	// NamespaceOption provides structure for configuring a new Event Hub namespace
	NamespaceOption func(h *Namespace) error
)

// NamespaceWithNewAMQPLinks is the Namespace surface for consumers of AMQPLinks.
type NamespaceWithNewAMQPLinks interface {
	Check() error
}

// NamespaceForAMQPLinks is the Namespace surface needed for the internals of AMQPLinks.
type NamespaceForAMQPLinks interface {
	NegotiateClaim(ctx context.Context, entityPath string) (context.CancelFunc, <-chan struct{}, error)
	NewAMQPSession(ctx context.Context) (amqpwrap.AMQPSession, uint64, error)
	NewRPCLink(ctx context.Context, managementPath string) (amqpwrap.RPCLink, uint64, error)
	GetEntityAudience(entityPath string) string

	// Recover destroys the currently held AMQP connection and recreates it, if needed.
	//
	// NOTE: cancelling the context only cancels the initialization of a new AMQP
	// connection - the previous connection is always closed.
	Recover(ctx context.Context, clientRevision uint64) error

	Close(ctx context.Context, permanently bool) error
}

// NamespaceWithConnectionString configures a namespace with the information provided in a Event Hub connection string
func NamespaceWithConnectionString(connStr string) NamespaceOption {
	return func(ns *Namespace) error {
		props, err := exported.ParseConnectionString(connStr)
		if err != nil {
			return err
		}

		ns.FQDN = props.FullyQualifiedNamespace

		provider, err := sbauth.NewTokenProviderWithConnectionString(props)
		if err != nil {
			return err
		}

		ns.TokenProvider = provider
		return nil
	}
}

// NamespaceWithCustomEndpoint sets a custom endpoint, useful for when you're connecting through a TCP proxy.
// When establishing a TCP connection we connect to this address. The audience is extracted from the
// fullyQualifiedNamespace given to NamespaceWithTokenCredential or the endpoint in the connection string passed
// to NamespaceWithConnectionString.
func NamespaceWithCustomEndpoint(customEndpoint string) NamespaceOption {
	return func(ns *Namespace) error {
		ns.customEndpoint = customEndpoint
		return nil
	}
}

// NamespaceWithTLSConfig appends to the TLS config.
func NamespaceWithTLSConfig(tlsConfig *tls.Config) NamespaceOption {
	return func(ns *Namespace) error {
		ns.tlsConfig = tlsConfig
		return nil
	}
}

// NamespaceWithUserAgent appends to the root user-agent value.
func NamespaceWithUserAgent(userAgent string) NamespaceOption {
	return func(ns *Namespace) error {
		ns.userAgent = userAgent
		return nil
	}
}

// NamespaceWithWebSocket configures the namespace and all entities to use wss:// rather than amqps://
func NamespaceWithWebSocket(newWebSocketConn func(ctx context.Context, args exported.WebSocketConnParams) (net.Conn, error)) NamespaceOption {
	return func(ns *Namespace) error {
		ns.newWebSocketConn = newWebSocketConn
		return nil
	}
}

// NamespaceWithTokenCredential sets the token provider on the namespace
// fullyQualifiedNamespace is the Event Hub namespace name (ex: myservicebus.servicebus.windows.net)
func NamespaceWithTokenCredential(fullyQualifiedNamespace string, tokenCredential azcore.TokenCredential) NamespaceOption {
	return func(ns *Namespace) error {
		ns.TokenProvider = sbauth.NewTokenProvider(tokenCredential)
		ns.FQDN = fullyQualifiedNamespace
		return nil
	}
}

func NamespaceWithRetryOptions(retryOptions exported.RetryOptions) NamespaceOption {
	return func(ns *Namespace) error {
		ns.RetryOptions = retryOptions
		return nil
	}
}

// NewNamespace creates a new namespace configured through NamespaceOption(s)
func NewNamespace(opts ...NamespaceOption) (*Namespace, error) {
	ns := &Namespace{}

	ns.newClientFn = ns.newClientImpl

	for _, opt := range opts {
		err := opt(ns)
		if err != nil {
			return nil, err
		}
	}

	return ns, nil
}

func (ns *Namespace) newClientImpl(ctx context.Context, connID uint64) (amqpwrap.AMQPClient, error) {
	connOptions := amqp.ConnOptions{
		SASLType:    amqp.SASLTypeAnonymous(),
		MaxSessions: 65535,
		Properties: map[string]any{
			"product":    "MSGolangClient",
			"version":    Version,
			"platform":   runtime.GOOS,
			"framework":  runtime.Version(),
			"user-agent": ns.getUserAgent(),
		},
		HostName: ns.FQDN,
	}

	if ns.tlsConfig != nil {
		connOptions.TLSConfig = ns.tlsConfig
	}

	if ns.newWebSocketConn != nil {
		nConn, err := ns.newWebSocketConn(ctx, exported.WebSocketConnParams{
			Host: ns.getWSSHostURI() + "$servicebus/websocket",
		})

		if err != nil {
			return nil, err
		}

		connOptions.HostName = ns.FQDN
		client, err := amqp.NewConn(ctx, nConn, &connOptions)
		return &amqpwrap.AMQPClientWrapper{Inner: client, ConnID: connID}, err
	}

	client, err := amqp.Dial(ctx, ns.getAMQPHostURI(true), &connOptions)
	return &amqpwrap.AMQPClientWrapper{Inner: client, ConnID: connID}, err
}

// NewAMQPSession creates a new AMQP session with the internally cached *amqp.Client.
// Returns a closeable AMQP session and the current client revision.
func (ns *Namespace) NewAMQPSession(ctx context.Context) (amqpwrap.AMQPSession, uint64, error) {
	client, clientRevision, err := ns.GetAMQPClientImpl(ctx)

	if err != nil {
		return nil, 0, err
	}

	session, err := client.NewSession(ctx, nil)

	if err != nil {
		return nil, 0, err
	}

	return session, clientRevision, err
}

// Close closes the current cached client.
func (ns *Namespace) Close(ctx context.Context, permanently bool) error {
	ns.clientMu.Lock()
	defer ns.clientMu.Unlock()

	if permanently {
		ns.closedPermanently = true
	}

	if ns.client != nil {
		err := ns.client.Close()
		ns.client = nil

		if err != nil {
			log.Writef(exported.EventConn, "Failed when closing AMQP connection: %s", err)
		}
	}

	return nil
}

// Check returns an error if the namespace cannot be used (ie, closed permanently), or nil otherwise.
func (ns *Namespace) Check() error {
	ns.clientMu.RLock()
	defer ns.clientMu.RUnlock()

	if ns.closedPermanently {
		return ErrClientClosed
	}

	return nil
}

var ErrClientClosed = NewErrNonRetriable("client has been closed by user")

// Recover destroys the currently held AMQP connection and recreates it, if needed.
//
// NOTE: cancelling the context only cancels the initialization of a new AMQP
// connection - the previous connection is always closed.
func (ns *Namespace) Recover(ctx context.Context, theirConnID uint64) error {
	if err := ns.Check(); err != nil {
		return err
	}

	ns.clientMu.Lock()
	defer ns.clientMu.Unlock()

	if ns.closedPermanently {
		return ErrClientClosed
	}

	if ns.connID != theirConnID {
		log.Writef(exported.EventConn, "Skipping connection recovery, already recovered: %d vs %d. Links will still be recovered.", ns.connID, theirConnID)
		return nil
	}

	if ns.client != nil {
		oldClient := ns.client
		ns.client = nil

		if err := oldClient.Close(); err != nil {
			// the error on close isn't critical, we don't need to exit or
			// return it.
			log.Writef(exported.EventConn, "Error closing old client: %s", err.Error())
		}
	}

	log.Writef(exported.EventConn, "Creating a new client (rev:%d)", ns.connID)

	if _, _, err := ns.updateClientWithoutLock(ctx); err != nil {
		return err
	}

	return nil
}

// negotiateClaimFn matches the signature for NegotiateClaim, and is used when we want to stub things out for tests.
type negotiateClaimFn func(
	ctx context.Context, audience string, conn amqpwrap.AMQPClient, provider auth.TokenProvider) error

// negotiateClaim performs initial authentication and starts periodic refresh of credentials.
// the returned func is to cancel() the refresh goroutine.
func (ns *Namespace) NegotiateClaim(ctx context.Context, entityPath string) (context.CancelFunc, <-chan struct{}, error) {
	return ns.startNegotiateClaimRenewer(ctx,
		entityPath,
		NegotiateClaim,
		nextClaimRefreshDuration)
}

// startNegotiateClaimRenewer does an initial claim request and then starts a goroutine that
// continues to automatically refresh in the background.
// Returns a func() that can be used to cancel the background renewal, a channel that will be closed
// when the background renewal stops or an error.
func (ns *Namespace) startNegotiateClaimRenewer(ctx context.Context,
	entityPath string,
	cbsNegotiateClaim negotiateClaimFn,
	nextClaimRefreshDurationFn func(expirationTime time.Time, currentTime time.Time) time.Duration) (func(), <-chan struct{}, error) {
	audience := ns.GetEntityAudience(entityPath)

	refreshClaim := func(ctx context.Context) (time.Time, error) {
		log.Writef(exported.EventAuth, "(%s) refreshing claim", entityPath)

		amqpClient, clientRevision, err := ns.GetAMQPClientImpl(ctx)

		if err != nil {
			return time.Time{}, err
		}

		token, expiration, err := ns.TokenProvider.GetTokenAsTokenProvider(audience)

		if err != nil {
			log.Writef(exported.EventAuth, "(%s) negotiate claim, failed getting token: %s", entityPath, err.Error())
			return time.Time{}, err
		}

		log.Writef(exported.EventAuth, "(%s) negotiate claim, token expires on %s", entityPath, expiration.Format(time.RFC3339))

		// You're not allowed to have multiple $cbs links open in a single connection.
		// The current cbs.NegotiateClaim implementation automatically creates and shuts
		// down it's own link so we have to guard against that here.
		ns.negotiateClaimMu.Lock()
		err = cbsNegotiateClaim(ctx, audience, amqpClient, token)
		ns.negotiateClaimMu.Unlock()

		if err != nil {
			// Note we only handle connection recovery here since (currently)
			// the negotiateClaim code creates it's own link each time.
			if GetRecoveryKind(err) == RecoveryKindConn {
				if err := ns.Recover(ctx, clientRevision); err != nil {
					log.Writef(exported.EventAuth, "(%s) negotiate claim, failed in connection recovery: %s", entityPath, err)
				}
			}

			log.Writef(exported.EventAuth, "(%s) negotiate claim, failed: %s", entityPath, err.Error())
			return time.Time{}, err
		}

		return expiration, nil
	}

	expiresOn, err := refreshClaim(ctx)

	if err != nil {
		return nil, nil, err
	}

	// start the periodic refresh of credentials
	refreshCtx, cancelRefreshCtx := context.WithCancel(context.Background())
	refreshStoppedCh := make(chan struct{})

	// connection strings with embedded SAS tokens will return a zero expiration time since they can't be renewed.
	if expiresOn.IsZero() {
		log.Writef(exported.EventAuth, "Token does not have an expiration date, no background renewal needed.")

		// cancel everything related to the claims refresh loop.
		cancelRefreshCtx()
		close(refreshStoppedCh)

		return func() {}, refreshStoppedCh, nil
	}

	go func() {
		defer cancelRefreshCtx()
		defer close(refreshStoppedCh)

	TokenRefreshLoop:
		for {
			nextClaimAt := nextClaimRefreshDurationFn(expiresOn, time.Now())

			log.Writef(exported.EventAuth, "(%s) next refresh in %s", entityPath, nextClaimAt)

			select {
			case <-refreshCtx.Done():
				return
			case <-time.After(nextClaimAt):
				for {
					err := utils.Retry(refreshCtx, exported.EventAuth, func() string { return "NegotiateClaimRefresh" }, ns.RetryOptions, func(ctx context.Context, args *utils.RetryFnArgs) error {
						tmpExpiresOn, err := refreshClaim(ctx)

						if err != nil {
							return err
						}

						expiresOn = tmpExpiresOn
						return nil
					}, IsFatalEHError)

					if err == nil {
						break
					}

					if GetRecoveryKind(err) == RecoveryKindFatal {
						log.Writef(exported.EventAuth, "[%s] fatal error, stopping token refresh loop: %s", entityPath, err.Error())
						break TokenRefreshLoop
					}
				}
			}
		}
	}()

	return func() {
		cancelRefreshCtx()
		<-refreshStoppedCh
	}, refreshStoppedCh, nil
}

func (ns *Namespace) GetAMQPClientImpl(ctx context.Context) (amqpwrap.AMQPClient, uint64, error) {
	if err := ns.Check(); err != nil {
		return nil, 0, err
	}

	ns.clientMu.Lock()
	defer ns.clientMu.Unlock()

	if ns.closedPermanently {
		return nil, 0, ErrClientClosed
	}

	return ns.updateClientWithoutLock(ctx)
}

// updateClientWithoutLock takes care of initializing a client (if needed)
// and returns the initialized client and it's connection ID, or an error.
func (ns *Namespace) updateClientWithoutLock(ctx context.Context) (amqpwrap.AMQPClient, uint64, error) {
	if ns.client != nil {
		return ns.client, ns.connID, nil
	}

	connStart := time.Now()
	log.Writef(exported.EventConn, "Creating new client, current rev: %d", ns.connID)

	newConnID := ns.connID + 1
	tempClient, err := ns.newClientFn(ctx, newConnID)

	if err != nil {
		return nil, 0, err
	}

	ns.connID = newConnID
	ns.client = tempClient
	log.Writef(exported.EventConn, "Client created, new rev: %d, took %dms", ns.connID, time.Since(connStart)/time.Millisecond)

	return ns.client, ns.connID, err
}

func (ns *Namespace) getWSSHostURI() string {
	return fmt.Sprintf("wss://%s/", ns.FQDN)
}

func (ns *Namespace) getAMQPHostURI(useCustomEndpoint bool) string {
	fqdn := ns.FQDN

	if useCustomEndpoint && ns.customEndpoint != "" {
		fqdn = ns.customEndpoint
	}

	if ns.TokenProvider.InsecureDisableTLS {
		return fmt.Sprintf("amqp://%s/", fqdn)
	} else {
		return fmt.Sprintf("amqps://%s/", fqdn)
	}
}

func (ns *Namespace) GetHTTPSHostURI() string {
	return fmt.Sprintf("https://%s/", ns.FQDN)
}

func (ns *Namespace) GetEntityAudience(entityPath string) string {
	return ns.getAMQPHostURI(false) + entityPath
}

func (ns *Namespace) getUserAgent() string {
	userAgent := rootUserAgent
	if ns.userAgent != "" {
		userAgent = fmt.Sprintf("%s %s", ns.userAgent, userAgent)
	}
	return userAgent
}

// nextClaimRefreshDuration figures out the proper interval for the next authorization
// refresh.
//
// It applies a few real world adjustments:
// - We assume the expiration time is 10 minutes ahead of when it actually is, to adjust for clock drift.
// - We don't let the refresh interval fall below 2 minutes
// - We don't let the refresh interval go above 49 days
//
// This logic is from here:
// https://github.com/Azure/azure-sdk-for-net/blob/bfd3109d0f9afa763131731d78a31e39c81101b3/sdk/servicebus/Azure.Messaging.ServiceBus/src/Amqp/AmqpConnectionScope.cs#L998
func nextClaimRefreshDuration(expirationTime time.Time, currentTime time.Time) time.Duration {
	const min = 2 * time.Minute
	const max = 49 * 24 * time.Hour
	const clockDrift = 10 * time.Minute

	var refreshDuration = expirationTime.Sub(currentTime) - clockDrift

	if refreshDuration < min {
		return min
	} else if refreshDuration > max {
		return max
	}

	return refreshDuration
}
