package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "scip-apex",
	Short: "SCIP indexer for Apex source code",
	Long:  "Generates SCIP (Sourcegraph Code Intelligence Protocol) index files from Apex source code.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
