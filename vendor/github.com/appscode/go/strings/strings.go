package strings

import (
	"sort"
	"strings"
	"unicode/utf8"
)

// Benchmark 19246 ns/op.
func Reverse(s string) string {
	size := len(s)
	buf := make([]byte, size)
	for start := 0; start < size; {
		r, n := utf8.DecodeRuneInString(s[start:])
		start += n
		utf8.EncodeRune(buf[size-start:], r)
	}
	return string(buf)
}

func PrefixFold(s, prefix string) bool {
	return len(s) >= len(prefix) && strings.EqualFold(prefix, s[:len(prefix)])
}

func IsEmpty(s *string) bool {
	return s == nil || *s == ""
}

func IsBothAlphaNum(a string) bool {
	alpha := false
	num := false
	for _, c := range a {
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') {
			alpha = true
		} else if c >= '0' && c <= '9' {
			num = true
		}
		if alpha && num {
			return true
		}
	}
	return false
}

func Contains(a []string, e string) bool {
	for _, s := range a {
		if s == e {
			return true
		}
	}
	return false
}

// Allowed char: [a-z0-9]([a-z0-9-]*[a-z0-9])?
// Makes it safe as a subdomain
func DomainForm(s string) string {
	runes := make([]rune, len(s))
	for i, r := range strings.ToLower(s) {
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r == '-') || (r == '.') {
			runes[i] = r
		} else if r == '_' {
			runes[i] = '-' // _ --> -
		}
	}
	return strings.Trim(string(runes), "-")
}

func Filter(s []string, f func(string) bool) []string {
	ret := make([]string, 0)
	for _, elm := range s {
		if !f(elm) {
			ret = append(ret, elm)
		}
	}
	return ret
}

func Join(a []*string, sep string) string {
	b := make([]string, len(a))
	for i, s := range a {
		b[i] = *s
	}
	return strings.Join(b, sep)
}

func EqualSlice(a, b []string) bool {
	if a == nil && b == nil {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	if len(a) != len(b) {
		return false
	}

	// Copy slices
	aCopy := append([]string(nil), a...)
	bCopy := append([]string(nil), b...)

	sort.Strings(aCopy)
	sort.Strings(bCopy)

	for i := range aCopy {
		if aCopy[i] != bCopy[i] {
			return false
		}
	}
	return true
}
