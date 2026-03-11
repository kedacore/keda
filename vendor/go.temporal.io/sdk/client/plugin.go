package client

import "go.temporal.io/sdk/internal"

// Plugin is a plugin that can configure client options and surround client
// creation/connection. Many plugin implementers may prefer the simpler
// [go.temporal.io/sdk/temporal.SimplePlugin] instead.
//
// All client plugins must embed [go.temporal.io/sdk/client.PluginBase]. All
// plugins must implement Name().
//
// All client plugins that also implement [go.temporal.io/sdk/worker.Plugin] are
// automatically configured on workers made from the client.
//
// NOTE: Experimental
type Plugin = internal.ClientPlugin

// PluginBase must be embedded into client plugin implementations.
//
// NOTE: Experimental
type PluginBase = internal.ClientPluginBase

// PluginConfigureClientOptions are options for ConfigureClient on a
// client plugin.
//
// NOTE: Experimental
type PluginConfigureClientOptions = internal.ClientPluginConfigureClientOptions

// PluginNewClientOptions are options for NewClient on a client plugin.
//
// NOTE: Experimental
type PluginNewClientOptions = internal.ClientPluginNewClientOptions
