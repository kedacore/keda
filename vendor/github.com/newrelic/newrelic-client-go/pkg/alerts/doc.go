/*
Package alerts provides a programmatic API for interacting with the New Relic
Alerts product.  It can be used for a variety of operations, including:

- Creating, reading, updating, and deleting alert policies

- Creating, reading, updating, and deleting alert notification channels

- Associating one or more notification channels with an alert policy

- Creating, reading, updating, and deleting APM alert conditions

- Creating, reading, updating, and deleting NRQL alert conditions

- Creating, reading, updating, and deleting Synthetics alert conditions

- Creating, reading, updating, and deleting multi-location Synthetics conditions

- Creating, reading, updating, and deleting Infrastructure alert conditions

- Creating, reading, updating, and deleting Plugins alert conditions

- Associating one or more alert conditions with a policy

Authentication

You will need a valid API key to communicate with the backend New Relic APIs
that provide this functionality.  Use a Personal API key for authentication.
See the API key documentation below for more information on how to locate this
key:

https://docs.newrelic.com/docs/apis/get-started/intro-apis/types-new-relic-api-keys

*/
package alerts
