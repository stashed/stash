package rand

import (
	"crypto"
	"strings"

	pass "gomodules.xyz/password-generator"
)

// Use this for generating random pat of a ID. Do not use this for generating short passwords or secrets.
func Characters(len int) string {
	return pass.GenerateForCharset(len, pass.Lowercase|pass.Numbers)
}

func WithUniqSuffix(seed string) string {
	return seed + "-" + Characters(6)
}

func GenerateToken() string {
	return pass.Generate(32)
}

func GenerateTokenWithLength(len int) string {
	return pass.Generate(len)
}

func GeneratePassword() string {
	return pass.Generate(16)
}

func DigestForIndex(body string) string {
	s := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ._"

	body = strings.Trim(body, " \t\n\r")
	body = strings.TrimRight(body, "=")

	hash := crypto.SHA1.New()
	hash.Write([]byte(body))
	digest := hash.Sum(nil)
	var keyIndex string = ""
	for i := 0; i < 12; i++ {
		keyIndex += string(s[int(digest[i]&0x3F)])
	}

	return keyIndex
}
