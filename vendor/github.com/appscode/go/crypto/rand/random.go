package rand

import (
	"crypto"
	"crypto/rand"
	"encoding/base64"
	"strings"
)

// 32 chars as done in Phabricator Filesystem::readRandomCharacters
// This is NOT base32 encoding.
var randChars = []rune("abcdefghijklmnopqrstuvwxyz234567")

// Use this for generating random pat of a ID. Do not use this for generating short passwords or secrets.
func Characters(len int) string {
	bytes := make([]byte, len)
	rand.Read(bytes)
	r := make([]rune, len)
	for i, b := range bytes {
		r[i] = randChars[b>>3]
	}
	return string(r)
}

func WithUniqSuffix(seed string) string {
	return seed + "-" + Characters(6)
}

func GenerateToken() string {
	b := make([]byte, 256)
	rand.Read(b)
	temp := base64.RawURLEncoding.EncodeToString(b)
	return temp[0:32]
}

func GenerateTokenWithLength(len int) string {
	b := make([]byte, len)
	rand.Read(b)
	temp := base64.RawURLEncoding.EncodeToString(b)
	return temp[0:len]
}

func GeneratePassword() string {
	b := make([]byte, 128)
	rand.Read(b)
	temp := base64.RawURLEncoding.EncodeToString(b)
	return temp[0:16]
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
