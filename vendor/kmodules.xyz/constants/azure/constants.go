package azure

import (
	"log"
	"os"
)

const (
	AZURE_SUBSCRIPTION_ID = "AZURE_SUBSCRIPTION_ID"
	AZURE_TENANT_ID       = "AZURE_TENANT_ID"
	AZURE_CLIENT_ID       = "AZURE_CLIENT_ID"
	AZURE_CLIENT_SECRET   = "AZURE_CLIENT_SECRET"
	AZURE_ACCOUNT_NAME    = "AZURE_ACCOUNT_NAME"
	AZURE_ACCOUNT_KEY     = "AZURE_ACCOUNT_KEY"
)

const (
	AzureClientSecret   = "client-secret"
	AzureSubscriptionID = "subscription-id"
	AzureTenantID       = "tenant-id"
	AzureClientID       = "client-id"
)

func CredentialsFromEnv() map[string][]byte {
	subscriptionID := os.Getenv(AZURE_SUBSCRIPTION_ID)
	tenantID := os.Getenv(AZURE_TENANT_ID)
	clientID := os.Getenv(AZURE_CLIENT_ID)
	clientSecret := os.Getenv(AZURE_CLIENT_SECRET)
	if len(subscriptionID) == 0 || len(tenantID) == 0 || len(clientID) == 0 || len(clientSecret) == 0 {
		log.Println("Azure credentials for empty")
		return map[string][]byte{}
	}

	return map[string][]byte{
		AzureSubscriptionID: []byte(subscriptionID),
		AzureTenantID:       []byte(tenantID),
		AzureClientID:       []byte(clientID),
		AzureClientSecret:   []byte(clientSecret),
	}
}
