/*
Package apm provides a programmatic API for interacting with the New Relic
APM product.  It can be used for a variety of operations, including:

- Reading, updating, and deleting APM applications

- Creating, reading, and deleting APM deployment markers

- Reading APM key transactions

- Creating, reading, and deleting APM labels

# Authentication

You will need a valid Personal API key to communicate with the backend New Relic
APIs that provide this functionality. See the API key documentation below for
more information on how to locate this key:

https://docs.newrelic.com/docs/apis/get-started/intro-apis/types-new-relic-api-keys

# Labels

New Relic One entity tags are currently the preferred method for organizing your
New Relic resources.  Consider using entity tags via the `entities` package if
you are just getting started.  More information about entity tags and APM labels
can be found at the following URL:

https://docs.newrelic.com/docs/new-relic-one/use-new-relic-one/core-concepts/tagging-use-tags-organize-group-what-you-monitor#labels
*/
package apm
