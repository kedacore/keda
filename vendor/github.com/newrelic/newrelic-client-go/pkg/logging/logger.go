package logging

// Logger interface implements a simple logger.
type Logger interface {
	Error(string, ...interface{})
	Warn(string, ...interface{})
	Info(string, ...interface{})
	Debug(string, ...interface{})
	Trace(string, ...interface{})
	SetLevel(string)
}
