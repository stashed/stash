/*
Copyright 2019 AppsCode Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package api

import (
	"gopkg.in/square/go-jose.v2/jwt"
)

// LicenseVerificationParams represents the license token for verification
type LicenseVerificationParams struct {
	Raw string `json:"raw"`
}

// License represents the product license for user
type License struct {
	Issuer           string           `json:"issuer,omitempty"`
	Subject          string           `json:"subject,omitempty"`
	Audience         jwt.Audience     `json:"audience,omitempty"`
	Expiry           *jwt.NumericDate `json:"expiry,omitempty"`
	NotBefore        *jwt.NumericDate `json:"not_before,omitempty"`
	IssuedAt         *jwt.NumericDate `json:"issued_at,omitempty"`
	ID               string           `json:"id,omitempty"`
	SubscribedPlans  []SubscribedPlan `json:"subscribed_plans"`
	SubscriptionID   string           `json:"subscription_id"`
	SubscriptionName string           `json:"subscription_name"`
	JWT              string           `json:"jwt"`
	Status           string           `json:"status"`
	CanceledAt       *int64           `json:"canceled_at"`
	IpAddress        *string          `json:"ip_address"`
	CancelerID       *string          `json:"canceler_id"`
}

// SubscribedPlan represents included plans in the license
type SubscribedPlan struct {
	PlanID    string `json:"plan"`
	ProductID string `json:"product"`
	OwnerID   int64  `json:"owner"`
}
