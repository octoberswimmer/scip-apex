package cmd

import (
	"fmt"
	"os"

	"github.com/octoberswimmer/aer/indexer"
	"github.com/spf13/cobra"
)

var (
	outputFile  string
	projectRoot string
	packageDir  string
	namespace   string
)

var indexCmd = &cobra.Command{
	Use:   "index [source-dirs...]",
	Short: "Index Apex source files and generate a SCIP index",
	Long:  "Walk .cls and .trigger files in the given directories, parse them, resolve symbols, and write a SCIP index file.",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		root := projectRoot
		if root == "" {
			var err error
			root, err = os.Getwd()
			if err != nil {
				return fmt.Errorf("getting working directory: %w", err)
			}
		}
		opts := indexer.Options{
			SourceDirs:  args,
			OutputFile:  outputFile,
			ProjectRoot: root,
			PackageDir:  packageDir,
			Namespace:   namespace,
		}
		return indexer.Run(opts)
	},
}

func init() {
	indexCmd.Flags().StringVarP(&outputFile, "output", "o", "index.scip", "Output file path")
	indexCmd.Flags().StringVar(&projectRoot, "project-root", "", "Project root for relative paths (default: cwd)")
	indexCmd.Flags().StringVar(&packageDir, "package-dir", "", "Directory with .pkg files to load")
	indexCmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Default namespace for loaded code")
	rootCmd.AddCommand(indexCmd)
}
