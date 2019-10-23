package cli

import (
	"context"
	"strings"
	"time"

	"kmodules.xyz/client-go/tools/analytics"

	"github.com/appscode/go/log/golog"
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
	LoggerOptions     golog.Options
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
