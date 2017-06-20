package strings

import "strings"

func Val(v string, def string) string {
	trimmed := strings.TrimSpace(v)
	if trimmed == "" {
		return def
	}
	return trimmed
}
