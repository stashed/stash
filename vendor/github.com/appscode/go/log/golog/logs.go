package golog

import (
	"flag"
	"log"

	. "github.com/appscode/go/log"
)

func init() {
	flag.Set("logtostderr", "true")
}

// LogWriter serves as a bridge between the standard log package and the appscode/log package.
type LogWriter struct{}

// Write implements the io.Writer interface.
func (writer LogWriter) Write(data []byte) (n int, err error) {
	Info(string(data))
	return len(data), nil
}

// InitLogs initializes logs the way we want for kubernetes.
func InitLogs() {
	log.SetOutput(LogWriter{})
	log.SetFlags(0)
}

// FlushLogs flushes logs immediately.
func FlushLogs() {
	Flush()
}
