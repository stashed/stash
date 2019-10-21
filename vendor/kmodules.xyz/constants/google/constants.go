package google

import (
	"io/ioutil"
	"log"
	"os"
)

const (
	GOOGLE_PROJECT_ID               = "GOOGLE_PROJECT_ID"
	GOOGLE_SERVICE_ACCOUNT_JSON_KEY = "GOOGLE_SERVICE_ACCOUNT_JSON_KEY"
	GOOGLE_APPLICATION_CREDENTIALS  = "GOOGLE_APPLICATION_CREDENTIALS"
)

const (
	GCPSACredentialJson = "sa.json"
)

func ServiceAccountFromEnv() string {
	if data := os.Getenv(GOOGLE_SERVICE_ACCOUNT_JSON_KEY); len(data) > 0 {
		return data
	}
	if data, err := ioutil.ReadFile(os.Getenv(GOOGLE_APPLICATION_CREDENTIALS)); err == nil {
		return string(data)
	}
	log.Println("GOOGLE_SERVICE_ACCOUNT_JSON_KEY and GOOGLE_APPLICATION_CREDENTIALS are empty")
	return ""
}

func CredentialsFromEnv() map[string][]byte {
	sa := ServiceAccountFromEnv()
	if len(sa) == 0 {
		return map[string][]byte{}
	}
	return map[string][]byte{
		GCPSACredentialJson: []byte(sa),
	}
}
