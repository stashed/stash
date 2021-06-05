/*
Copyright AppsCode Inc. and Contributors

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

package cli

import (
	"context"
	"strings"
	"time"

	"kmodules.xyz/client-go/tools/analytics"

	ga "github.com/jpillora/go-ogle-analytics"
	"github.com/spf13/cobra"
)

const (
	gaTrackingCode     = "UA-62096468-20"
	gaTrackingInterval = 1 * time.Hour
)

var (
	AnalyticsClientID = analytics.ClientID()
	EnableAnalytics   = true
)

func SendAnalytics(c *cobra.Command, version string) {
	if !EnableAnalytics {
		return
	}
	if client, err := ga.NewClient(gaTrackingCode); err == nil {
		client.ClientID(AnalyticsClientID)
		parts := strings.Split(c.CommandPath(), " ")
		_ = client.Send(ga.NewEvent(parts[0], strings.Join(parts[1:], "/")).Label(version))
	}
}

func SendPeriodicAnalytics(c *cobra.Command, version string) context.CancelFunc {
	if !EnableAnalytics {
		return func() {}
	}
	ticker := time.NewTicker(gaTrackingInterval)
	go func() {
		for range ticker.C {
			SendAnalytics(c, version)
		}
	}()
	return ticker.Stop
}
