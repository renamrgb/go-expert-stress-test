// Package cmd defines the CLI surface of the application using Cobra.
package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/renamrgb/go-expert-stress-test/internal/loadtest"
)

const defaultRequestTimeout = 30 * time.Second

var (
	url         string
	requests    int
	concurrency int
)

// rootCmd is the single command of this CLI: it runs the load test itself,
// so `go-expert-stress-test --url=... --requests=... --concurrency=...`
// works with no subcommand.
var rootCmd = &cobra.Command{
	Use:   "go-expert-stress-test",
	Short: "CLI load testing tool for web services",
	Long: "go-expert-stress-test fires a configurable number of concurrent GET requests\n" +
		"at a target URL and prints a report with execution time and the distribution\n" +
		"of HTTP status codes received.",
	Example: "  go-expert-stress-test --url=http://google.com --requests=1000 --concurrency=10",
	RunE:    runLoadTest,
	// Errors are reported once by main.go; avoid Cobra's own duplicate
	// "Error: ..." print. Input is validated by loadtest.Config.Validate,
	// so no usage dump is needed on business-logic errors either.
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.Flags().StringVar(&url, "url", "", "URL of the service to be tested (required)")
	rootCmd.Flags().IntVar(&requests, "requests", 0, "Total number of requests to perform (required, > 0)")
	rootCmd.Flags().IntVar(&concurrency, "concurrency", 1, "Number of simultaneous requests")

	_ = rootCmd.MarkFlagRequired("url")
	_ = rootCmd.MarkFlagRequired("requests")
}

func runLoadTest(cmd *cobra.Command, _ []string) error {
	cfg := loadtest.Config{
		URL:         url,
		Requests:    requests,
		Concurrency: concurrency,
		Timeout:     defaultRequestTimeout,
	}

	if err := cfg.Validate(); err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Starting load test: url=%s requests=%d concurrency=%d\n\n", cfg.URL, cfg.Requests, cfg.Concurrency)

	rpt, err := loadtest.Run(cfg)
	if err != nil {
		return err
	}

	rpt.Print(out)

	return nil
}

// Execute runs the root command. It is the sole entry point called from
// main.go.
func Execute() error {
	return rootCmd.Execute()
}
