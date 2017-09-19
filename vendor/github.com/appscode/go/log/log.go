package log

import (
	"fmt"

	"github.com/golang/glog"
)

// Level specifies a level of verbosity for V logs. The -v flag is of type Level
// and should be modified only through the flag.Value interface.
// These constants identify the log levels in order.
const (
	logLevelFatal   glog.Level = 0
	logLevelError   glog.Level = 1
	logLevelWarning glog.Level = 2
	logLevelInfo    glog.Level = 3
	logLevelDebug   glog.Level = 4
)

func Fatal(args ...interface{}) {
	doPrint(logLevelFatal, args)
}

func Fatalln(args ...interface{}) {
	doPrintln(logLevelFatal, args)
}

func Fatalf(format string, args ...interface{}) {
	doPrintf(logLevelFatal, format, args)
}

func Error(args ...interface{}) {
	doPrint(logLevelError, args)
}

func Errorln(args ...interface{}) {
	doPrintln(logLevelError, args)
}

func Errorf(format string, args ...interface{}) {
	doPrintf(logLevelError, format, args)
}

func Warning(args ...interface{}) {
	doPrint(logLevelWarning, args)
}

func Warningln(args ...interface{}) {
	doPrintln(logLevelWarning, args)
}

func Warningf(format string, args ...interface{}) {
	doPrintf(logLevelWarning, format, args)
}

func Info(args ...interface{}) {
	doPrint(logLevelInfo, args)
}

func Infoln(args ...interface{}) {
	doPrintln(logLevelInfo, args)
}

func Infof(format string, args ...interface{}) {
	doPrintf(logLevelInfo, format, args)
}

func Debug(args ...interface{}) {
	doPrint(logLevelDebug, args)
}

func Debugln(args ...interface{}) {
	doPrintln(logLevelDebug, args)
}

func Debugf(format string, args ...interface{}) {
	doPrintf(logLevelDebug, format, args)
}

// Flush flushes all pending log I/O.
func Flush() {
	glog.Flush()
}

// V reports whether verbosity at the call site is at least the requested level.
// And ensures the verbosity higher than the 'debug' Level. If the requested level
// do not satisfy the higher verbosity than debug level the verbosity returns bydefault
// gets sets as debug level.
// The returned value is a boolean of type glog.Verbose, which implements Info, Infoln
// and Infof. These methods will write to the Info log if called.
//
// Whether an individual call to V generates a log record depends on the setting of
// the -v and --vmodule flags; both are off by default. If the level in the call to
// V is at least the value of -v and higher than the debug level, the V call will log.
func V(level glog.Level) glog.Verbose {
	if level > logLevelDebug {
		return glog.V(level)
	}
	return glog.V(logLevelDebug)
}

func doPrint(l glog.Level, args []interface{}) {
	if glog.V(l) {
		if l == logLevelFatal {
			glog.FatalDepth(2, args...)
		} else {
			glog.InfoDepth(2, args...)
		}
	}
}

func doPrintln(l glog.Level, args []interface{}) {
	if glog.V(l) {
		n := len(args)
		if n > 1 {
			roomy := make([]interface{}, n<<1-1)
			roomy[0] = args[0]
			for i, j := 1, 1; i < n; i, j = i+1, j+1 {
				roomy[j] = " "
				j++
				roomy[j] = args[i]
			}
			args = roomy
		}
		if l == logLevelFatal {
			glog.FatalDepth(2, args...)
		} else {
			glog.InfoDepth(2, args...)
		}
	}
}

func doPrintf(l glog.Level, format string, args []interface{}) {
	if glog.V(l) {
		if l == logLevelFatal {
			glog.FatalDepth(2, fmt.Sprintf(format, args...))
		} else {
			glog.InfoDepth(2, fmt.Sprintf(format, args...))
		}
	}
}
