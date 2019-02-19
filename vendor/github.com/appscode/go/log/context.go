package log

import (
	gtx "context"
	"fmt"

	"github.com/appscode/go/context"
	"github.com/golang/glog"
)

// Type logger enables logging with context with glog
type Logger struct {
	ctx string
}

// New creates an context logger instance which opens access to logging
// methods. It accepts interface context to format the values as defined
func New(c gtx.Context) *Logger {
	ctx := context.ID(c)
	if ctx != "" {
		ctx = fmt.Sprintf("[%s] ", ctx)
	}
	return &Logger{ctx}
}

func (l *Logger) Fatal(args ...interface{}) {
	glog.FatalDepth(1, l.ctx, args)
}

func (l *Logger) Fatalln(args ...interface{}) {
	glog.FatalDepth(1, fmt.Sprintln(append([]interface{}{l.ctx}, args...)...))
}

func (l *Logger) Fatalf(format string, args ...interface{}) {
	glog.FatalDepth(1, l.ctx, fmt.Sprintf(format, args...))
}

func (l *Logger) Error(args ...interface{}) {
	glog.ErrorDepth(1, l.ctx, args)
}

func (l *Logger) Errorln(args ...interface{}) {
	glog.ErrorDepth(1, fmt.Sprintln(append([]interface{}{l.ctx}, args...)...))
}

func (l *Logger) Errorf(format string, args ...interface{}) {
	glog.ErrorDepth(1, l.ctx, fmt.Sprintf(format, args...))
}

func (l *Logger) Warning(args ...interface{}) {
	glog.WarningDepth(1, l.ctx, args)
}

func (l *Logger) Warningln(args ...interface{}) {
	glog.WarningDepth(1, fmt.Sprintln(append([]interface{}{l.ctx}, args...)...))
}

func (l *Logger) Warningf(format string, args ...interface{}) {
	glog.WarningDepth(1, l.ctx, fmt.Sprintf(format, args...))
}

func (l *Logger) Info(args ...interface{}) {
	glog.InfoDepth(1, l.ctx, args)
}

func (l *Logger) Infoln(args ...interface{}) {
	glog.InfoDepth(1, fmt.Sprintln(append([]interface{}{l.ctx}, args...)...))
}

func (l *Logger) Infof(format string, args ...interface{}) {
	glog.InfoDepth(1, l.ctx, fmt.Sprintf(format, args...))
}

func (l *Logger) Debug(args ...interface{}) {
	if glog.V(LevelDebug) {
		glog.InfoDepth(1, l.ctx, args)
	}
}

func (l *Logger) Debugln(args ...interface{}) {
	if glog.V(LevelDebug) {
		glog.InfoDepth(1, fmt.Sprintln(append([]interface{}{l.ctx}, args...)...))
	}
}

func (l *Logger) Debugf(format string, args ...interface{}) {
	if glog.V(LevelDebug) {
		glog.InfoDepth(1, l.ctx, fmt.Sprintf(format, args...))
	}
}

// Flush flushes all pending log I/O.
func (l *Logger) Flush() {
	glog.Flush()
}
