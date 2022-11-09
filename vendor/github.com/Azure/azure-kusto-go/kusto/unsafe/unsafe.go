// Package unsafe provides methods and types that loosen the native protections of the Kusto package.
package unsafe

// Stmt can be used in optional arguments to kusto.NewStmt() to allow the use of unsafe
// methods on that object.
type Stmt struct {
	// Adds indicates if a Stmt is allowed to use Unsafe.Add().
	// This removes SQL-like injection attack prevention.
	Add bool
	// SuppressWarning allows the Unsafe warning log message to be suppressed for this Stmt.
	SuppressWarning bool
}
