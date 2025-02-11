// Copyright Splunk Inc.
// SPDX-License-Identifier: Apache-2.0

/*
Package signalflow contains a SignalFx SignalFlow client,
which can be used to execute analytics jobs against the SignalFx backend.

Not all SignalFlow messages are handled at this time,
and some will be silently dropped.
All of the most important and useful ones are supported though.

The client will automatically attempt to reconnect to the backend
if the connection is broken after a short delay.

SignalFlow is documented at https://dev.splunk.com/observability/docs/signalflow/messages.
*/
package signalflow
