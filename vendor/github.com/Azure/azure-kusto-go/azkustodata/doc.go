/*
Package azkustodata provides a client for querying and managing data in Azure Data Explorer (Kusto) clusters.

The package supports running both Kusto Query Language (KQL) queries and management commands, with built-in support for
streaming responses and mapping results to Go structs.

To start using this package, create an instance of the Client, passing in a connection string built using the
NewConnectionStringBuilder() function. The Client can be authenticated using various methods from the Azure Identity package.

Example For Querying the Kusto cluster:

	kcsb := azkustodata.NewConnectionStringBuilder("https://help.kusto.windows.net/").WithDefaultAzureCredential()
	client, err := azkustodata.New(kcsb)

	if err != nil {
		panic(err)
	}

	defer client.Close() // Always close the client when done.

	ctx := context.Background()
	dataset, err := client.IterativeQuery(ctx, "Samples", kql.New("PopulationData"))

	// Don't forget to close the dataset when you're done.
	defer dataset.Close()

	primaryResult := <-dataset.Tables() // The first table in the dataset will be the primary results.

	// Make sure to check for errors.
	if primaryResult.Err() != nil {
		panic("add error handling")
	}

	for rowResult := range primaryResult.Table().Rows() {
		if rowResult.Err() != nil {
			panic("add error handling")
		}
		row := rowResult.Row()

		fmt.Println(row) // As a convenience, printing a *table.Row will output csv
		// or Access the columns directly
		fmt.Println(row.IntByName("EventId"))
		fmt.Println(row.StringByIndex(1))
	}

Example for Management/Administration commands:

	kcsb := azkustodata.NewConnectionStringBuilder("https://help.kusto.windows.net/").WithDefaultAzureCredential()
	client, err := azkustodata.New(kcsb)

	if err != nil {
		panic(err)
	}

	defer client.Close() // Always close the client when done.

	ctx := context.Background()
	dataset, err := client.Mgmt(ctx, "Samples", kql.New(".show tables"))

	table := dataset.Tables()[0]

	// convert the table to a struct
	structs, err := query.ToStructs[myStruct](table)

To handle results, the package provides utilities to directly stream rows, fetch tables into memory, and map results to structs.

For complete documentation, please visit:
https://github.com/Azure/azure-kusto-go
https://pkg.go.dev/github.com/Azure/azure-kusto-go/azkustodata
*/
package azkustodata
