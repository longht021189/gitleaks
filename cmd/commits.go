package cmd

import (
	"github.com/spf13/cobra"
	"github.com/zricethezav/gitleaks/v8/flags"
)

var commitsCmd = &cobra.Command{
	Use:   "commits",
	Short: "Detect in Commit List",
	Run:   runDetectCommits,
}

func init() {
	detectCmd.AddCommand(commitsCmd)
	flags := flags.GetCommitsFlags()
	gitRequestCmd.Flags().StringVar(&flags.File, "commits-file", "", "Commits (--format=oneline) in File")
}

func runDetectCommits(cmd *cobra.Command, args []string) {
	runDetect(cmd, args)
}
