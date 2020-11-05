package password

import (
	"crypto/rand"
	"math/big"
)

type Charset int

const (
	Uppercase     Charset = 1 << iota // 1 << 0 which is 00000001
	Lowercase                         // 1 << 1 which is 00000010
	Numbers                           // 1 << 2 which is 00000100
	Unreserved                        // 1 << 3 which is 00001000
	Reserved                          // 1 << 4 which is 00010000
	SimpleSymbols                     // 1 << 5 which is 00100000
	Symbols                           // 1 << 6 which is 01000000
	AlphaNum      = Uppercase | Lowercase | Numbers
	Default       = Uppercase | Lowercase | Numbers | SimpleSymbols
)

var (
	uppercase          = []byte(`ABCDEFGHIJKLMNOPQRSTUVWXYZ`)
	len_uppercase      = len(uppercase)
	lowercase          = []byte(`abcdefghijklmnopqrstuvwxyz`)
	len_lowercase      = len(lowercase)
	numbers            = []byte(`0123456789`)
	len_numbers        = len(numbers)
	unreserved         = []byte(`-._~`) // ref: https://perishablepress.com/stop-using-unsafe-characters-in-urls/
	len_unreserved     = len(unreserved)
	reserved           = []byte(`!#$&'()*+,/:;=?@[]`)
	len_reserved       = len(reserved)
	simple_symbols     = []byte(`!$&()*,-.;=_~`) // ref: https://github.com/golang/go/blob/release-branch.go1.15/src/net/url/url.go#L1158-L1186 ,  missing: Unreserved | Reserved - #/?[]':+@
	len_simple_symbols = len(simple_symbols)
	symbols            = []byte(`!"#$%&'()*+,-./:;<=>?@^[\]_{|}~` + "`")
	len_symbols        = len(symbols)
)

func Generate(n int) string {
	if n <= 2 {
		return GenerateForCharset(n, AlphaNum)
	}
	return GenerateForCharset(1, AlphaNum) + GenerateForCharset(n-2, Default) + GenerateForCharset(1, AlphaNum)
}

func GenerateForCharset(n int, chset Charset) string {
	buf := make([]byte, n)

	count := 0
	if chset&Uppercase != 0 {
		count += len_uppercase
	}
	if chset&Lowercase != 0 {
		count += len_lowercase
	}
	if chset&Numbers != 0 {
		count += len_numbers
	}
	if chset&Unreserved != 0 {
		count += len_unreserved
	}
	if chset&Reserved != 0 {
		count += len_reserved
	}
	if chset&SimpleSymbols != 0 {
		count += len_simple_symbols
	}
	if chset&Symbols != 0 {
		count += len_symbols
	}
	max := big.NewInt(int64(count))

	for i := 0; i < n; i++ {
		r, err := rand.Int(rand.Reader, max)
		if err != nil {
			panic(err)
		}
		idx := int(r.Int64())

		if chset&Uppercase != 0 {
			if idx < len_uppercase {
				buf[i] = uppercase[idx]
				continue
			} else {
				idx -= len_uppercase
			}
		}
		if chset&Lowercase != 0 {
			if idx < len_lowercase {
				buf[i] = lowercase[idx]
				continue
			} else {
				idx -= len_lowercase
			}
		}
		if chset&Numbers != 0 {
			if idx < len_numbers {
				buf[i] = numbers[idx]
				continue
			} else {
				idx -= len_numbers
			}
		}
		if chset&Unreserved != 0 {
			if idx < len_unreserved {
				buf[i] = unreserved[idx]
				continue
			} else {
				idx -= len_unreserved
			}
		}
		if chset&Reserved != 0 {
			if idx < len_reserved {
				buf[i] = reserved[idx]
				continue
			} else {
				idx -= len_reserved
			}
		}
		if chset&SimpleSymbols != 0 {
			if idx < len_simple_symbols {
				buf[i] = simple_symbols[idx]
				continue
			} else {
				idx -= len_simple_symbols
			}
		}
		if chset&Symbols != 0 {
			if idx < len_symbols {
				buf[i] = symbols[idx]
				continue
			} else {
				idx -= len_symbols
			}
		}
	}
	return string(buf)
}
