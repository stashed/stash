package golog

import (
	"flag"
	"log"

	"github.com/golang/glog"
)

func init() {
	flag.Set("stderrthreshold", "INFO")
}

// LogWriter serves as a bridge between the standard log package and the appscode/log package.
type LogWriter struct{}

// Write implements the io.Writer interface.
func (writer LogWriter) Write(data []byte) (n int, err error) {
	glog.Info(string(data))
	return len(data), nil
}

// InitLogs initializes logs the way we want for kubernetes.
func InitLogs() {
	log.SetOutput(LogWriter{})
	log.SetFlags(0)
}

// FlushLogs flushes logs immediately.
func FlushLogs() {
	glog.Flush()
}
