/*
Package synthetics provides a programmatic API for interacting with the New Relic
Synthetics API.  It can be used for a variety of operations, including:

- Creating, reading, updating, and deleting Synthetics monitors

- Reading and updating Synthetics monitor scripts

- Associating Synthetics monitor scripts with existing Synthetics monitors

- Creating, reading, updating, and deleting Synthetics secure credentials

Synthetics labels have been EOL'd as of July 20, 2020. This functionality has been
superceded by entity tags, which can be provisioned via the `entities` package.
More information can be found here:

https://discuss.newrelic.com/t/end-of-life-notice-synthetics-labels-and-synthetics-apm-group-by-tag/103781

# Authentication

You will need a Personal API key to communicate with the backend New Relic API
that provides this functionality.  See the API key documentation below for more
information on how to locate this keys:

https://docs.newrelic.com/docs/apis/get-started/intro-apis/types-new-relic-api-keys
*/
package synthetics
