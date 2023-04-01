// Copyright 2020 Microsoft Corporation. All rights reserved.
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//
// Additional licenses for third-pary packages can be found in third_party_licenses/

/*
Package kusto provides a client for accessing Azure Data Explorer, also known as Kusto.

For details on the Azure Kusto service, see: https://azure.microsoft.com/en-us/services/data-explorer/

For general documentation on APIs and the Kusto query language, see: https://docs.microsoft.com/en-us/azure/data-explorer/

# Creating an Authorizer and a Client

To begin using this package, create an Authorizer and a client targeting your Kusto endpoint:

	// auth package is: "github.com/Azure/go-autorest/autorest/azure/auth"

	authorizer := kusto.Authorization{
		Config: auth.NewClientCredentialsConfig(clientID, clientSecret, tenantID),
	}

	client, err := kusto.New(endpoint, authorizer)
	if err != nil {
		panic("add error handling")
	}

For more examples on ways to create an Authorization object, see the Authorization object documentation.

# Querying for Rows

Kusto provides a single method for querying, Query().  Query uses a Stmt object to provides SQL-like injection protection
and accepts only string constants for arguments.

	// table package is: data/table

	// Query our database table "systemNodes" for the CollectionTimes and the NodeIds.
	iter, err := client.Query(ctx, "database", kusto.NewStmt("systemNodes | project CollectionTime, NodeId"))
	if err != nil {
		panic("add error handling")
	}
	defer iter.Stop()

	// .Do() will call the function for every row in the table.
	err = iter.Do(
		func(row *table.Row) error {
			if row.Replace {
				fmt.Println("---") // Replace flag indicates that the query result should be cleared and replaced with this row
			}
			fmt.Println(row) // As a convenience, printing a *table.Row will output csv
			return nil
		},
	)
	if err != nil {
		panic("add error handling")
	}

# Querying Rows Into Structs

Keeping our query the same, instead of printing the Rows we will simply put them into a slice of structs

	// NodeRec represents our Kusto data that will be returned.
	type NodeRec struct {
		// ID is the table's NodeId. We use the field tag here to to instruct our client to convert NodeId to ID.
		ID int64 `kusto:"NodeId"`
		// CollectionTime is Go representation of the Kusto datetime type.
		CollectionTime time.Time
	}

	iter, err := client.Query(ctx, "database", kusto.NewStmt("systemNodes | project CollectionTime, NodeId"))
	if err != nil {
		panic("add error handling")
	}
	defer iter.Stop()

	recs := []NodeRec{}
	err = iter.Do(
		func(row *table.Row) error {
			rec := NodeRec{}
			if err := row.ToStruct(&rec); err != nil {
				return err
			}
			if row.Replace {
				recs = recs[:0]  // Replace flag indicates that the query result should be cleared and replaced with this row
			}
			recs = append(recs, rec)
			return nil
		},
	)
	if err != nil {
		panic("add error handling")
	}

A struct object can use fields to store the Kusto values as normal Go values, pointers to Go values and as value.Kusto types.
The value.Kusto types are useful when you need to distiguish between the zero value of a variable and the value not being
set in Kusto.

All value.Kusto types have a .Value and .Valid field. .Value is the native Go value, .Valid is a bool which
indicates if the value was set. More information can be found in the sub-package data/value.

The following is a conversion table from the Kusto column types to native Go values within a struct that are allowed:

	From Kusto Type			To Go Kusto Type
	==============================================================================
	bool				value.Bool, bool, *bool
	------------------------------------------------------------------------------
	datetime			value.DateTime, time.Time, *time.Time
	------------------------------------------------------------------------------
	dynamic				value.Dynamic, string, *string
	------------------------------------------------------------------------------
	guid				value.GUID, uuid.UUID, *uuid.UUID
	------------------------------------------------------------------------------
	int				value.Int, int32, *int32
	------------------------------------------------------------------------------
	long				value.Long, int64, *int64
	------------------------------------------------------------------------------
	real				value.Real, float64, *float64
	------------------------------------------------------------------------------
	string				value.String, string, *string
	------------------------------------------------------------------------------
	timestamp			value.Timestamp, time.Duration, *time.Duration
	------------------------------------------------------------------------------
	decimal				value.Decimal, string, *string
	==============================================================================

For more information on Kusto scalar types, see: https://docs.microsoft.com/en-us/azure/kusto/query/scalar-data-types/

# Stmt

Every query is done using a Stmt. A Stmt is built with Go string constants and can do variable substitution
using Kusto's Query Paramaters.

	// rootStmt builds a query that will gather all nodes in the DB.
	rootStmt := kusto.NewStmt("systemNodes | project CollectionTime, NodeId")

	// singleNodeStmt creates a new Stmt based on rootStmt and adds a "where" clause to find a single node.
	// We pass a definition that sets the word ParamNodeId to a variable that will be substituted for a
	// Kusto Long type (which is a a Go int64).
	singleNodeStmt := rootStmt.Add(" | where NodeId == ParamNodeId").MustDefinitions(
		kusto.NewDefinitions().Must(
			kusto.ParamTypes{
				"ParamNodeId": kusto.ParamType{Type: types.Long},
			},
		),
	)

	// Query using our singleNodeStmt, variable substituting for ParamNodeId
	iter, err := client.Query(
		ctx,
		"database",
		singleNode.MustParameters(
			kusto.NewParameters().Must(
				kusto.QueryValues{"ParamNodeId": int64(100)},
			),
		),
	)

# Ingest

Support for Kusto ingestion from local files, Azure Blob Storage and streaming is supported in the sub-package ingest.
See documentation in that package for more details

# Mocking

To support mocking for this client in your code for hermetic testing purposes, this client supports mocking the data
returned by our RowIterator object. Please see the MockRows documentation for code examples.

# Package Examples

Below you will find a simple and complex example of doing Query() the represent compiled code:
*/
package kusto
