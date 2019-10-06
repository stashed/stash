package os

import "os"

func LookupEnv(key string, alts ...string) (string, bool) {
	if val, ok := os.LookupEnv(key); ok {
		return val, true
	}
	for _, alt := range alts {
		if val, ok := os.LookupEnv(alt); ok {
			return val, true
		}
	}
	return "", false
}

func Getenv(key string, alts ...string) string {
	val, _ := LookupEnv(key)
	return val
}
