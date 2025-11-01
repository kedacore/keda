package internal

// Below are the metadata which will be embedded as part of headers in every RPC call made by this client to Temporal server.
// Update to the metadata below is typically done by the Temporal team as part of a major feature or behavior change.

const (
	// SDKVersion is a semver (https://semver.org/) that represents the version of this Temporal GoSDK.
	// Server validates if SDKVersion fits its supported range and rejects request if it doesn't.
	//
	// Exposed as: [go.temporal.io/sdk/temporal.SDKVersion]
	SDKVersion = "1.37.0"

	// SDKName represents the name of the SDK.
	SDKName = clientNameHeaderValue

	// SupportedServerVersions is a semver rages (https://github.com/blang/semver#ranges) of server versions that
	// are supported by this Temporal SDK.
	// Server validates if its version fits into SupportedServerVersions range and rejects request if it doesn't.
	SupportedServerVersions = ">=1.0.0 <2.0.0"
)
