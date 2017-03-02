package log

import (
	"github.com/golang/glog"
)

type Context interface {
	String() string
}

// Type logger enables logging with context with glog
type Logger struct {
	c Context
}

// New creates an context logger instance which opens access to logging
// methods. It accepts interface context to format the values as defined
func New(c Context) *Logger {
	if c == nil {
		c = &defaultContext{}
	}
	return &Logger{
		c: c,
	}
}

type defaultContext struct{}

func (d *defaultContext) String() string {
	return ""
}

func (l *Logger) Fatal(args ...interface{}) {
	formatArgs(l.c, &args)
	doPrint(logLevelFatal, args)
}

func (l *Logger) Fatalln(args ...interface{}) {
	formatArgs(l.c, &args)
	doPrintln(logLevelFatal, args)
}

func (l *Logger) Fatalf(format string, args ...interface{}) {
	formatArgs(l.c, &args)
	doPrintf(logLevelFatal, formatString(l.c, format), args)
}

func (l *Logger) Error(args ...interface{}) {
	formatArgs(l.c, &args)
	doPrint(logLevelError, args)
}

func (l *Logger) Errorln(args ...interface{}) {
	formatArgs(l.c, &args)
	doPrintln(logLevelError, args)
}

func (l *Logger) Errorf(format string, args ...interface{}) {
	formatArgs(l.c, &args)
	doPrintf(logLevelError, formatString(l.c, format), args)
}

func (l *Logger) Warning(args ...interface{}) {
	formatArgs(l.c, &args)
	doPrint(logLevelWarning, args)
}

func (l *Logger) Warningln(args ...interface{}) {
	formatArgs(l.c, &args)
	doPrintln(logLevelWarning, args)
}

func (l *Logger) Warningf(format string, args ...interface{}) {
	formatArgs(l.c, &args)
	doPrintf(logLevelWarning, formatString(l.c, format), args)
}

func (l *Logger) Info(args ...interface{}) {
	formatArgs(l.c, &args)
	doPrint(logLevelInfo, args)
}

func (l *Logger) Infoln(args ...interface{}) {
	formatArgs(l.c, &args)
	doPrintln(logLevelInfo, args)
}

func (l *Logger) Infof(format string, args ...interface{}) {
	formatArgs(l.c, &args)
	doPrintf(logLevelInfo, formatString(l.c, format), args)
}

func (l *Logger) Debug(args ...interface{}) {
	formatArgs(l.c, &args)
	doPrint(logLevelDebug, args)
}

func (l *Logger) Debugln(args ...interface{}) {
	formatArgs(l.c, &args)
	doPrintln(logLevelDebug, args)
}

func (l *Logger) Debugf(format string, args ...interface{}) {
	formatArgs(l.c, &args)
	doPrintf(logLevelDebug, formatString(l.c, format), args)
}

// Flush flushes all pending log I/O.
func (l *Logger) Flush() {
	glog.Flush()
}

// V reports whether verbosity at the call site is at least the requested level.
// And ensures the verbosity higher than the 'debug' Level. If the requested level
// do not satisfy the higher verbosity than debug level the verbosity returns default
// gets sets as debug level.
// The returned value is a boolean of type glog.Verbose, which implements Info, Infoln
// and Infof. These methods will write to the Info log if called.
//
// Whether an individual call to V generates a log record depends on the setting of
// the -v and --vmodule flags; both are off by default. If the level in the call to
// V is at least the value of -v and higher than the debug level, the V call will log.
func (l *Logger) V(level glog.Level) glog.Verbose {
	if level > logLevelDebug {
		return glog.V(level)
	}
	return glog.V(logLevelDebug)
}

func formatArgs(c Context, args *[]interface{}) {
	if c != nil {
		if val := c.String(); val != "" {
			// appending the context to the argument list as first argument.
			*args = append((*args)[:0], append([]interface{}{val}, (*args)[0:]...)...)
		}
	}
}

func formatString(c Context, format string) string {
	if c == nil {
		return format
	}
	if val := c.String(); val != "" {
		// if context is not nil, adding context values as formatted by the provider.
		format = "%s " + format
	}
	return format
}
