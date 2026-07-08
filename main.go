// Command go-expert-stress-test is a CLI load testing tool. It fires a
// configurable number of concurrent GET requests at a target URL and prints
// a report summarizing execution time and the distribution of HTTP status
// codes received.
package main

import (
	"fmt"
	"os"

	"github.com/renamrgb/go-expert-stress-test/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
