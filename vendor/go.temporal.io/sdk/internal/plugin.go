package internal

import (
	"context"
	"fmt"

	"github.com/nexus-rpc/sdk-go/nexus"
	historypb "go.temporal.io/api/history/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/log"
)

// ClientPlugin is a plugin that can configure client options and surround client
// creation/connection. Many plugin implementers may prefer the simpler
// [go.temporal.io/sdk/temporal.SimplePlugin] instead.
//
// All client plugins must embed [go.temporal.io/sdk/client.PluginBase]. All
// plugins must implement Name().
//
// All client plugins that also implement [go.temporal.io/sdk/worker.Plugin] are
// automatically configured on workers made from the client.
//
// Exposed as: [go.temporal.io/sdk/client.Plugin]
//
// NOTE: Experimental
type ClientPlugin interface {
	// Name returns the name for this plugin.
	Name() string

	// ConfigureClient is called when a client is created but before the options
	// are validated. This call gives plugins a chance to adjust options as
	// needed. This often includes adding interceptors.
	ConfigureClient(context.Context, ClientPluginConfigureClientOptions) error

	// NewClient is called when a client is being created/connected after
	// options have been set. This is meant to surround dial calls. Implementers
	// must either return an error or call next.
	//
	// This method intentionally does not allow control over the actual client
	// instance because only the explicit client instance can be used in the SDK.
	NewClient(
		ctx context.Context,
		options ClientPluginNewClientOptions,
		next func(context.Context, ClientPluginNewClientOptions) error,
	) error

	// Plugins must embed [go.temporal.io/sdk/client.PluginBase].
	mustEmbedClientPluginBase()
}

// ClientPluginConfigureClientOptions are options for ConfigureClient on a
// client plugin.
//
// Exposed as: [go.temporal.io/sdk/client.PluginConfigureClientOptions]
//
// NOTE: Experimental
type ClientPluginConfigureClientOptions struct {
	// ClientOptions are the set of mutable options that can be adjusted by
	// plugins.
	ClientOptions *ClientOptions
}

// ClientPluginNewClientOptions are options for NewClient on a client plugin.
//
// Exposed as: [go.temporal.io/sdk/client.PluginNewClientOptions]
//
// NOTE: Experimental
type ClientPluginNewClientOptions struct {
	// ClientOptions are the set of options used for the client. These should
	// not be mutated, that should be done via the ConfigureClient method.
	ClientOptions ClientOptions

	// Lazy is whether the new client call is being invoked lazily or not.
	Lazy bool

	// FromExisting is set to a non-nil Client if this client is being created
	// from an existing client.
	FromExisting Client
}

// ClientPluginBase must be embedded into client plugin implementations.
//
// Exposed as: [go.temporal.io/sdk/client.PluginBase]
//
// NOTE: Experimental
type ClientPluginBase struct{}

var _ ClientPlugin = struct {
	pluginNamePanicForTypeChecking
	ClientPluginBase
}{}

// WorkerPlugin is a plugin that can configure worker/replayer options and
// surround worker/replayer runs. Many plugin implementers may prefer the
// simpler [go.temporal.io/sdk/temporal.SimplePlugin] instead.
//
// All worker plugins must embed [go.temporal.io/sdk/worker.PluginBase]. All
// plugins must implement Name().
//
// Exposed as: [go.temporal.io/sdk/worker.Plugin]
//
// NOTE: Experimental
type WorkerPlugin interface {
	// Name returns the name for this plugin.
	Name() string

	// ConfigureWorker is called when a worker is created but before the options
	// are validated. This call gives plugins a chance to adjust options as
	// needed. This often includes adding interceptors.
	//
	// Note, at this time, due to [go.temporal.io/worker.New] not returning an
	// error, any errors returned from here become panics. Also, at this time,
	// the context cannot be supplied by a user, so it's effectively
	// meaningless.
	ConfigureWorker(context.Context, WorkerPluginConfigureWorkerOptions) error

	// StartWorker is called to start a worker. This is called on Worker.Start
	// or Worker.Run. Implementers should return an error or invoke next.
	StartWorker(
		ctx context.Context,
		options WorkerPluginStartWorkerOptions,
		next func(context.Context, WorkerPluginStartWorkerOptions) error,
	) error

	// StopWorker is called to stop a worker. This is called on Worker.Stop or
	// if Worker.Run is interrupted via its interrupt channel. However, if a
	// fatal worker error occurs during Worker.Run, this may not be called.
	// Implementers can account for this situation by setting OnFatalError in
	// the worker options. Implementers should invoke next.
	StopWorker(
		ctx context.Context,
		options WorkerPluginStopWorkerOptions,
		next func(context.Context, WorkerPluginStopWorkerOptions),
	)

	// ConfigureWorkflowReplayer is called when the workflow replayer is created
	// but before the options are validated. This call gives plugins a chance to
	// adjust options as needed. This often includes adding interceptors.
	ConfigureWorkflowReplayer(context.Context, WorkerPluginConfigureWorkflowReplayerOptions) error

	// ReplayWorkflow is called for each individual workflow replay on the
	// replayer. Implementers should return an error or invoke next.
	ReplayWorkflow(
		ctx context.Context,
		options WorkerPluginReplayWorkflowOptions,
		next func(context.Context, WorkerPluginReplayWorkflowOptions) error,
	) error

	// Plugins must embed [go.temporal.io/sdk/worker.PluginBase].
	mustEmbedWorkerPluginBase()
}

// WorkerPluginBase must be embedded into worker plugin implementations.
//
// Exposed as: [go.temporal.io/sdk/worker.PluginBase]
//
// NOTE: Experimental
type WorkerPluginBase struct{}

var _ WorkerPlugin = struct {
	pluginNamePanicForTypeChecking
	WorkerPluginBase
}{}

// WorkerPluginConfigureWorkerOptions are options for ConfigureWorker on a
// worker plugin.
//
// Exposed as: [go.temporal.io/sdk/worker.PluginConfigureWorkerOptions]
//
// NOTE: Experimental
type WorkerPluginConfigureWorkerOptions struct {
	// WorkerInstanceKey is the unique, immutable instance key for this worker.
	WorkerInstanceKey string

	// TaskQueue is the immutable task queue for this worker.
	TaskQueue string

	// WorkerOptions are the set of mutable options that can be adjusted by
	// plugins.
	WorkerOptions *WorkerOptions

	// WorkerRegistryOptions are the set of callbacks that can be adjusted by
	// plugins. If adjusting a callback that is already set, implementers may
	// want to take care to invoke the existing callback inside their own.
	WorkerRegistryOptions *WorkerPluginConfigureWorkerRegistryOptions
}

// WorkerPluginConfigureWorkerRegistryOptions are the set of callbacks that can
// be adjusted by plugins when configuring workers. If adjusting a callback that
// is already set, implementers may want to take care to invoke the existing
// callback inside their own.
//
// Exposed as: [go.temporal.io/sdk/worker.PluginConfigureWorkerRegistryOptions]
//
// NOTE: Experimental
type WorkerPluginConfigureWorkerRegistryOptions struct {
	// Called when a workflow is registered. The first parameter will be the workflow.
	OnRegisterWorkflow func(any, RegisterWorkflowOptions)
	// Called when a dynamic workflow is registered. The first parameter will be the workflow.
	OnRegisterDynamicWorkflow func(any, DynamicRegisterWorkflowOptions)
	// Called when an activity is registered. The first parameter will be the activity.
	OnRegisterActivity func(any, RegisterActivityOptions)
	// Called when a dynamic activity is registered. The first parameter will be the activity.
	OnRegisterDynamicActivity func(any, DynamicRegisterActivityOptions)
	// Called when a Nexus service is registered. The first parameter will be the Nexus service.
	OnRegisterNexusService func(*nexus.Service)
}

// WorkerPluginStartWorkerOptions are options for StartWorker on a worker
// plugin.
//
// Exposed as: [go.temporal.io/sdk/worker.PluginStartWorkerOptions]
//
// NOTE: Experimental
type WorkerPluginStartWorkerOptions struct {
	// WorkerInstanceKey is the unique, immutable instance key for this worker.
	WorkerInstanceKey string

	// WorkerRegistry is the worker registry plugins can use to register items
	// with the worker. Implementers should usually not mutate this before
	// passing to "next", but instead add register callbacks in ConfigureWorker
	// if needed. Calls to this registry to invoke the OnX callbacks set in
	// ConfigureWorker.
	WorkerRegistry interface {
		RegisterWorkflowWithOptions(any, RegisterWorkflowOptions)
		RegisterDynamicWorkflow(any, DynamicRegisterWorkflowOptions)
		RegisterActivityWithOptions(any, RegisterActivityOptions)
		RegisterDynamicActivity(any, DynamicRegisterActivityOptions)
		RegisterNexusService(*nexus.Service)
	}
}

// WorkerPluginStopWorkerOptions are options for StopWorker on a worker plugin.
//
// Exposed as: [go.temporal.io/sdk/worker.PluginStopWorkerOptions]
//
// NOTE: Experimental
type WorkerPluginStopWorkerOptions struct {
	// WorkerInstanceKey is the unique, immutable instance key for this worker.
	WorkerInstanceKey string
}

// WorkerPluginConfigureWorkflowReplayerOptions are options for
// ConfigureWorkflowReplayer on a worker plugin.
//
// Exposed as: [go.temporal.io/sdk/worker.PluginConfigureWorkflowReplayerOptions]
//
// NOTE: Experimental
type WorkerPluginConfigureWorkflowReplayerOptions struct {
	// WorkflowReplayerInstanceKey is the unique, immutable instance key for
	// this workflow replayer.
	WorkflowReplayerInstanceKey string

	// WorkflowReplayerOptions are the set of mutable options that can be
	// adjusted by plugins.
	WorkflowReplayerOptions *WorkflowReplayerOptions

	// WorkflowReplayerRegistryOptions are the set of callbacks that can be
	// adjusted by plugins. If adjusting a callback that is already set,
	// implementers may want to take care to invoke the existing callback inside
	// their own.
	WorkflowReplayerRegistryOptions *WorkerPluginConfigureWorkflowReplayerRegistryOptions
}

// WorkerPluginConfigureWorkflowReplayerRegistryOptions are the set of callbacks
// that can be adjusted by plugins when configuring workflow replayers. If
// adjusting a callback that is already set, implementers may want to take care
// to invoke the existing callback inside their own.
//
// Exposed as: [go.temporal.io/sdk/worker.PluginConfigureWorkflowReplayerRegistryOptions]
//
// NOTE: Experimental
type WorkerPluginConfigureWorkflowReplayerRegistryOptions struct {
	// Called when a workflow is registered. The first parameter will be the workflow.
	OnRegisterWorkflow func(any, RegisterWorkflowOptions)
	// Called when a dynamic workflow is registered. The first parameter will be the workflow.
	OnRegisterDynamicWorkflow func(any, DynamicRegisterWorkflowOptions)
}

// WorkerPluginReplayWorkflowOptions are options for ReplayWorkflow on a worker
// plugin.
//
// Exposed as: [go.temporal.io/sdk/worker.PluginReplayWorkflowOptions]
//
// NOTE: Experimental
type WorkerPluginReplayWorkflowOptions struct {
	// WorkflowReplayerInstanceKey is the unique, immutable instance key for
	// this workflow replayer.
	WorkflowReplayerInstanceKey string

	// WorkflowReplayRegistry is the workflow replayer registry plugins can use
	// to register items with the replayer. Implementers should usually not
	// mutate this before passing to "next", but instead add register callbacks
	// in ConfigureWorkflowReplayer if needed. Calls to this registry to invoke
	// the OnX callbacks set in ConfigureWorkflowReplayer.
	WorkflowReplayRegistry interface {
		RegisterWorkflowWithOptions(any, RegisterWorkflowOptions)
		RegisterDynamicWorkflow(any, DynamicRegisterWorkflowOptions)
	}

	// History to replay.
	History *historypb.History

	// All fields below are coalesced from overloads. No guarantees are made
	// about their values.
	Logger                log.Logger
	WorkflowServiceClient workflowservice.WorkflowServiceClient
	Namespace             string
	OriginalExecution     WorkflowExecution
}

type pluginNamePanicForTypeChecking struct{}

func (pluginNamePanicForTypeChecking) Name() string { panic("unreachable") }

// SimplePlugin implements both [go.temporal.io/sdk/client.Plugin] and
// [go.temporal.io/sdk/worker.Plugin] from a given set of options. Use
// [go.temporal.io/sdk/temporal.NewSimplePlugin] to instantiate this.
//
// Exposed as: [go.temporal.io/sdk/temporal.SimplePlugin]
//
// NOTE: Experimental
type SimplePlugin struct {
	options SimplePluginOptions
}

var _ ClientPlugin = (*SimplePlugin)(nil)
var _ WorkerPlugin = (*SimplePlugin)(nil)

// SimplePluginOptions are options for NewSimplePlugin.
//
// Exposed as: [go.temporal.io/sdk/temporal.SimplePluginOptions]
//
// NOTE: Experimental
type SimplePluginOptions struct {
	// Name is the required name of the plugin.
	Name string

	// DataConverter, if set, overrides any user-set or previous-plugin-set data
	// converter on client or replayer options. Use ConfigureClient or
	// ConfigureWorkflowReplayer if needing to react to existing options instead
	// of overwriting.
	DataConverter converter.DataConverter

	// FailureConverter, if set, overrides any user-set or previous-plugin-set
	// failure converter on client or replayer options. Use ConfigureClient or
	// ConfigureWorkflowReplayer if needing to react to existing options instead
	// of overwriting.
	FailureConverter converter.FailureConverter

	// ContextPropagators are appended to any user-set or previous-plugin-set
	// context propagators on client or replayer options. Use ConfigureClient or
	// ConfigureWorkflowReplayer if needing to react to existing options instead
	// of appending.
	ContextPropagators []ContextPropagator

	// ClientInterceptors are appended to any user-set or previous-plugin-set
	// client interceptors on client options. Use ConfigureClient if needing to
	// react to existing options instead of appending.
	ClientInterceptors []ClientInterceptor

	// WorkerInterceptors are appended to any user-set or previous-plugin-set
	// worker interceptors on worker or replayer options. Use ConfigureWorker or
	// ConfigureWorkflowReplayer if needing to react to existing options instead
	// of appending.
	WorkerInterceptors []WorkerInterceptor

	// ConfigureClient if set is invoked to adjust client options. This is
	// invoked after any above options are set.
	ConfigureClient func(context.Context, ClientPluginConfigureClientOptions) error

	// ConfigureWorker if set is invoked to adjust worker options. This is
	// invoked after any above options are set.
	ConfigureWorker func(context.Context, WorkerPluginConfigureWorkerOptions) error

	// ConfigureWorkflowReplayer if set is invoked to adjust workflow replayer
	// options. This is invoked after any above options are set.
	ConfigureWorkflowReplayer func(context.Context, WorkerPluginConfigureWorkflowReplayerOptions) error

	// RunContextBefore is invoked on worker start or before each workflow
	// replay of a replayer. Implementers can use this to register items or
	// simply start something needed.
	RunContextBefore func(context.Context, SimplePluginRunContextBeforeOptions) error

	// RunContextAfter is invoked on worker stop or after each workflow replay
	// of a replayer. Implementers can use this to close something started
	// before.
	//
	// See the note on [WorkerPlugin.StopWorker] about rare situations in which
	// this may not run on worker completion.
	RunContextAfter func(context.Context, SimplePluginRunContextAfterOptions)
}

// SimplePluginRunContextBeforeOptions are options for RunContextBefore on a
// simple plugin.
//
// Exposed as: [go.temporal.io/sdk/temporal.SimplePluginRunContextBeforeOptions]
//
// NOTE: Experimental
type SimplePluginRunContextBeforeOptions struct {
	// InstanceKey is the unique, immutable instance key for the worker or
	// workflow replayer.
	InstanceKey string

	// WorkflowReplayer is true if this is a workflow replayer, or false if it
	// is a worker.
	WorkflowReplayer bool

	// Registry is the worker/replayer registry plugins can use to register
	// items. Note, activity and Nexus service registration do nothing if
	// WorkflowReplayer is true.
	Registry interface {
		RegisterWorkflowWithOptions(any, RegisterWorkflowOptions)
		RegisterDynamicWorkflow(any, DynamicRegisterWorkflowOptions)
		RegisterActivityWithOptions(any, RegisterActivityOptions)
		RegisterDynamicActivity(any, DynamicRegisterActivityOptions)
		RegisterNexusService(*nexus.Service)
	}
}

// SimplePluginRunContextAfterOptions are options for RunContextAfter on a
// simple plugin.
//
// Exposed as: [go.temporal.io/sdk/temporal.SimplePluginRunContextAfterOptions]
//
// NOTE: Experimental
type SimplePluginRunContextAfterOptions struct {
	// InstanceKey is the unique, immutable instance key for the worker or
	// workflow replayer.
	InstanceKey string
}

func (ClientPluginBase) ConfigureClient(context.Context, ClientPluginConfigureClientOptions) error {
	return nil
}

func (ClientPluginBase) NewClient(
	ctx context.Context,
	options ClientPluginNewClientOptions,
	next func(context.Context, ClientPluginNewClientOptions) error,
) error {
	return next(ctx, options)
}

//lint:ignore U1000 Intentionally unused
func (ClientPluginBase) mustEmbedClientPluginBase() {}

func (WorkerPluginBase) ConfigureWorker(context.Context, WorkerPluginConfigureWorkerOptions) error {
	return nil
}

func (WorkerPluginBase) StartWorker(
	ctx context.Context,
	options WorkerPluginStartWorkerOptions,
	next func(context.Context, WorkerPluginStartWorkerOptions) error,
) error {
	return next(ctx, options)
}

func (WorkerPluginBase) StopWorker(
	ctx context.Context,
	options WorkerPluginStopWorkerOptions,
	next func(context.Context, WorkerPluginStopWorkerOptions),
) {
	next(ctx, options)
}

func (WorkerPluginBase) ConfigureWorkflowReplayer(context.Context, WorkerPluginConfigureWorkflowReplayerOptions) error {
	return nil
}

func (WorkerPluginBase) ReplayWorkflow(
	ctx context.Context,
	options WorkerPluginReplayWorkflowOptions,
	next func(context.Context, WorkerPluginReplayWorkflowOptions) error,
) error {
	return next(ctx, options)
}

//lint:ignore U1000 Intentionally unused
func (WorkerPluginBase) mustEmbedWorkerPluginBase() {}

// NewSimplePlugin creates a new SimplePlugin with the given options.
func NewSimplePlugin(options SimplePluginOptions) (*SimplePlugin, error) {
	if options.Name == "" {
		return nil, fmt.Errorf("name required")
	}
	return &SimplePlugin{options}, nil
}

// We impl these instead of embedding plugin base to force SimplePlugin to
// explicitly account for new plugin things added
func (*SimplePlugin) mustEmbedClientPluginBase() {}
func (*SimplePlugin) mustEmbedWorkerPluginBase() {}

func (s *SimplePlugin) Name() string { return s.options.Name }

func (s *SimplePlugin) ConfigureClient(ctx context.Context, options ClientPluginConfigureClientOptions) error {
	if s.options.DataConverter != nil {
		options.ClientOptions.DataConverter = s.options.DataConverter
	}
	if s.options.FailureConverter != nil {
		options.ClientOptions.FailureConverter = s.options.FailureConverter
	}
	if len(s.options.ContextPropagators) > 0 {
		options.ClientOptions.ContextPropagators =
			append(options.ClientOptions.ContextPropagators, s.options.ContextPropagators...)
	}
	if len(s.options.ClientInterceptors) > 0 {
		options.ClientOptions.Interceptors = append(options.ClientOptions.Interceptors, s.options.ClientInterceptors...)
	}
	if s.options.ConfigureClient != nil {
		if err := s.options.ConfigureClient(ctx, options); err != nil {
			return err
		}
	}
	return nil
}

func (*SimplePlugin) NewClient(
	ctx context.Context,
	options ClientPluginNewClientOptions,
	next func(context.Context, ClientPluginNewClientOptions) error,
) error {
	return next(ctx, options)
}

func (s *SimplePlugin) ConfigureWorker(ctx context.Context, options WorkerPluginConfigureWorkerOptions) error {
	if len(s.options.WorkerInterceptors) > 0 {
		options.WorkerOptions.Interceptors = append(options.WorkerOptions.Interceptors, s.options.WorkerInterceptors...)
	}
	if s.options.ConfigureWorker != nil {
		if err := s.options.ConfigureWorker(ctx, options); err != nil {
			return err
		}
	}
	return nil
}

func (s *SimplePlugin) StartWorker(
	ctx context.Context,
	options WorkerPluginStartWorkerOptions,
	next func(context.Context, WorkerPluginStartWorkerOptions) error,
) error {
	if s.options.RunContextBefore != nil {
		if err := s.options.RunContextBefore(
			ctx,
			SimplePluginRunContextBeforeOptions{
				InstanceKey: options.WorkerInstanceKey,
				Registry:    options.WorkerRegistry,
			},
		); err != nil {
			return err
		}
	}
	return next(ctx, options)
}

func (s *SimplePlugin) StopWorker(
	ctx context.Context,
	options WorkerPluginStopWorkerOptions,
	next func(context.Context, WorkerPluginStopWorkerOptions),
) {
	if s.options.RunContextAfter != nil {
		s.options.RunContextAfter(
			ctx,
			SimplePluginRunContextAfterOptions{InstanceKey: options.WorkerInstanceKey},
		)
	}
	next(ctx, options)
}

func (s *SimplePlugin) ConfigureWorkflowReplayer(
	ctx context.Context,
	options WorkerPluginConfigureWorkflowReplayerOptions,
) error {
	if s.options.DataConverter != nil {
		options.WorkflowReplayerOptions.DataConverter = s.options.DataConverter
	}
	if s.options.FailureConverter != nil {
		options.WorkflowReplayerOptions.FailureConverter = s.options.FailureConverter
	}
	if len(s.options.ContextPropagators) > 0 {
		options.WorkflowReplayerOptions.ContextPropagators = append(
			options.WorkflowReplayerOptions.ContextPropagators,
			s.options.ContextPropagators...,
		)
	}
	// Go over every client interceptor and append if it's also a worker interceptor
	for _, interceptor := range s.options.ClientInterceptors {
		if workerInterceptor, _ := interceptor.(WorkerInterceptor); workerInterceptor != nil {
			options.WorkflowReplayerOptions.Interceptors = append(
				options.WorkflowReplayerOptions.Interceptors,
				workerInterceptor,
			)
		}
	}
	if len(s.options.WorkerInterceptors) > 0 {
		options.WorkflowReplayerOptions.Interceptors = append(
			options.WorkflowReplayerOptions.Interceptors,
			s.options.WorkerInterceptors...,
		)
	}
	if s.options.ConfigureWorkflowReplayer != nil {
		if err := s.options.ConfigureWorkflowReplayer(ctx, options); err != nil {
			return err
		}
	}
	return nil
}

type simplePluginWorkflowReplayerRegistry struct {
	registerWorkflowWithOptions func(any, RegisterWorkflowOptions)
	registerDynamicWorkflow     func(any, DynamicRegisterWorkflowOptions)
}

func (s simplePluginWorkflowReplayerRegistry) RegisterWorkflowWithOptions(w any, options RegisterWorkflowOptions) {
	s.registerWorkflowWithOptions(w, options)
}

func (s simplePluginWorkflowReplayerRegistry) RegisterDynamicWorkflow(w any, options DynamicRegisterWorkflowOptions) {
	s.registerDynamicWorkflow(w, options)
}

func (simplePluginWorkflowReplayerRegistry) RegisterActivityWithOptions(any, RegisterActivityOptions) {
	// No-op
}

func (simplePluginWorkflowReplayerRegistry) RegisterDynamicActivity(any, DynamicRegisterActivityOptions) {
	// No-op
}

func (simplePluginWorkflowReplayerRegistry) RegisterNexusService(*nexus.Service) {
	// No-op
}

func (s *SimplePlugin) ReplayWorkflow(
	ctx context.Context,
	options WorkerPluginReplayWorkflowOptions,
	next func(context.Context, WorkerPluginReplayWorkflowOptions) error,
) error {
	if s.options.RunContextBefore != nil {
		if err := s.options.RunContextBefore(
			ctx,
			SimplePluginRunContextBeforeOptions{
				InstanceKey:      options.WorkflowReplayerInstanceKey,
				WorkflowReplayer: true,
				Registry: simplePluginWorkflowReplayerRegistry{
					registerWorkflowWithOptions: options.WorkflowReplayRegistry.RegisterWorkflowWithOptions,
					registerDynamicWorkflow:     options.WorkflowReplayRegistry.RegisterDynamicWorkflow,
				},
			},
		); err != nil {
			return err
		}
	}
	if s.options.RunContextAfter != nil {
		defer s.options.RunContextAfter(
			ctx,
			SimplePluginRunContextAfterOptions{
				InstanceKey: options.WorkflowReplayerInstanceKey,
			},
		)
	}
	return next(ctx, options)
}
