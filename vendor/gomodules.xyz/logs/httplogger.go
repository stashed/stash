package logs

import (
	"bytes"

	"k8s.io/klog/v2"
)

// HTTPLogger serves as a bridge between the standard log package and the klog package.
type HTTPLogger struct{}

// Write implements the io.Writer interface.
func (writer HTTPLogger) Write(data []byte) (n int, err error) {
	if bytes.Contains(data, []byte("Content-Length: ")) ||
		bytes.Contains(data, []byte("Content-Type: ")) ||
		bytes.Contains(data, []byte("User-Agent: ")) ||
		bytes.Contains(data, []byte("Set-Cookie: ")) {
		if klog.V(8).Enabled() {
			klog.InfoDepth(1, string(data))
			return len(data), nil
		}
		return 0, nil
	}
	klog.InfoDepth(1, string(data))
	return len(data), nil
}
