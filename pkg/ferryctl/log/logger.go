package log

type Logger interface {
	Printf(format string, args ...interface{})
}
