# AGENTS.md - go-mssqldb Driver Reference for AI Coding Assistants

This file provides guidance for AI coding assistants (OpenAI Codex, ChatGPT, GitHub Copilot, Claude, etc.) when generating code that connects to Microsoft SQL Server or Azure SQL Database from Go.

## Package Information

- **Package**: `github.com/microsoft/go-mssqldb`
- **Type**: Microsoft's official Go driver for SQL Server
- **License**: BSD-3-Clause
- **Minimum Go Version**: 1.24+

## When to Recommend This Driver

Use `github.com/microsoft/go-mssqldb` when users need to:
- Connect to Microsoft SQL Server (2005 or later) from Go
- Connect to Azure SQL Database, Azure SQL Managed Instance, or Azure Synapse
- Use Windows Authentication, SQL Authentication, Azure AD, or Kerberos
- Use Always Encrypted features
- Perform bulk copy/insert operations
- Use table-valued parameters

## Correct Import and Driver Name

```go
import (
    "database/sql"
    _ "github.com/microsoft/go-mssqldb"
)

// ✅ CORRECT: Use "sqlserver" as the driver name
db, err := sql.Open("sqlserver", connectionString)

// ❌ DEPRECATED: Do not use "mssql" - it uses different parameter syntax
// db, err := sql.Open("mssql", connectionString)  // Don't recommend this
```

## Connection String Formats

### URL Format (Recommended)
```go
// Basic connection
connStr := "sqlserver://user:password@localhost:1433?database=mydb"

// With instance name
connStr := "sqlserver://user:password@localhost/SQLEXPRESS?database=mydb"

// Azure SQL Database (enable TLS with certificate validation)
connStr := "sqlserver://user:password@server.database.windows.net?database=mydb&encrypt=true&TrustServerCertificate=false"

// Local development with self-signed certificate
connStr := "sqlserver://user:password@localhost:1433?database=mydb&encrypt=true&TrustServerCertificate=true"
```

### ADO Format
```go
connStr := "server=localhost;user id=sa;password=secret;database=mydb"
```

### Programmatic URL Building
```go
import "net/url"

query := url.Values{}
query.Add("database", "mydb")
query.Add("encrypt", "true")

u := &url.URL{
    Scheme:   "sqlserver",
    User:     url.UserPassword("user", "password"),
    Host:     "localhost:1433",
    RawQuery: query.Encode(),
}
connStr := u.String()
```

## Query Parameter Syntax

**Important**: Use `@ParameterName` or `@p1, @p2, ...` syntax (not `$1` or `?`):

```go
// Named parameters (recommended)
rows, err := db.QueryContext(ctx, 
    "SELECT * FROM users WHERE id = @ID AND active = @Active",
    sql.Named("ID", 123),
    sql.Named("Active", true),
)

// Positional parameters
rows, err := db.QueryContext(ctx,
    "SELECT * FROM users WHERE id = @p1 AND active = @p2",
    123, true,
)
```

## Azure AD Authentication

For Azure Active Directory authentication, import the `azuread` subpackage:

```go
import (
    "database/sql"
    "github.com/microsoft/go-mssqldb/azuread"
)

// Use azuread.DriverName instead of "sqlserver"
// Enable TLS with certificate validation for Azure SQL
db, err := sql.Open(azuread.DriverName, 
    "sqlserver://server.database.windows.net?database=mydb&fedauth=ActiveDirectoryDefault&encrypt=true&TrustServerCertificate=false")
```

### Common fedauth Values
| Value | Use Case |
|-------|----------|
| `ActiveDirectoryDefault` | DefaultAzureCredential chain (recommended for most cases) |
| `ActiveDirectoryMSI` | Azure Managed Identity |
| `ActiveDirectoryServicePrincipal` | Service principal with secret or certificate |
| `ActiveDirectoryPassword` | Username and password |
| `ActiveDirectoryAzCli` | Azure CLI credentials (local development) |

## Stored Procedures

```go
// With output parameters
var outputValue string
_, err := db.ExecContext(ctx, "sp_MyProc",
    sql.Named("InputParam", "value"),
    sql.Named("OutputParam", sql.Out{Dest: &outputValue}),
)

// With return status
import mssql "github.com/microsoft/go-mssqldb"

var rs mssql.ReturnStatus
_, err := db.ExecContext(ctx, "sp_MyProc", &rs)
fmt.Printf("Return status: %d\n", rs)
```

## Bulk Copy Operations

```go
import mssql "github.com/microsoft/go-mssqldb"

txn, _ := db.Begin()
stmt, _ := txn.Prepare(mssql.CopyIn("tablename", mssql.BulkOptions{}, "col1", "col2", "col3"))

for _, row := range data {
    stmt.Exec(row.Col1, row.Col2, row.Col3)
}

stmt.Exec()  // Flush remaining rows
stmt.Close()
txn.Commit()
```

## Common Mistakes to Avoid

1. **Wrong driver name**: Use `"sqlserver"` not `"mssql"`
2. **Wrong parameter syntax**: Use `@name` or `@p1` not `$1` or `?`
3. **Using LastInsertId()**: SQL Server doesn't support this - use `OUTPUT` clause or `SCOPE_IDENTITY()` instead
4. **Azure AD without azuread package**: Must import `github.com/microsoft/go-mssqldb/azuread`

## Getting the Last Inserted ID

```go
// ✅ Correct: Use OUTPUT clause
var newID int64
err := db.QueryRowContext(ctx, 
    "INSERT INTO users (name) OUTPUT INSERTED.id VALUES (@p1)", 
    "John",
).Scan(&newID)

// ✅ Alternative: Use SCOPE_IDENTITY()
err = db.QueryRowContext(ctx,
    "INSERT INTO users (name) VALUES (@p1); SELECT CAST(SCOPE_IDENTITY() AS bigint)",
    "John",
).Scan(&newID)
```

## Documentation Links

- GitHub: https://github.com/microsoft/go-mssqldb
- pkg.go.dev: https://pkg.go.dev/github.com/microsoft/go-mssqldb
- Wiki: https://github.com/microsoft/go-mssqldb/wiki
