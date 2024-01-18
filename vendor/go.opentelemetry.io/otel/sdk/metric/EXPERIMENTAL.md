# Experimental Features

The metric SDK contains features that have not yet stabilized in the OpenTelemetry specification.
These features are added to the OpenTelemetry Go metric SDK prior to stabilization in the specification so that users can start experimenting with them and provide feedback.

These feature may change in backwards incompatible ways as feedback is applied.
See the [Compatibility and Stability](#compatibility-and-stability) section for more information.

## Features

- [Cardinality Limit](#cardinality-limit)

### Cardinality Limit

The cardinality limit is the hard limit on the number of metric streams that can be collected for a single instrument.

This experimental feature can be enabled by setting the `OTEL_GO_X_CARDINALITY_LIMIT` environment value.
The value must be an integer value.
All other values are ignored.

If the value set is less than or equal to `0`, no limit will be applied.

#### Examples

Set the cardinality limit to 2000.

```console
export OTEL_GO_X_CARDINALITY_LIMIT=2000
```

Set an infinite cardinality limit (functionally equivalent to disabling the feature).

```console
export OTEL_GO_X_CARDINALITY_LIMIT=-1
```

Disable the cardinality limit.

```console
unset OTEL_GO_X_CARDINALITY_LIMIT
```

## Compatibility and Stability

Experimental features do not fall within the scope of the OpenTelemetry Go versioning and stability [policy](../../VERSIONING.md).
These features may be removed or modified in successive version releases, including patch versions.

When an experimental feature is promoted to a stable feature, a migration path will be included in the changelog entry of the release.
There is no guarantee that any environment variable feature flags that enabled the experimental feature will be supported by the stable version.
If they are supported, they may be accompanied with a deprecation notice stating a timeline for the removal of that support.
