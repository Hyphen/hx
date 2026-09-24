package appcloud

import (
	"fmt"

	"github.com/Hyphen/cli/internal/appcloud"
	"github.com/Hyphen/cli/pkg/cprint"
	"github.com/Hyphen/cli/pkg/flags"
	"github.com/spf13/cobra"
)

var metricsCmd = &cobra.Command{
	Use:   "metrics",
	Short: "Query gateway telemetry (HTTP traffic and error logs)",
}

var (
	metricsApp    string
	metricsDomain string
	metricsMethod string
	metricsKind   string
	metricsStatus int
	metricsFrom   string
	metricsTo     string
	metricsLimit  int
)

var metricsHTTPCmd = &cobra.Command{
	Use:   "http",
	Short: "Gateway HTTP traffic logs (one row per served request)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		printer := cprint.NewCPrinter(flags.VerboseFlag)
		rows, err := newAppCloudService().QueryHTTPMetrics(appcloud.HTTPMetricsParams{
			AppID:  metricsApp,
			Domain: metricsDomain,
			Method: metricsMethod,
			Status: metricsStatus,
			From:   metricsFrom,
			To:     metricsTo,
			Limit:  metricsLimit,
		})
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			printer.Info("No HTTP logs found.")
			return nil
		}
		for _, r := range rows {
			cache := "miss"
			if r.CacheHit {
				cache = "hit"
			}
			printer.Print(fmt.Sprintf("%s  %3d  %-6s %s%s  %dms  %dB  [%s]",
				r.TS, r.Status, r.Method, r.Domain, r.Path, r.DurationMS, r.Bytes, cache))
		}
		return nil
	},
}

var metricsErrorsCmd = &cobra.Command{
	Use:   "errors",
	Short: "Gateway error logs (failures attributable to a domain)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		printer := cprint.NewCPrinter(flags.VerboseFlag)
		rows, err := newAppCloudService().QueryErrorMetrics(appcloud.ErrorMetricsParams{
			AppID:  metricsApp,
			Domain: metricsDomain,
			Kind:   metricsKind,
			From:   metricsFrom,
			To:     metricsTo,
			Limit:  metricsLimit,
		})
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			printer.Info("No error logs found.")
			return nil
		}
		for _, r := range rows {
			printer.Print(fmt.Sprintf("%s  %-16s %s  %s", r.TS, r.Kind, r.Domain, r.Message))
		}
		return nil
	},
}

func init() {
	for _, c := range []*cobra.Command{metricsHTTPCmd, metricsErrorsCmd} {
		c.Flags().StringVar(&metricsApp, "app", "", "Filter by app id")
		c.Flags().StringVar(&metricsDomain, "domain", "", "Filter by domain")
		c.Flags().StringVar(&metricsFrom, "from", "", "Start of the time window (RFC3339)")
		c.Flags().StringVar(&metricsTo, "to", "", "End of the time window (RFC3339)")
		c.Flags().IntVar(&metricsLimit, "limit", 0, "Max rows to return")
	}
	metricsHTTPCmd.Flags().StringVar(&metricsMethod, "method", "", "Filter by HTTP method")
	metricsHTTPCmd.Flags().IntVar(&metricsStatus, "status", 0, "Filter by exact status code")
	metricsErrorsCmd.Flags().StringVar(&metricsKind, "kind", "", "Filter by error kind")

	metricsCmd.AddCommand(metricsHTTPCmd)
	metricsCmd.AddCommand(metricsErrorsCmd)
}
