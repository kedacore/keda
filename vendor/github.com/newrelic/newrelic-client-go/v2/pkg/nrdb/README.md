# NRDB Package

The `nrdb` package provides a programmatic API for interacting with NRDB, New Relic's Datastore, allowing you to execute NRQL queries and process their results in Go applications.

## Overview

This package contains several methods for querying New Relic's NRDB database using NRQL (New Relic Query Language). The core functionality revolves around executing NRQL queries against your New Relic accounts and handling the structured response data.

## Query Functions

### Original Query Function

The original `Query()` function is the standard (preferred) way to execute NRQL queries:

```go
func (n *Nrdb) Query(accountID int, query NRQL) (*NRDBResultContainer, error)
```

This function processes the query and returns an `NRDBResultContainer` with the results.

#### Usage Example

```go
client := newrelic.New(config)
accountID := 12345678

query := `SELECT count(*) FROM Transaction`
results, err := client.Nrdb.Query(accountID, nrdb.NRQL(query))
if err != nil {
    log.Fatal("Error running query:", err)
}

// Access the results
fmt.Printf("Query results: %v\n", results.Results)
```

### Limitation with FACET + TIMESERIES Queries

The original `Query()` function has a limitation when handling queries that combine both `FACET` and `TIMESERIES` clauses. In these cases, the NerdGraph API may return `otherResult` and `totalResult` as arrays rather than single objects, which conflicts with the `NRDBResultContainer` structure that expects single objects for these fields.

This can lead to JSON unmarshalling errors like:

```
json: cannot unmarshal array into Go struct field .data.Actor.Account.NRQL.otherResult of type nrdb.NRDBResult
```

#### Example of a Problematic Query

```go
// This query will cause unmarshalling errors with the original Query() function
query := `SELECT count(*) FROM Transaction FACET appName TIMESERIES 1 hour SINCE 1 day ago`
```

### Enhanced Query Function

To address the limitation with FACET+TIMESERIES queries, we've introduced the `PerformNRQLQuery()` function:

```go
func (n *Nrdb) PerformNRQLQuery(accountID int, query NRQL) (*NRDBResultContainerMultiResultCustomized, error)
```

This function uses a customized result container that handles both single objects and arrays for `otherResult` and `totalResult` fields, making it compatible with all types of NRQL queries, including those with combined `FACET` and `TIMESERIES` clauses.

For consistency and ease of use, the `otherResult` and `totalResult` fields are **always returned as `NRDBMultiResultCustomized`** (which is a type alias for `[]NRDBResult`):
- When the API returns a single object, it's wrapped in an array with one element
- When the API returns an array, it's used directly
- When the field is null, an empty array is returned

This eliminates the need for type checking and assertions when using these fields.

#### Usage Example

```go
client := newrelic.New(config)
accountID := 12345678

// Using a query that combines FACET and TIMESERIES
query := `SELECT count(*) FROM Transaction FACET appName TIMESERIES 1 hour SINCE 1 day ago`
results, err := client.Nrdb.PerformNRQLQuery(accountID, nrdb.NRQL(query))
if err != nil {
    log.Fatal("Error running query:", err)
}

// No need for type assertions - otherResult and totalResult are always NRDBMultiResultCustomized
for i, result := range results.OtherResult {
    fmt.Printf("Other Result %d: %v\n", i, result)
}

for i, result := range results.TotalResult {
    fmt.Printf("Total Result %d: %v\n", i, result)
}

// Even for simple queries, the result is consistent
query = `SELECT count(*) FROM Transaction`
results, err = client.Nrdb.PerformNRQLQuery(accountID, nrdb.NRQL(query))
if err != nil {
    log.Fatal("Error running query:", err)
}

// For a simple query, we'll still get arrays (with 0 or 1 elements)
if len(results.OtherResult) > 0 {
    fmt.Printf("Other Result: %v\n", results.OtherResult[0])
}
```

## When to Use Each Function

1. **Use `Query()` when**:
   - You're executing simple NRQL queries without combined `FACET` and `TIMESERIES` clauses, and would like to use the standardized method; and/or
   - The queries you've specified with `Query()` are expected to return single objects for `otherResult` and `totalResult`.

2. **Use `PerformNRQLQuery()` when**:
   - You're executing queries that combine `FACET` and `TIMESERIES` clauses, or similar clauses leading to complex NRQL queries, expected to return arrays for `otherResult` and `totalResult`, and/or
   - Your code deals in performing _many_ kinds of NRQL queries, with some expected to return results in simpler structures while some others (e.g. NRQL queries with multiple clauses as stated above) expected to return results in lists, for which you need the function called to handle consistent array packing for `otherResult` and `totalResult` fields in either of these scenarios.

## Query Types and Expected Response Formats

| Query Type | Original Function (`Query()`) | Enhanced Function (`PerformNRQLQuery()`)           |
|------------|-------------------------------|----------------------------------------------------|
| Simple | Single Object                 | `NRDBMultiResultCustomized` with 1 element         |
| FACET only | Single Object                 | `NRDBMultiResultCustomized` with 1 element         |
| TIMESERIES only | Single Object                 | `NRDBMultiResultCustomized` with 1 element         |
| FACET + TIMESERIES | **Error** (incompatible)      | `NRDBMultiResultCustomized` with multiple elements |

## Troubleshooting

If you encounter JSON unmarshalling errors when using `Query()` with complex queries, switch to `PerformNRQLQuery()` to handle the variable response structure.

If you need to maintain compatibility with code that expects the standard `NRDBResultContainer`, but want to use `PerformNRQLQuery()` for its flexibility, you can handle the `NRDBMultiResultCustomized` appropriately:

```go
customResults, err := client.Nrdb.PerformNRQLQuery(accountID, nrdb.NRQL(query))
if err != nil {
    return nil, err
}

// Access the first element if available
if len(customResults.OtherResult) > 0 {
    singleOtherResult := customResults.OtherResult[0]
    // Use with code expecting NRDBResult
}
```
## Type Definitions

The package includes specialized types for handling different response formats:

- `NRDBResult`: A single result object (map[string]interface{})
- `NRDBMultiResultCustomized`: A collection of result objects ([]NRDBResult)
- `NRDBResultContainerMultiResultCustomized`: A container that uses `NRDBMultiResultCustomized` for `otherResult` and `totalResult` fields

## Further Reading

- [New Relic NRQL Documentation](https://docs.newrelic.com/docs/insights/nrql-new-relic-query-language/nrql-resources/nrql-syntax-components-functions)
- [NerdGraph API Documentation](https://docs.newrelic.com/docs/apis/graphql-api/tutorials/query-nrql-through-new-relic-graphql-api)
