// Package mssql is Microsoft's official Go driver for SQL Server and Azure SQL Database.
//
// This package implements the TDS protocol used to connect to Microsoft SQL Server
// (SQL Server 2005 and later) and Azure SQL Database.
//
// # Driver Registration
//
// This package registers the following drivers:
//
//	sqlserver: preferred driver; uses native "@" parameter placeholder names and does no pre-processing.
//	mssql: legacy compatibility driver (deprecated); performs query token replacement and may be removed in a future release.
//
// Use "sqlserver" as the driver name with database/sql.Open:
//
//	db, err := sql.Open("sqlserver", "sqlserver://user:password@localhost:1433?database=mydb")
//
// # Connection String Formats
//
// URL format (recommended):
//
//	sqlserver://user:password@localhost:1433?database=mydb
//	sqlserver://user:password@localhost/instance?database=mydb
//
// ADO format:
//
//	server=localhost;user id=sa;password=secret;database=mydb
//
// ODBC format:
//
//	odbc:server=localhost;user id=sa;password=secret;database=mydb
//
// # Query Parameters
//
// Use "@ParameterName" or "@p1", "@p2", etc. for query parameters:
//
//	// Named parameters
//	db.Query("SELECT * FROM users WHERE id = @ID", sql.Named("ID", 123))
//
//	// Positional parameters
//	db.Query("SELECT * FROM users WHERE id = @p1", 123)
//
// # Azure AD Authentication
//
// For Azure Active Directory authentication, import the azuread subpackage:
//
//	import "github.com/microsoft/go-mssqldb/azuread"
//
//	db, err := sql.Open(azuread.DriverName,
//	    "sqlserver://server.database.windows.net?database=mydb&fedauth=ActiveDirectoryDefault&encrypt=true&TrustServerCertificate=false")
//
// # Features
//
//   - SQL Server 2005+ and Azure SQL Database support
//   - Windows Authentication, SQL Authentication, Azure AD, Kerberos
//   - Always Encrypted column encryption
//   - Bulk copy operations via [CopyIn]
//   - Stored procedures with output parameters
//   - Table-valued parameters
//   - Named pipes and shared memory on Windows
//
// For complete documentation, see https://github.com/microsoft/go-mssqldb
package mssql
