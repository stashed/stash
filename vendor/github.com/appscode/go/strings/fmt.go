package strings

import (
	"strings"
)

func Fmt(s string) string {
	stripper := &Stripe{
		Result: "",
	}
	stripper.Write(s)
	return stripper.Result
}

// Striplines wraps an output stream, stripping runs of consecutive empty lines.
// You must call Flush before the output stream will be complete.
// Implements io.WriteCloser, Writer, Closer.
type Stripe struct {
	Result      string
	lastLine    []byte
	currentLine []byte
}

func (w *Stripe) Write(p string) (int, error) {
	totalN := 0
	s := string(p)
	if !strings.Contains(s, "\n") {
		w.currentLine = append(w.currentLine, p...)
		return 0, nil
	}
	cur := string(append(w.currentLine, p...))
	lastN := strings.LastIndex(cur, "\n")
	s = cur[:lastN]
	for _, line := range strings.Split(s, "\n") {
		n, err := w.writeLn(line + "\n")
		w.lastLine = []byte(line)
		if err != nil {
			return totalN, err
		}
		totalN += n
	}
	rem := cur[(lastN + 1):]
	w.currentLine = []byte(rem)
	return totalN, nil
}

// Close flushes the last of the output into the underlying writer.
func (w *Stripe) Close() error {
	_, err := w.writeLn(string(w.currentLine))
	return err
}

func (w *Stripe) writeLn(line string) (n int, err error) {
	if strings.TrimSpace(string(w.lastLine)) == "" && strings.TrimSpace(line) == "" {
		return 0, nil
	} else {
		w.Result = w.Result + line
		return len(line), nil
	}
}
