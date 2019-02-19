package log

import (
	"fmt"

	"github.com/golang/glog"
)

// Level specifies a level of verbosity for V logs. The -v flag is of type Level
// and should be modified only through the flag.Value interface.
// These constants identify the log levels in order.
const (
	LevelDebug glog.Level = 4
)

func Fatal(args ...interface{}) {
	glog.FatalDepth(1, args...)
}

func Fatalln(args ...interface{}) {
	glog.FatalDepth(1, fmt.Sprintln(args...))
}

func Fatalf(format string, args ...interface{}) {
	glog.FatalDepth(1, fmt.Sprintf(format, args...))
}

func Error(args ...interface{}) {
	glog.ErrorDepth(1, args...)
}

func Errorln(args ...interface{}) {
	glog.ErrorDepth(1, fmt.Sprintln(args...))
}

func Errorf(format string, args ...interface{}) {
	glog.ErrorDepth(1, fmt.Sprintf(format, args...))
}

func Warning(args ...interface{}) {
	glog.WarningDepth(1, args...)
}

func Warningln(args ...interface{}) {
	glog.WarningDepth(1, fmt.Sprintln(args...))
}

func Warningf(format string, args ...interface{}) {
	glog.WarningDepth(1, fmt.Sprintf(format, args...))
}

func Info(args ...interface{}) {
	glog.InfoDepth(1, args...)
}

func Infoln(args ...interface{}) {
	glog.InfoDepth(1, fmt.Sprintln(args...))
}

func Infof(format string, args ...interface{}) {
	glog.InfoDepth(1, fmt.Sprintf(format, args...))
}

func Debug(args ...interface{}) {
	if glog.V(LevelDebug) {
		glog.InfoDepth(1, args...)
	}
}

func Debugln(args ...interface{}) {
	if glog.V(LevelDebug) {
		glog.InfoDepth(1, fmt.Sprintln(args...))
	}
}

func Debugf(format string, args ...interface{}) {
	if glog.V(LevelDebug) {
		glog.InfoDepth(1, fmt.Sprintf(format, args...))
	}
}

// Flush flushes all pending log I/O.
func Flush() {
	glog.Flush()
}
