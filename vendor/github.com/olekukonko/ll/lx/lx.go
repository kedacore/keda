package lx

import (
	"strings"
	"time"
)

// Formatting constants for log output.
// These constants define the characters used to format log messages, ensuring consistency
// across handlers (e.g., text, JSON, colorized). They are used to construct namespace paths,
// level indicators, and field separators in log entries.
const (
	Space        = " "  // Single space for separating elements (e.g., between level and message)
	DoubleSpace  = "  " // Double space for indentation (e.g., for hierarchical output)
	Slash        = "/"  // Separator for namespace paths (e.g., "parent/child")
	Arrow        = "→"  // Arrow for NestedPath style namespaces (e.g., [parent]→[child])
	LeftBracket  = "["  // Opening bracket for namespaces and fields (e.g., [app])
	RightBracket = "]"  // Closing bracket for namespaces and fields (e.g., [app])
	Colon        = ":"  // Separator after namespace or level (e.g., [app]: INFO:)
	Dot          = "."  // Separator for namespace paths (e.g., "parent.child")
	Newline      = "\n" // Newline for separating log entries or stack trace lines
)

// DefaultEnabled defines the default logging state (disabled).
// It specifies whether logging is enabled by default for new Logger instances in the ll package.
// Set to false to prevent logging until explicitly enabled.
const (
	DefaultEnabled = false // Default state for new loggers (disabled)
)

// Log level constants, ordered by increasing severity.
// These constants define the severity levels for log messages, used to filter logs based
// on the logger’s minimum level. They are ordered to allow comparison (e.g., LevelDebug < LevelWarn).
const (
	LevelNone    LevelType = iota // Debug level for detailed diagnostic information
	LevelInfo                     // Info level for general operational messages
	LevelWarn                     // Warn level for warning conditions
	LevelError                    // Error level for error conditions requiring attention
	LevelFatal                    // Fatal level for critical error conditions
	LevelDebug                    // None level for logs without a specific severity (e.g., raw output)
	LevelUnknown                  // None level for logs without a specific severity (e.g., raw output)
)

// String constants for each level
const (
	DebugString   = "DEBUG"
	InfoString    = "INFO"
	WarnString    = "WARN"
	WarningString = "WARNING"
	ErrorString   = "ERROR"
	FatalString   = "FATAL"
	NoneString    = "NONE"
	UnknownString = "UNKNOWN"

	TextString    = "TEXT"
	JSONString    = "JSON"
	DumpString    = "DUMP"
	SpecialString = "SPECIAL"
	RawString     = "RAW"
)

// Log class constants, defining the type of log entry.
// These constants categorize log entries by their content or purpose, influencing how
// handlers process them (e.g., text, JSON, hex dump).
const (
	ClassText    ClassType = iota // Text entries for standard log messages
	ClassJSON                     // JSON entries for structured output
	ClassDump                     // Dump entries for hex/ASCII dumps
	ClassSpecial                  // Special entries for custom or non-standard logs
	ClassRaw                      // Raw entries for unformatted output
	ClassUnknown                  // Raw entries for unformatted output
)

// Namespace style constants.
// These constants define how namespace paths are formatted in log output, affecting the
// visual representation of hierarchical namespaces.
const (
	FlatPath   StyleType = iota // Formats namespaces as [parent/child]
	NestedPath                  // Formats namespaces as [parent]→[child]
)

// LevelType represents the severity of a log message.
// It is an integer type used to define log levels (Debug, Info, Warn, Error, None), with associated
// string representations for display in log output.
type LevelType int

// String converts a LevelType to its string representation.
// It maps each level constant to a human-readable string, returning "UNKNOWN" for invalid levels.
// Used by handlers to display the log level in output.
// Example:
//
//	var level lx.LevelType = lx.LevelInfo
//	fmt.Println(level.String()) // Output: INFO
func (l LevelType) String() string {
	switch l {
	case LevelDebug:
		return DebugString
	case LevelInfo:
		return InfoString
	case LevelWarn:
		return WarnString
	case LevelError:
		return ErrorString
	case LevelFatal:
		return FatalString
	case LevelNone:
		return NoneString
	default:
		return UnknownString
	}
}

// LevelParse converts a string to its corresponding LevelType.
// It parses a string (case-insensitive) and returns the corresponding LevelType, defaulting to
// LevelUnknown for unrecognized strings. Supports "WARNING" as an alias for "WARN".
func LevelParse(s string) LevelType {
	switch strings.ToUpper(s) {
	case DebugString:
		return LevelDebug
	case InfoString:
		return LevelInfo
	case WarnString, WarningString: // Allow both "WARN" and "WARNING"
		return LevelWarn
	case ErrorString:
		return LevelError
	case NoneString:
		return LevelNone
	default:
		return LevelUnknown
	}
}

// StyleType defines how namespace paths are formatted in log output.
// It is an integer type used to select between FlatPath ([parent/child]) and NestedPath
// ([parent]→[child]) styles, affecting how handlers render namespace hierarchies.
type StyleType int

// Entry represents a single log entry passed to handlers.
// It encapsulates all information about a log message, including its timestamp, severity,
// content, namespace, metadata, and formatting style. Handlers process Entry instances
// to produce formatted output (e.g., text, JSON). The struct is immutable once created,
// ensuring thread-safety in handler processing.
type Entry struct {
	Timestamp time.Time              // Time the log was created
	Level     LevelType              // Severity level of the log (Debug, Info, Warn, Error, None)
	Message   string                 // Log message content
	Namespace string                 // Namespace path (e.g., "parent/child")
	Fields    map[string]interface{} // Additional key-value metadata (e.g., {"user": "alice"})
	Style     StyleType              // Namespace formatting style (FlatPath or NestedPath)
	Error     error                  // Associated error, if any (e.g., for error logs)
	Class     ClassType              // Type of log entry (Text, JSON, Dump, Special, Raw)
	Stack     []byte                 // Stack trace data (if present)
	Id        int                    `json:"-"` // Unique ID for the entry, ignored in JSON output
}

// Handler defines the interface for processing log entries.
// Implementations (e.g., TextHandler, JSONHandler) format and output log entries to various
// destinations (e.g., stdout, files). The Handle method returns an error if processing fails,
// allowing the logger to handle output failures gracefully.
// Example (simplified handler implementation):
//
//	type MyHandler struct{}
//	func (h *MyHandler) Handle(e *Entry) error {
//	    fmt.Printf("[%s] %s: %s\n", e.Namespace, e.Level.String(), e.Message)
//	    return nil
//	}
type Handler interface {
	Handle(e *Entry) error // Processes a log entry, returning any error
}

// Timestamper defines an interface for handlers that support timestamp configuration.
// It includes a method to enable or disable timestamp logging and optionally set the timestamp format.
type Timestamper interface {
	// Timestamped enables or disables timestamp logging and allows specifying an optional format.
	// Parameters:
	//   enable: Boolean to enable or disable timestamp logging
	//   format: Optional string(s) to specify the timestamp format
	Timestamped(enable bool, format ...string)
}

// ClassType represents the type of a log entry.
// It is an integer type used to categorize log entries (Text, JSON, Dump, Special, Raw),
// influencing how handlers process and format them.
type ClassType int

// String converts a ClassType to its string representation.
// It maps each class constant to a human-readable string, returning "UNKNOWN" for invalid classes.
// Used by handlers to indicate the entry type in output (e.g., JSON fields).
// Example:
//
//	var class lx.ClassType = lx.ClassText
//	fmt.Println(class.String()) // Output: TEST
func (t ClassType) String() string {
	switch t {
	case ClassText:
		return TextString
	case ClassJSON:
		return JSONString
	case ClassDump:
		return DumpString
	case ClassSpecial:
		return SpecialString
	case ClassRaw:
		return RawString
	default:
		return UnknownString
	}
}

// ParseClass converts a string to its corresponding ClassType.
// It parses a string (case-insensitive) and returns the corresponding ClassType, defaulting to
// ClassUnknown for unrecognized strings.
func ParseClass(s string) ClassType {
	switch strings.ToUpper(s) {
	case TextString:
		return ClassText
	case JSONString:
		return ClassJSON
	case DumpString:
		return ClassDump
	case SpecialString:
		return ClassSpecial
	case RawString:
		return ClassRaw
	default:
		return ClassUnknown
	}
}
